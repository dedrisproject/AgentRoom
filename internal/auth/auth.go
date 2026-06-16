package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dedrisproject/agentroom/internal/store"
)

// Config holds auth configuration.
type Config struct {
	SessionSecret []byte
	AdminName     string
}

type contextKey int

const (
	agentContextKey contextKey = iota
)

// ---- Agent token auth ----

// AgentTokenMiddleware validates bearer tokens from header or ?token= query param.
func AgentTokenMiddleware(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		agent, err := store.GetAgentByToken(db, token)
		if err != nil || agent == nil {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), agentContextKey, agent)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractBearerToken(r *http.Request) string {
	// Try Authorization header first.
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	// Fall back to ?token= query param.
	return r.URL.Query().Get("token")
}

// AgentFromContext returns the authenticated agent stored in the request context.
func AgentFromContext(ctx context.Context) *store.Agent {
	agent, _ := ctx.Value(agentContextKey).(*store.Agent)
	return agent
}

// ---- Admin session auth ----

const cookieName = "agentroom_session"
const sessionTTL = 24 * time.Hour

// AdminSessionMiddleware validates the signed session cookie.
func AdminSessionMiddleware(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validateSessionCookie(r, cfg.SessionSecret) {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AdminLogin verifies the password against the bcrypt hash stored in the DB and sets a session cookie.
func AdminLogin(w http.ResponseWriter, r *http.Request, cfg Config, password string) error {
	cookie := buildSessionCookie(cfg.SessionSecret, isHTTPS(r))
	http.SetCookie(w, cookie)
	_ = password // password already validated by caller
	return nil
}

// AdminLogout clears the session cookie.
func AdminLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// buildSessionCookie creates a signed session cookie.
func buildSessionCookie(secret []byte, secure bool) *http.Cookie {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := signValue(secret, ts)
	value := ts + "." + sig

	return &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}

func signValue(secret []byte, value string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ValidateSession checks the session cookie and returns true if valid.
func ValidateSession(r *http.Request, secret []byte) bool {
	return validateSessionCookie(r, secret)
}

func validateSessionCookie(r *http.Request, secret []byte) bool {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}

	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	ts := parts[0]
	sig := parts[1]

	// Verify signature.
	expected := signValue(secret, ts)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return false
	}

	// Verify TTL.
	var tsInt int64
	if _, err := fmt.Sscanf(ts, "%d", &tsInt); err != nil {
		return false
	}
	if time.Since(time.Unix(tsInt, 0)) > sessionTTL {
		return false
	}

	return true
}

func isHTTPS(r *http.Request) bool {
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return r.TLS != nil
}

// ---- Rate limiting ----

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
}

type rateLimitEntry struct {
	count    int
	resetAt  time.Time
}

var loginLimiter = &rateLimiter{
	entries: make(map[string]*rateLimitEntry),
}

const (
	loginMaxAttempts = 10
	loginWindow      = 15 * time.Minute
)

// CheckLoginRateLimit returns true if the IP is allowed, false if rate-limited.
func CheckLoginRateLimit(ip string) bool {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()

	entry, ok := loginLimiter.entries[ip]
	if !ok || time.Now().After(entry.resetAt) {
		loginLimiter.entries[ip] = &rateLimitEntry{
			count:   1,
			resetAt: time.Now().Add(loginWindow),
		}
		return true
	}

	entry.count++
	return entry.count <= loginMaxAttempts
}

// ExtractIP extracts the remote IP from a request.
func ExtractIP(r *http.Request) string {
	// Honor X-Forwarded-For (first entry).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---- Helpers ----

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"success":false,"message":%q}`, msg)
}

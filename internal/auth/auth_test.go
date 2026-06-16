package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionCookieRoundTrip(t *testing.T) {
	secret := []byte("super-secret-key")

	cookie := buildSessionCookie(secret, false)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(cookie)

	if !validateSessionCookie(r, secret) {
		t.Fatal("freshly issued cookie should validate")
	}
	if validateSessionCookie(r, []byte("different-secret")) {
		t.Fatal("cookie should not validate under a different secret")
	}
}

func TestSessionCookieTampered(t *testing.T) {
	secret := []byte("super-secret-key")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: "1700000000.not-a-valid-signature"})
	if validateSessionCookie(r, secret) {
		t.Fatal("tampered signature must be rejected")
	}
}

func TestSessionCookieExpired(t *testing.T) {
	secret := []byte("super-secret-key")
	old := fmt.Sprintf("%d", time.Now().Add(-sessionTTL-time.Hour).Unix())
	value := old + "." + signValue(secret, old)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: cookieName, Value: value})
	if validateSessionCookie(r, secret) {
		t.Fatal("expired-but-correctly-signed cookie must be rejected")
	}
}

func TestLoginRateLimit(t *testing.T) {
	// Reset shared limiter state for a deterministic test.
	loginLimiter.mu.Lock()
	loginLimiter.entries = map[string]*rateLimitEntry{}
	loginLimiter.lastSweep = time.Time{}
	loginLimiter.mu.Unlock()

	ip := "203.0.113.7"
	for i := 0; i < loginMaxAttempts; i++ {
		if !CheckLoginRateLimit(ip) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if CheckLoginRateLimit(ip) {
		t.Fatal("attempt beyond the limit should be blocked")
	}
	if !CheckLoginRateLimit("198.51.100.2") {
		t.Fatal("a different IP should not be affected")
	}
}

func TestExtractIPForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-For", "9.9.9.9, 10.0.0.1")
	if got := ExtractIP(r); got != "9.9.9.9" {
		t.Fatalf("expected first XFF entry, got %q", got)
	}
}

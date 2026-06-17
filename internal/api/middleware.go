package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/dedrisproject/agentroom/internal/auth"
)

// maxRequestBody caps incoming request bodies to mitigate memory-exhaustion.
const maxRequestBody = 1 << 20 // 1 MiB

// statusRecorder captures the response status code for access logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	s.wrote = true
	return s.ResponseWriter.Write(b)
}

// withMiddleware wraps h with security headers, a request body-size limit,
// and structured access logging.
func withMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Conservative security headers (safe for an admin tool + JSON API).
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")

		// Bound the request body.
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		h.ServeHTTP(rec, r)

		// Log the path only (never the raw query) so agent tokens passed as
		// ?token= are not written to logs.
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"dur", time.Since(start).Round(time.Microsecond).String(),
			"ip", auth.ExtractIP(r),
		)
	})
}

// healthz is an unauthenticated health check that verifies DB connectivity.
func healthz(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if err := db.PingContext(r.Context()); err != nil {
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

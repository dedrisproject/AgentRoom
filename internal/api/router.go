package api

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/dedrisproject/agentroom/internal/auth"
)

// UIHandler is implemented by the web UI handler.
type UIHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// WebRegistrar is implemented by the web UI handler.
type WebRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// NewRouter wires all routes and returns the root handler.
// It uses a two-mux approach to avoid Go 1.22 ServeMux pattern conflicts between
// method-scoped UI patterns (GET /) and method-agnostic API prefix patterns (/api/).
func NewRouter(database *sql.DB, authCfg auth.Config, adminName string, ui WebRegistrar) http.Handler {
	agent := &agentHandlers{
		db:        database,
		adminName: adminName,
	}
	admin := &adminHandlers{
		database:  database,
		authCfg:   authCfg,
		adminName: adminName,
	}

	// ---- Agent API mux (token auth) ----
	agentMux := http.NewServeMux()
	agentMux.HandleFunc("GET /api/agent/me", agent.me)
	agentMux.HandleFunc("GET /api/agent/messages", agent.listMessages)
	agentMux.HandleFunc("POST /api/agent/messages", agent.createMessage)
	agentMux.HandleFunc("POST /api/agent/messages/{id}/reply", agent.replyToMessage)
	agentMux.HandleFunc("POST /api/agent/messages/{id}/close", agent.closeThread)
	agentHandler := auth.AgentTokenMiddleware(database, agentMux)

	// ---- Admin API mux (session auth) ----
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /api/admin/rooms", admin.listRooms)
	adminMux.HandleFunc("POST /api/admin/rooms", admin.createRoom)
	adminMux.HandleFunc("GET /api/admin/rooms/{id}", admin.getRoom)
	adminMux.HandleFunc("DELETE /api/admin/rooms/{id}", admin.deleteRoom)
	adminMux.HandleFunc("POST /api/admin/rooms/{id}/agents", admin.createAgent)
	adminMux.HandleFunc("POST /api/admin/rooms/{id}/messages", admin.createMessage)
	adminMux.HandleFunc("GET /api/admin/agents/{id}", admin.getAgent)
	adminMux.HandleFunc("PUT /api/admin/agents/{id}", admin.updateAgent)
	adminMux.HandleFunc("DELETE /api/admin/agents/{id}", admin.deleteAgent)
	adminMux.HandleFunc("GET /api/admin/agents/{id}/instructions", admin.getInstructions)
	adminMux.HandleFunc("POST /api/admin/messages/{id}/reply", admin.replyToMessage)
	adminMux.HandleFunc("POST /api/admin/messages/{id}/close", admin.closeThread)
	adminMux.HandleFunc("GET /api/admin/blockers", admin.listBlockers)
	adminHandler := auth.AdminSessionMiddleware(authCfg, adminMux)

	// ---- Unauthenticated admin endpoints ----
	unauthMux := http.NewServeMux()
	unauthMux.HandleFunc("POST /api/admin/login", admin.login)
	unauthMux.HandleFunc("POST /api/admin/logout", admin.logout)

	// ---- Static file mux ----
	var staticHandler http.Handler
	if ui != nil {
		staticMux := http.NewServeMux()
		ui.RegisterRoutes(staticMux)
		staticHandler = staticMux
	}

	// ---- Top-level dispatcher ----
	// Avoids Go 1.22 pattern conflicts by dispatching based on path prefix
	// rather than registering overlapping patterns on the same mux.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == "/api/admin/login" || path == "/api/admin/logout":
			unauthMux.ServeHTTP(w, r)

		case strings.HasPrefix(path, "/api/admin/"):
			adminHandler.ServeHTTP(w, r)

		case strings.HasPrefix(path, "/api/agent/"):
			agentHandler.ServeHTTP(w, r)

		case strings.HasPrefix(path, "/static/"):
			if staticHandler != nil {
				staticHandler.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}

		default:
			if ui != nil {
				ui.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		}
	})
}

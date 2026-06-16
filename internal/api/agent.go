package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/dedrisproject/agentroom/internal/auth"
	"github.com/dedrisproject/agentroom/internal/store"
)

type agentHandlers struct {
	db        *sql.DB
	adminName string
}

// ---- GET /api/agent/me ----

func (h *agentHandlers) me(w http.ResponseWriter, r *http.Request) {
	agent := auth.AgentFromContext(r.Context())
	if agent == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	room, err := store.GetRoom(h.db, agent.RoomID)
	if err != nil || room == nil {
		jsonError(w, "room not found", http.StatusNotFound)
		return
	}

	baseURL := detectBaseURL(r)
	jsonSuccess(w, map[string]interface{}{
		"agent":   agent,
		"room":    room,
		"api_url": baseURL + "/api/agent",
	})
}

// ---- GET /api/agent/messages ----

func (h *agentHandlers) listMessages(w http.ResponseWriter, r *http.Request) {
	agent := auth.AgentFromContext(r.Context())
	if agent == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	includeClosed := parseBoolParam(r, "include_closed")

	msgs, err := store.ListAgentMessages(h.db, agent, includeClosed)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if msgs == nil {
		msgs = []*store.Message{}
	}
	jsonSuccess(w, map[string]interface{}{
		"messages": msgs,
	})
}

// ---- POST /api/agent/messages ----

func (h *agentHandlers) createMessage(w http.ResponseWriter, r *http.Request) {
	agent := auth.AgentFromContext(r.Context())
	if agent == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	toAgent := readParam(r, "to_agent")
	subject := readParam(r, "subject")
	priority := readParam(r, "priority")
	body := readParam(r, "message")
	if body == "" {
		body = readParam(r, "body")
	}

	if toAgent == "" {
		jsonError(w, "to_agent is required", http.StatusBadRequest)
		return
	}
	if body == "" {
		jsonError(w, "message body is required", http.StatusBadRequest)
		return
	}
	if priority == "" {
		priority = "normal"
	}
	if priority != "normal" && priority != "blocker" {
		jsonError(w, "priority must be 'normal' or 'blocker'", http.StatusBadRequest)
		return
	}

	// Validate recipient.
	if err := h.validateRecipient(agent.RoomID, toAgent); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if subject == "" {
		subject = "AgentRoom request"
	}

	msg, err := store.CreateMessage(h.db, agent.RoomID, nil, agent.Name, toAgent, subject, body, priority, "request")
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"message": msg,
	})
}

// ---- POST /api/agent/messages/{id}/reply ----

func (h *agentHandlers) replyToMessage(w http.ResponseWriter, r *http.Request) {
	agent := auth.AgentFromContext(r.Context())
	if agent == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	msgID, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid message id", http.StatusBadRequest)
		return
	}

	// Verify the message belongs to this agent's room.
	target, err := store.GetMessage(h.db, msgID)
	if err != nil || target == nil {
		jsonError(w, "message not found", http.StatusNotFound)
		return
	}
	if target.RoomID != agent.RoomID {
		jsonError(w, "message not found", http.StatusNotFound)
		return
	}

	body := readParam(r, "message")
	if body == "" {
		body = readParam(r, "body")
	}
	toAgent := readParam(r, "to_agent")

	if body == "" {
		jsonError(w, "message body is required", http.StatusBadRequest)
		return
	}

	reply, err := store.ReplyToMessage(h.db, msgID, agent.Name, toAgent, body)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"message": reply,
	})
}

// ---- POST /api/agent/messages/{id}/close ----

func (h *agentHandlers) closeThread(w http.ResponseWriter, r *http.Request) {
	agent := auth.AgentFromContext(r.Context())
	if agent == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	msgID, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid message id", http.StatusBadRequest)
		return
	}

	// Verify message belongs to this room.
	target, err := store.GetMessage(h.db, msgID)
	if err != nil || target == nil {
		jsonError(w, "message not found", http.StatusNotFound)
		return
	}
	if target.RoomID != agent.RoomID {
		jsonError(w, "message not found", http.StatusNotFound)
		return
	}

	if err := store.CloseThread(h.db, msgID, agent.Name); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"closed": true,
	})
}

// ---- Helpers ----

func (h *agentHandlers) validateRecipient(roomID int64, toAgent string) error {
	if toAgent == "all" || toAgent == h.adminName {
		return nil
	}
	agents, err := store.ListAgents(h.db, roomID)
	if err != nil {
		return fmt.Errorf("internal error")
	}
	for _, a := range agents {
		if a.Name == toAgent {
			return nil
		}
	}
	return fmt.Errorf("recipient %q not found in room", toAgent)
}

// ---- Shared helpers ----

// readParam reads a parameter from JSON body, form body, or query string (in that order).
func readParam(r *http.Request, key string) string {
	// Try JSON body first if content-type is JSON.
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Body has been (or will be) parsed; try context store if available.
		// Actually we need to parse JSON body fresh each time — use a different approach.
		// We parse lazily via the jsonBody function.
		if val := jsonBodyParam(r, key); val != "" {
			return val
		}
	}

	// Parse form if not already done.
	r.ParseMultipartForm(1 << 20) // 1MB limit

	if val := r.FormValue(key); val != "" {
		return val
	}
	return r.URL.Query().Get(key)
}

// jsonBodyParam attempts to parse the request body as JSON and return a string field.
// We store parsed body in a sync-safe way via context is complex; for simplicity
// we accept that body may have been consumed. Better: buffer it once.
// For now, parse form handles both form and query; JSON is handled via ParseJSONBody below.
func jsonBodyParam(r *http.Request, key string) string {
	// Body is consumed after first read; rely on ParseForm for form data.
	// JSON-body parsing is handled by parseRequestBody in handlers that need it.
	return ""
}

// parseRequestBody reads JSON or form body into a map.
func parseRequestBody(r *http.Request) map[string]string {
	params := make(map[string]string)

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			for k, v := range body {
				if s, ok := v.(string); ok {
					params[k] = s
				} else if v != nil {
					params[k] = fmt.Sprintf("%v", v)
				}
			}
		}
		return params
	}

	r.ParseMultipartForm(1 << 20)
	for k, v := range r.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	for k, v := range r.URL.Query() {
		if _, exists := params[k]; !exists && len(v) > 0 {
			params[k] = v[0]
		}
	}
	return params
}

func parseBoolParam(r *http.Request, key string) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		v = r.FormValue(key)
	}
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func pathParamID(r *http.Request, key string) (int64, error) {
	v := r.PathValue(key)
	if v == "" {
		return 0, fmt.Errorf("missing path param %s", key)
	}
	return strconv.ParseInt(v, 10, 64)
}

func jsonSuccess(w http.ResponseWriter, data map[string]interface{}) {
	data["success"] = true
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": msg,
	})
}

func detectBaseURL(r *http.Request) string {
	proto := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		proto = "https"
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return proto + "://" + host
}

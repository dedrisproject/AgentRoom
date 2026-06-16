package api

import (
	"database/sql"
	"net/http"

	"github.com/dedrisproject/agentroom/internal/auth"
	"github.com/dedrisproject/agentroom/internal/db"
	"github.com/dedrisproject/agentroom/internal/instructions"
	"github.com/dedrisproject/agentroom/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type adminHandlers struct {
	database  *sql.DB
	authCfg   auth.Config
	adminName string
}

// ---- POST /api/admin/login ----

func (h *adminHandlers) login(w http.ResponseWriter, r *http.Request) {
	ip := auth.ExtractIP(r)
	if !auth.CheckLoginRateLimit(ip) {
		jsonError(w, "too many attempts", http.StatusTooManyRequests)
		return
	}

	params := parseRequestBody(r)
	password := params["password"]

	hash, ok := db.GetSetting(h.database, "admin_password_hash")
	if !ok || hash == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := auth.AdminLogin(w, r, h.authCfg, password); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"logged_in": true,
	})
}

// ---- POST /api/admin/logout ----

func (h *adminHandlers) logout(w http.ResponseWriter, r *http.Request) {
	auth.AdminLogout(w, r)
	jsonSuccess(w, map[string]interface{}{
		"logged_out": true,
	})
}

// ---- GET /api/admin/rooms ----

func (h *adminHandlers) listRooms(w http.ResponseWriter, r *http.Request) {
	summaries, err := store.ListRooms(h.database)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if summaries == nil {
		summaries = []store.RoomSummary{}
	}
	jsonSuccess(w, map[string]interface{}{
		"rooms": summaries,
	})
}

// ---- POST /api/admin/rooms ----

func (h *adminHandlers) createRoom(w http.ResponseWriter, r *http.Request) {
	params := parseRequestBody(r)
	name := params["name"]
	if name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	room, err := store.CreateRoom(h.database, name)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonSuccess(w, map[string]interface{}{
		"room": room,
	})
}

// ---- GET /api/admin/rooms/{id} ----

func (h *adminHandlers) getRoom(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid room id", http.StatusBadRequest)
		return
	}

	room, err := store.GetRoom(h.database, id)
	if err != nil || room == nil {
		jsonError(w, "room not found", http.StatusNotFound)
		return
	}

	agents, err := store.ListAgents(h.database, id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if agents == nil {
		agents = []*store.Agent{}
	}

	msgs, err := store.GetRoomMessages(h.database, id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if msgs == nil {
		msgs = []*store.Message{}
	}

	jsonSuccess(w, map[string]interface{}{
		"room":     room,
		"agents":   agents,
		"messages": msgs,
	})
}

// ---- DELETE /api/admin/rooms/{id} ----

func (h *adminHandlers) deleteRoom(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid room id", http.StatusBadRequest)
		return
	}

	room, err := store.GetRoom(h.database, id)
	if err != nil || room == nil {
		jsonError(w, "room not found", http.StatusNotFound)
		return
	}

	if err := store.DeleteRoom(h.database, id); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"deleted": true,
	})
}

// ---- POST /api/admin/rooms/{id}/agents ----

func (h *adminHandlers) createAgent(w http.ResponseWriter, r *http.Request) {
	roomID, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid room id", http.StatusBadRequest)
		return
	}

	room, err := store.GetRoom(h.database, roomID)
	if err != nil || room == nil {
		jsonError(w, "room not found", http.StatusNotFound)
		return
	}

	params := parseRequestBody(r)
	name := params["name"]
	role := params["role"]
	repo := params["repo"]

	if name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	agent, err := store.CreateOrUpdateAgent(h.database, roomID, name, role, repo)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	baseURL := detectBaseURL(r)
	md := instructions.Generate(agent, room, baseURL, h.adminName)

	// The access token is intentionally included on create so the admin can hand it to the agent.
	w.WriteHeader(http.StatusCreated)
	jsonSuccess(w, map[string]interface{}{
		"agent":        agent,
		"instructions": md,
	})
}

// ---- GET /api/admin/agents/{id} ----

func (h *adminHandlers) getAgent(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid agent id", http.StatusBadRequest)
		return
	}

	agent, err := store.GetAgent(h.database, id)
	if err != nil || agent == nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"agent": agent,
	})
}

// ---- PUT /api/admin/agents/{id} ----

func (h *adminHandlers) updateAgent(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid agent id", http.StatusBadRequest)
		return
	}

	params := parseRequestBody(r)
	name := params["name"]
	role := params["role"]
	repo := params["repo"]

	if name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	agent, err := store.UpdateAgent(h.database, id, name, role, repo)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if agent == nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"agent": agent,
	})
}

// ---- DELETE /api/admin/agents/{id} ----

func (h *adminHandlers) deleteAgent(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid agent id", http.StatusBadRequest)
		return
	}

	agent, err := store.GetAgent(h.database, id)
	if err != nil || agent == nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	if err := store.SoftDeleteAgent(h.database, id); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"deleted": true,
	})
}

// ---- GET /api/admin/agents/{id}/instructions ----

func (h *adminHandlers) getInstructions(w http.ResponseWriter, r *http.Request) {
	id, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid agent id", http.StatusBadRequest)
		return
	}

	agent, err := store.GetAgent(h.database, id)
	if err != nil || agent == nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}

	room, err := store.GetRoom(h.database, agent.RoomID)
	if err != nil || room == nil {
		jsonError(w, "room not found", http.StatusNotFound)
		return
	}

	baseURL := detectBaseURL(r)
	md := instructions.Generate(agent, room, baseURL, h.adminName)

	jsonSuccess(w, map[string]interface{}{
		"instructions": md,
		"agent":        agent,
	})
}

// ---- POST /api/admin/rooms/{id}/messages ----

func (h *adminHandlers) createMessage(w http.ResponseWriter, r *http.Request) {
	roomID, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid room id", http.StatusBadRequest)
		return
	}

	room, err := store.GetRoom(h.database, roomID)
	if err != nil || room == nil {
		jsonError(w, "room not found", http.StatusNotFound)
		return
	}

	params := parseRequestBody(r)
	toAgent := params["to_agent"]
	subject := params["subject"]
	priority := params["priority"]
	body := params["message"]
	if body == "" {
		body = params["body"]
	}
	fromAgent := params["from_agent"]
	if fromAgent == "" {
		fromAgent = h.adminName
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
	if subject == "" {
		subject = "AgentRoom request"
	}

	msg, err := store.CreateMessage(h.database, roomID, nil, fromAgent, toAgent, subject, body, priority, "request")
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonSuccess(w, map[string]interface{}{
		"message": msg,
	})
}

// ---- POST /api/admin/messages/{id}/reply ----

func (h *adminHandlers) replyToMessage(w http.ResponseWriter, r *http.Request) {
	msgID, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid message id", http.StatusBadRequest)
		return
	}

	target, err := store.GetMessage(h.database, msgID)
	if err != nil || target == nil {
		jsonError(w, "message not found", http.StatusNotFound)
		return
	}

	params := parseRequestBody(r)
	body := params["message"]
	if body == "" {
		body = params["body"]
	}
	toAgent := params["to_agent"]
	fromAgent := params["from_agent"]
	if fromAgent == "" {
		fromAgent = h.adminName
	}

	if body == "" {
		jsonError(w, "message body is required", http.StatusBadRequest)
		return
	}

	reply, err := store.ReplyToMessage(h.database, msgID, fromAgent, toAgent, body)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"message": reply,
	})
}

// ---- POST /api/admin/messages/{id}/close ----

func (h *adminHandlers) closeThread(w http.ResponseWriter, r *http.Request) {
	msgID, err := pathParamID(r, "id")
	if err != nil {
		jsonError(w, "invalid message id", http.StatusBadRequest)
		return
	}

	target, err := store.GetMessage(h.database, msgID)
	if err != nil || target == nil {
		jsonError(w, "message not found", http.StatusNotFound)
		return
	}

	params := parseRequestBody(r)
	closedBy := params["closed_by"]
	if closedBy == "" {
		closedBy = h.adminName
	}

	if err := store.CloseThread(h.database, msgID, closedBy); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonSuccess(w, map[string]interface{}{
		"closed": true,
	})
}

// ---- GET /api/admin/blockers ----

func (h *adminHandlers) listBlockers(w http.ResponseWriter, r *http.Request) {
	blockers, err := store.ListGlobalBlockers(h.database)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if blockers == nil {
		blockers = []*store.Message{}
	}
	jsonSuccess(w, map[string]interface{}{
		"blockers": blockers,
	})
}

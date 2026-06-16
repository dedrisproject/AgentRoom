package api_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dedrisproject/agentroom/internal/api"
	"github.com/dedrisproject/agentroom/internal/auth"
	appdb "github.com/dedrisproject/agentroom/internal/db"
	"github.com/dedrisproject/agentroom/internal/store"
	"golang.org/x/crypto/bcrypt"
)

const testAdminPassword = "correct horse"

func newTestServer(t *testing.T) (*sql.DB, http.Handler) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	database, err := appdb.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := appdb.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	hash, _ := bcrypt.GenerateFromPassword([]byte(testAdminPassword), bcrypt.DefaultCost)
	if err := appdb.SetSetting(database, "admin_password_hash", string(hash)); err != nil {
		t.Fatalf("seed password: %v", err)
	}

	authCfg := auth.Config{SessionSecret: []byte("test-session-secret"), AdminName: "admin"}
	handler := api.NewRouter(database, authCfg, "admin", nil)
	return database, handler
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

// TestAgentCreateMessageJSON guards the regression where the agent API ignored
// JSON request bodies (parseRequestBody must handle application/json).
func TestAgentCreateMessageJSON(t *testing.T) {
	database, handler := newTestServer(t)
	room, _ := store.CreateRoom(database, "R")
	a, _ := store.CreateOrUpdateAgent(database, room.ID, "A", "", "")
	store.CreateOrUpdateAgent(database, room.ID, "B", "", "")

	body := `{"to_agent":"B","subject":"via json","message":"hello","priority":"blocker"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages?token="+a.AccessToken, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for JSON body, got %d: %s", rec.Code, rec.Body.String())
	}
	if out := decode(t, rec); out["success"] != true {
		t.Fatalf("expected success, got %v", out)
	}
}

func TestAgentCreateMessageForm(t *testing.T) {
	database, handler := newTestServer(t)
	room, _ := store.CreateRoom(database, "R")
	a, _ := store.CreateOrUpdateAgent(database, room.ID, "A", "", "")
	store.CreateOrUpdateAgent(database, room.ID, "B", "", "")

	form := "to_agent=B&message=hello+via+form&priority=normal"
	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages?token="+a.AccessToken, strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for form body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAgentUnknownRecipientRejected(t *testing.T) {
	database, handler := newTestServer(t)
	room, _ := store.CreateRoom(database, "R")
	a, _ := store.CreateOrUpdateAgent(database, room.ID, "A", "", "")

	body := `{"to_agent":"ghost","message":"hi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/agent/messages?token="+a.AccessToken, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown recipient, got %d", rec.Code)
	}
}

func TestAgentMissingTokenUnauthorized(t *testing.T) {
	_, handler := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/agent/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestAgentInboxRoundTrip(t *testing.T) {
	database, handler := newTestServer(t)
	room, _ := store.CreateRoom(database, "R")
	a, _ := store.CreateOrUpdateAgent(database, room.ID, "A", "", "")
	b, _ := store.CreateOrUpdateAgent(database, room.ID, "B", "", "")

	// A messages B directly via store, then B fetches the inbox through the API.
	store.CreateMessage(database, room.ID, nil, "A", "B", "subj", "body", "normal", "request")

	req := httptest.NewRequest(http.MethodGet, "/api/agent/messages?token="+b.AccessToken, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	out := decode(t, rec)
	msgs, ok := out["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected 1 message in B's inbox, got %v", out["messages"])
	}
	_ = a
}

func TestAdminLogin(t *testing.T) {
	_, handler := newTestServer(t)

	// Wrong password.
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader("password=wrong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rec.Code)
	}

	// Correct password sets a session cookie.
	req = httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader("password="+testAdminPassword))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for correct password, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Fatal("expected a session cookie to be set on login")
	}
}

func TestAdminRequiresSession(t *testing.T) {
	_, handler := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/rooms", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d", rec.Code)
	}
}

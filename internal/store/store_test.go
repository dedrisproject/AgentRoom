package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	appdb "github.com/dedrisproject/agentroom/internal/db"
)

func newTestDB(t *testing.T) *sql.DB {
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
	return database
}

func TestRoomLifecycle(t *testing.T) {
	db := newTestDB(t)

	room, err := CreateRoom(db, "Demo")
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if room.ID == 0 || room.Name != "Demo" {
		t.Fatalf("unexpected room: %+v", room)
	}

	got, err := GetRoom(db, room.ID)
	if err != nil || got == nil {
		t.Fatalf("get room: %v (got %v)", err, got)
	}

	summaries, err := ListRooms(db)
	if err != nil {
		t.Fatalf("list rooms: %v", err)
	}
	if len(summaries) != 1 || summaries[0].AgentsCount != 0 || summaries[0].BlockersCount != 0 {
		t.Fatalf("unexpected summaries: %+v", summaries)
	}

	if err := DeleteRoom(db, room.ID); err != nil {
		t.Fatalf("delete room: %v", err)
	}
	got, _ = GetRoom(db, room.ID)
	if got != nil {
		t.Fatalf("expected room deleted, got %+v", got)
	}
}

func TestGetRoomNotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := GetRoom(db, 4242)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing room, got %+v", got)
	}
}

func TestAgentUpsertAndSoftDelete(t *testing.T) {
	db := newTestDB(t)
	room, _ := CreateRoom(db, "R")

	a1, err := CreateOrUpdateAgent(db, room.ID, "backend", "Backend", "repo-a")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if a1.AccessToken == "" {
		t.Fatal("expected a token on create")
	}

	// Same name => upsert (same row, fresh token, updated metadata).
	a2, err := CreateOrUpdateAgent(db, room.ID, "backend", "Backend v2", "repo-b")
	if err != nil {
		t.Fatalf("upsert agent: %v", err)
	}
	if a2.ID != a1.ID {
		t.Fatalf("expected same id on upsert, got %d vs %d", a2.ID, a1.ID)
	}
	if a2.AccessToken == a1.AccessToken {
		t.Fatal("expected a fresh token on upsert")
	}
	if a2.Role != "Backend v2" || a2.Repo != "repo-b" {
		t.Fatalf("expected metadata updated, got %+v", a2)
	}

	// Token resolves to the active agent.
	byTok, err := GetAgentByToken(db, a2.AccessToken)
	if err != nil || byTok == nil {
		t.Fatalf("get by token: %v (got %v)", err, byTok)
	}

	// Soft delete deactivates the token.
	if err := SoftDeleteAgent(db, a1.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	byTok, _ = GetAgentByToken(db, a2.AccessToken)
	if byTok != nil {
		t.Fatal("expected token to stop working after soft delete")
	}

	// Re-adding the same name reactivates it.
	a3, _ := CreateOrUpdateAgent(db, room.ID, "backend", "", "")
	if a3.ID != a1.ID || !a3.Active {
		t.Fatalf("expected reactivation of same row, got %+v", a3)
	}
}

func TestUpdateAgentDuplicateName(t *testing.T) {
	db := newTestDB(t)
	room, _ := CreateRoom(db, "R")
	CreateOrUpdateAgent(db, room.ID, "a", "", "")
	b, _ := CreateOrUpdateAgent(db, room.ID, "b", "", "")

	_, err := UpdateAgent(db, b.ID, "a", "", "")
	if err == nil {
		t.Fatal("expected error renaming to an existing name")
	}
}

func TestRelevanceThreadingAndClose(t *testing.T) {
	db := newTestDB(t)
	room, _ := CreateRoom(db, "R")
	_, _ = CreateOrUpdateAgent(db, room.ID, "A", "", "")
	b, _ := CreateOrUpdateAgent(db, room.ID, "B", "", "")
	c, _ := CreateOrUpdateAgent(db, room.ID, "C", "", "")

	// A -> B blocker request.
	m, err := CreateMessage(db, room.ID, nil, "A", "B", "Need change", "details", "blocker", "request")
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	// B sees it; C does not.
	bInbox, _ := ListAgentMessages(db, b, false)
	if !containsMsg(bInbox, m.ID) {
		t.Fatal("B should see a message addressed to B")
	}
	cInbox, _ := ListAgentMessages(db, c, false)
	if containsMsg(cInbox, m.ID) {
		t.Fatal("C should NOT see a message addressed to B")
	}

	// Broadcast to all is seen by C.
	bc, _ := CreateMessage(db, room.ID, nil, "A", "all", "Heads up", "fyi", "normal", "request")
	cInbox, _ = ListAgentMessages(db, c, false)
	if !containsMsg(cInbox, bc.ID) {
		t.Fatal("C should see a broadcast to 'all'")
	}

	// Reply from B attaches to root, infers recipient A, inherits blocker priority, "Re:" subject.
	reply, err := ReplyToMessage(db, m.ID, "B", "", "done")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if reply.ParentID == nil || *reply.ParentID != m.ID {
		t.Fatalf("reply should attach to root %d, got parent %v", m.ID, reply.ParentID)
	}
	if reply.ToAgent != "A" {
		t.Fatalf("reply recipient should be inferred as A, got %q", reply.ToAgent)
	}
	if reply.Priority != "blocker" {
		t.Fatalf("reply should inherit blocker priority, got %q", reply.Priority)
	}
	if reply.Subject != "Re: Need change" {
		t.Fatalf("unexpected reply subject %q", reply.Subject)
	}

	// One open blocker exists globally.
	blockers, _ := ListGlobalBlockers(db)
	if len(blockers) != 1 || blockers[0].ID != m.ID {
		t.Fatalf("expected 1 global blocker (the root), got %+v", blockers)
	}

	// Closing the thread closes root + replies.
	if err := CloseThread(db, reply.ID, "admin"); err != nil {
		t.Fatalf("close thread: %v", err)
	}
	blockers, _ = ListGlobalBlockers(db)
	if len(blockers) != 0 {
		t.Fatalf("expected 0 open blockers after close, got %+v", blockers)
	}
	// Default inbox (open only) no longer shows the closed thread for B...
	bInbox, _ = ListAgentMessages(db, b, false)
	if containsMsg(bInbox, m.ID) {
		t.Fatal("closed thread should be hidden from default inbox")
	}
	// ...but include_closed surfaces it again.
	bInbox, _ = ListAgentMessages(db, b, true)
	if !containsMsg(bInbox, m.ID) {
		t.Fatal("closed thread should appear with include_closed=true")
	}
}

func containsMsg(msgs []*Message, id int64) bool {
	for _, m := range msgs {
		if m.ID == id {
			return true
		}
	}
	return false
}

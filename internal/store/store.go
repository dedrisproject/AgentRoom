package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ---- Types ----

type Room struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type RoomSummary struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	CreatedAt     string `json:"created_at"`
	AgentsCount   int    `json:"agents_count"`
	BlockersCount int    `json:"blockers_count"`
}

type Agent struct {
	ID          int64  `json:"id"`
	RoomID      int64  `json:"room_id"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Repo        string `json:"repo"`
	AccessToken string `json:"access_token,omitempty"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Message struct {
	ID        int64   `json:"id"`
	RoomID    int64   `json:"room_id"`
	ParentID  *int64  `json:"parent_id,omitempty"`
	FromAgent string  `json:"from_agent"`
	ToAgent   string  `json:"to_agent"`
	Subject   string  `json:"subject,omitempty"`
	Body      string  `json:"body"`
	Priority  string  `json:"priority"`
	Status    string  `json:"status"`
	Type      string  `json:"type"`
	ReadAt    *string `json:"read_at,omitempty"`
	CreatedAt string  `json:"created_at"`
	ClosedAt  *string `json:"closed_at,omitempty"`
	ClosedBy  *string `json:"closed_by,omitempty"`
}

// ---- Rooms ----

func CreateRoom(db *sql.DB, name string) (*Room, error) {
	res, err := db.Exec(`INSERT INTO rooms(name) VALUES(?)`, name)
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return GetRoom(db, id)
}

func GetRoom(db *sql.DB, id int64) (*Room, error) {
	r := &Room{}
	err := db.QueryRow(`SELECT id, name, created_at FROM rooms WHERE id = ?`, id).
		Scan(&r.ID, &r.Name, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	return r, nil
}

// ListRooms returns room summaries ordered: rooms with blockers first, then newest.
func ListRooms(db *sql.DB) ([]RoomSummary, error) {
	rows, err := db.Query(`
		SELECT
			r.id,
			r.name,
			r.created_at,
			COUNT(DISTINCT CASE WHEN a.active = 1 THEN a.id END)         AS agents_count,
			COUNT(DISTINCT CASE WHEN m.priority = 'blocker'
				AND m.status = 'open'
				AND m.parent_id IS NULL THEN m.id END)                    AS blockers_count
		FROM rooms r
		LEFT JOIN agents  a ON a.room_id = r.id
		LEFT JOIN messages m ON m.room_id = r.id
		GROUP BY r.id
		ORDER BY blockers_count DESC, r.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	var summaries []RoomSummary
	for rows.Next() {
		var s RoomSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.CreatedAt, &s.AgentsCount, &s.BlockersCount); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

func DeleteRoom(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM rooms WHERE id = ?`, id)
	return err
}

// ---- Agents ----

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateOrUpdateAgent upserts an agent: if the name already exists in the room, reactivate and
// update it with a fresh token; otherwise create a new row.
func CreateOrUpdateAgent(db *sql.DB, roomID int64, name, role, repo string) (*Agent, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Check for existing (even inactive) agent with this name in the room.
	var existingID int64
	err = db.QueryRow(
		`SELECT id FROM agents WHERE room_id = ? AND name = ?`, roomID, name,
	).Scan(&existingID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup agent: %w", err)
	}

	if existingID != 0 {
		// Reactivate + update.
		_, err = db.Exec(`
			UPDATE agents SET
				role = ?, repo = ?, access_token = ?, active = 1,
				updated_at = datetime('now')
			WHERE id = ?`,
			role, repo, token, existingID)
		if err != nil {
			return nil, fmt.Errorf("reactivate agent: %w", err)
		}
		return GetAgent(db, existingID)
	}

	// Insert new.
	res, err := db.Exec(`
		INSERT INTO agents(room_id, name, role, repo, access_token)
		VALUES(?, ?, ?, ?, ?)`,
		roomID, name, role, repo, token)
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetAgent(db, id)
}

func GetAgent(db *sql.DB, id int64) (*Agent, error) {
	a := &Agent{}
	var active int
	err := db.QueryRow(`
		SELECT id, room_id, name, COALESCE(role,''), COALESCE(repo,''),
		       access_token, active, created_at, updated_at
		FROM agents WHERE id = ?`, id).
		Scan(&a.ID, &a.RoomID, &a.Name, &a.Role, &a.Repo,
			&a.AccessToken, &active, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	a.Active = active == 1
	return a, nil
}

// GetAgentByToken resolves a token to an active agent.
func GetAgentByToken(db *sql.DB, token string) (*Agent, error) {
	a := &Agent{}
	var active int
	err := db.QueryRow(`
		SELECT id, room_id, name, COALESCE(role,''), COALESCE(repo,''),
		       access_token, active, created_at, updated_at
		FROM agents WHERE access_token = ? AND active = 1`, token).
		Scan(&a.ID, &a.RoomID, &a.Name, &a.Role, &a.Repo,
			&a.AccessToken, &active, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent by token: %w", err)
	}
	a.Active = active == 1
	return a, nil
}

// ListAgents returns active agents for a room.
func ListAgents(db *sql.DB, roomID int64) ([]*Agent, error) {
	rows, err := db.Query(`
		SELECT id, room_id, name, COALESCE(role,''), COALESCE(repo,''),
		       access_token, active, created_at, updated_at
		FROM agents WHERE room_id = ? AND active = 1
		ORDER BY name`, roomID)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		a := &Agent{}
		var active int
		if err := rows.Scan(&a.ID, &a.RoomID, &a.Name, &a.Role, &a.Repo,
			&a.AccessToken, &active, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Active = active == 1
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// UpdateAgent updates an agent's metadata. Returns error on duplicate name within room.
func UpdateAgent(db *sql.DB, id int64, name, role, repo string) (*Agent, error) {
	existing, err := GetAgent(db, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	// Check for name collision within room (excluding self).
	var collision int64
	err = db.QueryRow(
		`SELECT id FROM agents WHERE room_id = ? AND name = ? AND id != ?`,
		existing.RoomID, name, id,
	).Scan(&collision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check name collision: %w", err)
	}
	if collision != 0 {
		return nil, fmt.Errorf("agent name %q already exists in this room", name)
	}

	_, err = db.Exec(`
		UPDATE agents SET name = ?, role = ?, repo = ?, updated_at = datetime('now')
		WHERE id = ?`, name, role, repo, id)
	if err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}
	return GetAgent(db, id)
}

// SoftDeleteAgent sets active=0 for the agent.
func SoftDeleteAgent(db *sql.DB, id int64) error {
	_, err := db.Exec(`UPDATE agents SET active = 0, updated_at = datetime('now') WHERE id = ?`, id)
	return err
}

// ---- Messages ----

func scanMessage(rows *sql.Rows) (*Message, error) {
	m := &Message{}
	var parentID sql.NullInt64
	var subject, readAt, closedAt, closedBy sql.NullString
	err := rows.Scan(
		&m.ID, &m.RoomID, &parentID,
		&m.FromAgent, &m.ToAgent, &subject,
		&m.Body, &m.Priority, &m.Status, &m.Type,
		&readAt, &m.CreatedAt, &closedAt, &closedBy,
	)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		m.ParentID = &parentID.Int64
	}
	if subject.Valid {
		m.Subject = subject.String
	}
	if readAt.Valid {
		m.ReadAt = &readAt.String
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.String
	}
	if closedBy.Valid {
		m.ClosedBy = &closedBy.String
	}
	return m, nil
}

func scanMessageRow(row *sql.Row) (*Message, error) {
	m := &Message{}
	var parentID sql.NullInt64
	var subject, readAt, closedAt, closedBy sql.NullString
	err := row.Scan(
		&m.ID, &m.RoomID, &parentID,
		&m.FromAgent, &m.ToAgent, &subject,
		&m.Body, &m.Priority, &m.Status, &m.Type,
		&readAt, &m.CreatedAt, &closedAt, &closedBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		m.ParentID = &parentID.Int64
	}
	if subject.Valid {
		m.Subject = subject.String
	}
	if readAt.Valid {
		m.ReadAt = &readAt.String
	}
	if closedAt.Valid {
		m.ClosedAt = &closedAt.String
	}
	if closedBy.Valid {
		m.ClosedBy = &closedBy.String
	}
	return m, nil
}

const msgCols = `id, room_id, parent_id, from_agent, to_agent, subject, body, priority, status, type, read_at, created_at, closed_at, closed_by`

// CreateMessage inserts a new message and returns it.
func CreateMessage(db *sql.DB, roomID int64, parentID *int64, fromAgent, toAgent, subject, body, priority, msgType string) (*Message, error) {
	if priority == "" {
		priority = "normal"
	}
	if msgType == "" {
		msgType = "request"
	}

	var res sql.Result
	var err error
	if parentID != nil {
		res, err = db.Exec(`
			INSERT INTO messages(room_id, parent_id, from_agent, to_agent, subject, body, priority, type)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			roomID, *parentID, fromAgent, toAgent, nullStr(subject), body, priority, msgType)
	} else {
		res, err = db.Exec(`
			INSERT INTO messages(room_id, from_agent, to_agent, subject, body, priority, type)
			VALUES(?, ?, ?, ?, ?, ?, ?)`,
			roomID, fromAgent, toAgent, nullStr(subject), body, priority, msgType)
	}
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetMessage(db, id)
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetMessage fetches a single message by ID.
func GetMessage(db *sql.DB, id int64) (*Message, error) {
	row := db.QueryRow(`SELECT `+msgCols+` FROM messages WHERE id = ?`, id)
	return scanMessageRow(row)
}

// ListAgentMessages returns messages relevant to the agent and marks unread ones as read.
func ListAgentMessages(db *sql.DB, agent *Agent, includeClosed bool) ([]*Message, error) {
	// Build status clause.
	statusClause := ""
	if !includeClosed {
		statusClause = "AND status = 'open'"
	}

	query := fmt.Sprintf(`
		SELECT `+msgCols+`
		FROM messages
		WHERE room_id = ?
		  AND COALESCE(parent_id, id) IN (
			  SELECT DISTINCT COALESCE(m2.parent_id, m2.id) AS root_id
			  FROM messages m2
			  WHERE m2.room_id = ?
				AND (m2.to_agent = ? OR m2.to_agent = 'all' OR m2.from_agent = ?)
		  )
		  %s
		ORDER BY COALESCE(parent_id, id) ASC, created_at ASC
	`, statusClause)

	rows, err := db.Query(query, agent.RoomID, agent.RoomID, agent.Name, agent.Name)
	if err != nil {
		return nil, fmt.Errorf("list agent messages: %w", err)
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Mark unread messages addressed to this agent as read.
	_, err = db.Exec(`
		UPDATE messages SET read_at = datetime('now')
		WHERE room_id = ? AND to_agent = ? AND read_at IS NULL`,
		agent.RoomID, agent.Name)
	if err != nil {
		return nil, fmt.Errorf("mark read: %w", err)
	}

	return msgs, nil
}

// ReplyToMessage creates a reply, attaching to thread root.
func ReplyToMessage(db *sql.DB, messageID int64, fromAgent, toAgent, body string) (*Message, error) {
	// Fetch the target message.
	target, err := GetMessage(db, messageID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, fmt.Errorf("message %d not found", messageID)
	}

	// Find the root.
	var rootID int64
	if target.ParentID != nil {
		rootID = *target.ParentID
	} else {
		rootID = target.ID
	}

	root, err := GetMessage(db, rootID)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("root message %d not found", rootID)
	}

	// Infer recipient if not provided.
	if toAgent == "" {
		if fromAgent == root.FromAgent {
			toAgent = root.ToAgent
		} else {
			toAgent = root.FromAgent
		}
	}

	subject := "Re: " + root.Subject
	// Strip nested Re: prefixes if root already had one.
	if strings.HasPrefix(root.Subject, "Re: ") {
		subject = root.Subject
	} else {
		subject = "Re: " + root.Subject
	}

	return CreateMessage(db, root.RoomID, &rootID, fromAgent, toAgent, subject, body, root.Priority, "reply")
}

// CloseThread closes the root message and all its replies.
func CloseThread(db *sql.DB, messageID int64, closedBy string) error {
	// Find root.
	msg, err := GetMessage(db, messageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message %d not found", messageID)
	}

	var rootID int64
	if msg.ParentID != nil {
		rootID = *msg.ParentID
	} else {
		rootID = msg.ID
	}

	_, err = db.Exec(`
		UPDATE messages SET
			status = 'closed',
			closed_at = datetime('now'),
			closed_by = ?
		WHERE id = ? OR parent_id = ?`,
		closedBy, rootID, rootID)
	return err
}

// ListGlobalBlockers returns open root messages with priority=blocker across all rooms.
func ListGlobalBlockers(db *sql.DB) ([]*Message, error) {
	rows, err := db.Query(`
		SELECT `+msgCols+`
		FROM messages
		WHERE priority = 'blocker' AND status = 'open' AND parent_id IS NULL
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list global blockers: %w", err)
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// GetRoomMessages returns all messages for a room (admin view).
func GetRoomMessages(db *sql.DB, roomID int64) ([]*Message, error) {
	rows, err := db.Query(`
		SELECT `+msgCols+`
		FROM messages WHERE room_id = ?
		ORDER BY COALESCE(parent_id, id) ASC, created_at ASC`, roomID)
	if err != nil {
		return nil, fmt.Errorf("get room messages: %w", err)
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

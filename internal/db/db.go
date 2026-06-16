package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens the SQLite database at the given path with required pragmas.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite only supports one writer at a time; limit pool to avoid contention.
	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA busy_timeout=5000;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	return db, nil
}

// Migrate runs idempotent DDL to create all tables and indexes.
func Migrate(database *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS rooms (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT    NOT NULL,
			created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS agents (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id       INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			name          TEXT    NOT NULL,
			role          TEXT,
			repo          TEXT,
			access_token  TEXT    NOT NULL UNIQUE,
			active        INTEGER NOT NULL DEFAULT 1,
			created_at    TEXT    NOT NULL DEFAULT (datetime('now')),
			updated_at    TEXT    NOT NULL DEFAULT (datetime('now')),
			UNIQUE(room_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id     INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			parent_id   INTEGER REFERENCES messages(id) ON DELETE CASCADE,
			from_agent  TEXT    NOT NULL,
			to_agent    TEXT    NOT NULL,
			subject     TEXT,
			body        TEXT    NOT NULL,
			priority    TEXT    NOT NULL DEFAULT 'normal'  CHECK(priority IN ('normal','blocker')),
			status      TEXT    NOT NULL DEFAULT 'open'    CHECK(status   IN ('open','closed')),
			type        TEXT    NOT NULL DEFAULT 'request' CHECK(type     IN ('request','reply')),
			read_at     TEXT,
			created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
			closed_at   TEXT,
			closed_by   TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_room      ON messages(room_id)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_parent    ON messages(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_to_status ON messages(to_agent, status)`,
		`CREATE INDEX IF NOT EXISTS idx_msg_blockers  ON messages(priority, status)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_room    ON agents(room_id)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	for _, stmt := range stmts {
		if _, err := database.Exec(stmt); err != nil {
			return fmt.Errorf("migrate stmt %q: %w", stmt[:min(40, len(stmt))], err)
		}
	}
	return nil
}

// GetSetting retrieves a setting value by key.
func GetSetting(database *sql.DB, key string) (string, bool) {
	var value string
	err := database.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", false
	}
	return value, true
}

// SetSetting persists a key/value setting (upsert).
func SetSetting(database *sql.DB, key, value string) error {
	_, err := database.Exec(
		`INSERT INTO settings(key, value) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

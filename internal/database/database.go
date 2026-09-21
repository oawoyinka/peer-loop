package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens (and creates, if needed) the SQLite database file at path
// and runs migrations to make sure all required tables exist.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// SQLite only really supports one writer at a time; keep a single
	// connection so we don't hit "database is locked" errors under load.
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		name          TEXT NOT NULL,
		email         TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		bio           TEXT NOT NULL DEFAULT '',
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS skills (
		id   INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS user_skills (
		user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
		level    INTEGER NOT NULL CHECK (level BETWEEN 1 AND 5),
		PRIMARY KEY (user_id, skill_id)
	);

	CREATE TABLE IF NOT EXISTS sessions (
		token      TEXT PRIMARY KEY,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS messages (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		sender_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		body         TEXT NOT NULL,
		created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id, recipient_id);
	CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient_id, sender_id);

	CREATE TABLE IF NOT EXISTS bookings (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		requester_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		peer_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		skill_name    TEXT NOT NULL,
		scheduled_at  DATETIME NOT NULL,
		meet_link     TEXT NOT NULL DEFAULT '',
		status        TEXT NOT NULL DEFAULT 'pending', -- pending | confirmed | cancelled | completed
		notes         TEXT NOT NULL DEFAULT '',
		recap         TEXT NOT NULL DEFAULT '',
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS rooms (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL DEFAULT '',
		created_by  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS room_members (
		room_id   INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
		user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (room_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS room_posts (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id    INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		body       TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS points (
		user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		total   INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS reviews (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		booking_id  INTEGER NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
		reviewer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		reviewee_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		rating      INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
		comment     TEXT NOT NULL DEFAULT '',
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (booking_id, reviewer_id)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Seed a starter set of skills so the profile/matching UI has
	// something to show on first run. Ignored if they already exist.
	seed := []string{
		"Prompt Engineering", "RAG", "Fine-tuning", "Evaluation",
		"LLM Agents", "Python", "Go", "Backend APIs", "Data Engineering",
	}
	stmt, err := db.Prepare(`INSERT OR IGNORE INTO skills (name) VALUES (?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, name := range seed {
		if _, err := stmt.Exec(name); err != nil {
			return err
		}
	}
	return nil
}

package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open connects to the Postgres database at the given connection string
// (e.g. "postgres://user:pass@host:5432/dbname?sslmode=require") and runs
// migrations to make sure all required tables exist.
func Open(connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            SERIAL PRIMARY KEY,
		name          TEXT NOT NULL,
		email         TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		bio           TEXT NOT NULL DEFAULT '',
		created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS skills (
		id   SERIAL PRIMARY KEY,
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
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS messages (
		id           SERIAL PRIMARY KEY,
		sender_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		body         TEXT NOT NULL,
		created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id, recipient_id);
	CREATE INDEX IF NOT EXISTS idx_messages_recipient ON messages(recipient_id, sender_id);

	CREATE TABLE IF NOT EXISTS bookings (
		id            SERIAL PRIMARY KEY,
		requester_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		peer_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		skill_name    TEXT NOT NULL,
		scheduled_at  TIMESTAMP NOT NULL,
		meet_link     TEXT NOT NULL DEFAULT '',
		status        TEXT NOT NULL DEFAULT 'pending',
		notes         TEXT NOT NULL DEFAULT '',
		recap         TEXT NOT NULL DEFAULT '',
		created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS rooms (
		id          SERIAL PRIMARY KEY,
		name        TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL DEFAULT '',
		created_by  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS room_members (
		room_id   INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
		user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (room_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS room_posts (
		id         SERIAL PRIMARY KEY,
		room_id    INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		body       TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS points (
		user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		total   INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS reviews (
		id          SERIAL PRIMARY KEY,
		booking_id  INTEGER NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
		reviewer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		reviewee_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		rating      INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
		comment     TEXT NOT NULL DEFAULT '',
		created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (booking_id, reviewer_id)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	seed := []string{
		"Prompt Engineering", "RAG", "Fine-tuning", "Evaluation",
		"LLM Agents", "Python", "Go", "Backend APIs", "Data Engineering",
	}
	for _, name := range seed {
		if _, err := db.Exec(`INSERT INTO skills (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, name); err != nil {
			return err
		}
	}
	return nil
}

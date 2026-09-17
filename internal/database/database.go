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

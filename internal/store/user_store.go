package store

import (
	"database/sql"
	"errors"
	"strings"

	"peerloop/internal/models"
)

var ErrNotFound = errors.New("not found")
var ErrEmailTaken = errors.New("email already registered")

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// Create inserts a new user and returns their generated ID.
// passwordHash must already be hashed (see internal/auth).
func (s *UserStore) Create(name, email, passwordHash string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)`,
		name, email, passwordHash,
	)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return 0, ErrEmailTaken
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (s *UserStore) GetByEmail(email string) (models.User, error) {
	var u models.User
	row := s.db.QueryRow(
		`SELECT id, name, email, password_hash, bio, created_at FROM users WHERE email = ?`,
		email,
	)
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Bio, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *UserStore) GetByID(id int64) (models.User, error) {
	var u models.User
	row := s.db.QueryRow(
		`SELECT id, name, email, password_hash, bio, created_at FROM users WHERE id = ?`,
		id,
	)
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Bio, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *UserStore) UpdateBio(userID int64, bio string) error {
	_, err := s.db.Exec(`UPDATE users SET bio = ? WHERE id = ?`, bio, userID)
	return err
}

func isUniqueConstraintErr(err error) bool {
	// modernc.org/sqlite reports constraint violations with this substring;
	// good enough for our purposes without importing its error types directly.
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

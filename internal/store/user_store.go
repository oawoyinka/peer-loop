package store

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

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
	var id int64
	err := s.db.QueryRow(
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		name, email, passwordHash,
	).Scan(&id)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return 0, ErrEmailTaken
		}
		return 0, err
	}
	return id, nil
}

func (s *UserStore) GetByEmail(email string) (models.User, error) {
	var u models.User
	row := s.db.QueryRow(
		`SELECT id, name, email, password_hash, bio, created_at FROM users WHERE email = $1`,
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
		`SELECT id, name, email, password_hash, bio, created_at FROM users WHERE id = $1`,
		id,
	)
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Bio, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *UserStore) UpdateBio(userID int64, bio string) error {
	_, err := s.db.Exec(`UPDATE users SET bio = $1 WHERE id = $2`, bio, userID)
	return err
}

// isUniqueConstraintErr reports whether err is a Postgres unique-violation
// error (SQLSTATE 23505) — e.g. a duplicate email or duplicate review.
func isUniqueConstraintErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

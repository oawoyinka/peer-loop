package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"time"
)

const (
	cookieName = "peerloop_session"
	sessionTTL = 7 * 24 * time.Hour
)

var ErrNoSession = errors.New("no valid session")

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create makes a new session for userID and sets the session cookie on w.
func (s *SessionStore) Create(w http.ResponseWriter, userID int64) error {
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	expires := time.Now().Add(sessionTTL)

	_, err = s.db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expires,
	)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // enable once served over HTTPS
	})
	return nil
}

// UserIDFromRequest reads the session cookie from r and returns the
// associated user ID, or ErrNoSession if there isn't a valid one.
func (s *SessionStore) UserIDFromRequest(r *http.Request) (int64, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return 0, ErrNoSession
	}

	var userID int64
	var expiresAt time.Time
	err = s.db.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE token = ?`,
		cookie.Value,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNoSession
	}
	if err != nil {
		return 0, err
	}
	if time.Now().After(expiresAt) {
		return 0, ErrNoSession
	}
	return userID, nil
}

// Destroy deletes the session referenced by the cookie on r and clears
// the cookie on the client.
func (s *SessionStore) Destroy(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

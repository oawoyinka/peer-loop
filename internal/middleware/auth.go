package middleware

import (
	"context"
	"net/http"

	"peerloop/internal/auth"
)

type contextKey string

const userIDKey contextKey = "userID"

// RequireAuth wraps a handler so it only runs for logged-in users;
// anyone else is redirected to /login. On success, the user's ID is
// stashed in the request context for the handler to read via UserID.
func RequireAuth(sessions *auth.SessionStore) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			userID, err := sessions.UserIDFromRequest(r)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next(w, r.WithContext(ctx))
		}
	}
}

// UserID reads the authenticated user's ID out of the request context.
// Only valid inside a handler wrapped with RequireAuth.
func UserID(r *http.Request) int64 {
	id, _ := r.Context().Value(userIDKey).(int64)
	return id
}

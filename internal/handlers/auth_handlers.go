package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"peerloop/internal/auth"
	"peerloop/internal/store"
)

type AuthHandlers struct {
	users     *store.UserStore
	sessions  *auth.SessionStore
	templates *template.Template
}

func NewAuthHandlers(users *store.UserStore, sessions *auth.SessionStore, templates *template.Template) *AuthHandlers {
	return &AuthHandlers{users: users, sessions: sessions, templates: templates}
}

type formPage struct {
	Error string
}

func (h *AuthHandlers) RegisterPage(w http.ResponseWriter, r *http.Request) {
	render(w, h.templates, "register.html", formPage{})
}

func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")

	if name == "" || email == "" || len(password) < 8 {
		render(w, h.templates, "register.html", formPage{
			Error: "Name, email and a password of at least 8 characters are required.",
		})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	userID, err := h.users.Create(name, email, hash)
	if err == store.ErrEmailTaken {
		render(w, h.templates, "register.html", formPage{Error: "That email is already registered."})
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.sessions.Create(w, userID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	render(w, h.templates, "login.html", formPage{})
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")

	user, err := h.users.GetByEmail(email)
	if err == store.ErrNotFound || !auth.CheckPassword(user.PasswordHash, password) {
		render(w, h.templates, "login.html", formPage{Error: "Incorrect email or password."})
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.sessions.Create(w, user.ID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessions.Destroy(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

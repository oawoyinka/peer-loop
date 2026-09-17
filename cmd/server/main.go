package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"peerloop/internal/auth"
	"peerloop/internal/database"
	"peerloop/internal/handlers"
	"peerloop/internal/middleware"
	"peerloop/internal/store"
)

func main() {
	dbPath := getEnv("DB_PATH", "data/peerloop.db")
	port := getEnv("PORT", "8080")

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	templates, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	userStore := store.NewUserStore(db)
	skillStore := store.NewSkillStore(db)
	sessions := auth.NewSessionStore(db)

	authH := handlers.NewAuthHandlers(userStore, sessions, templates)
	profileH := handlers.NewProfileHandlers(userStore, skillStore, templates)
	matchH := handlers.NewMatchHandlers(skillStore, templates)

	requireAuth := middleware.RequireAuth(sessions)

	mux := http.NewServeMux()

	// Static assets.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Public routes.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /register", authH.RegisterPage)
	mux.HandleFunc("POST /register", authH.Register)
	mux.HandleFunc("GET /login", authH.LoginPage)
	mux.HandleFunc("POST /login", authH.Login)
	mux.HandleFunc("GET /logout", authH.Logout)

	// Authenticated routes.
	mux.HandleFunc("GET /profile", requireAuth(profileH.View))
	mux.HandleFunc("POST /profile/bio", requireAuth(profileH.UpdateBio))
	mux.HandleFunc("POST /profile/skills", requireAuth(profileH.AddSkill))
	mux.HandleFunc("GET /peers", requireAuth(matchH.Peers))

	log.Printf("PeerLoop listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

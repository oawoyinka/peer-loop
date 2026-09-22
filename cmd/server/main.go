package main

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"

	"peerloop/internal/aiclient"
	"peerloop/internal/auth"
	"peerloop/internal/database"
	"peerloop/internal/handlers"
	"peerloop/internal/middleware"
	"peerloop/internal/store"
	"peerloop/web"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set. Example: postgres://user:pass@host:5432/dbname?sslmode=require")
	}
	port := getEnv("PORT", "8080")

	db, err := database.Open(databaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	templates, err := template.ParseFS(web.Templates, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	// Stores
	userStore := store.NewUserStore(db)
	skillStore := store.NewSkillStore(db)
	sessions := auth.NewSessionStore(db)
	messageStore := store.NewMessageStore(db)
	bookingStore := store.NewBookingStore(db)
	roomStore := store.NewRoomStore(db)
	pointsStore := store.NewPointsStore(db)
	reviewStore := store.NewReviewStore(db)
	aiClient := aiclient.NewClient()

	if !aiClient.Available() {
		log.Println("NOTE: ANTHROPIC_API_KEY is not set - AI session recaps will show a setup message instead of generating.")
	}

	// Handlers
	authH := handlers.NewAuthHandlers(userStore, sessions, templates)
	profileH := handlers.NewProfileHandlers(userStore, skillStore, pointsStore, templates)
	matchH := handlers.NewMatchHandlers(skillStore, templates)
	dashboardH := handlers.NewDashboardHandlers(userStore, skillStore, pointsStore, templates)
	chatH := handlers.NewChatHandlers(userStore, skillStore, messageStore, pointsStore, templates)
	bookingH := handlers.NewBookingHandlers(bookingStore, skillStore, pointsStore, aiClient, templates)
	roomH := handlers.NewRoomHandlers(roomStore, pointsStore, templates)
	reviewH := handlers.NewReviewHandlers(reviewStore, bookingStore, pointsStore, templates)
	leaderboardH := handlers.NewLeaderboardHandlers(pointsStore, templates)

	requireAuth := middleware.RequireAuth(sessions)

	mux := http.NewServeMux()

	// Static assets (embedded into the binary — see web/embed.go).
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		log.Fatalf("failed to load embedded static assets: %v", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Public routes.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /register", authH.RegisterPage)
	mux.HandleFunc("POST /register", authH.Register)
	mux.HandleFunc("GET /login", authH.LoginPage)
	mux.HandleFunc("POST /login", authH.Login)
	mux.HandleFunc("GET /logout", authH.Logout)

	// Dashboard, profile, matching.
	mux.HandleFunc("GET /dashboard", requireAuth(dashboardH.View))
	mux.HandleFunc("GET /profile", requireAuth(profileH.View))
	mux.HandleFunc("POST /profile/bio", requireAuth(profileH.UpdateBio))
	mux.HandleFunc("POST /profile/skills", requireAuth(profileH.AddSkill))
	mux.HandleFunc("GET /peers", requireAuth(matchH.Peers))

	// Chat.
	mux.HandleFunc("GET /chat", requireAuth(chatH.Inbox))
	mux.HandleFunc("GET /chat/{userID}", requireAuth(chatH.Conversation))
	mux.HandleFunc("POST /chat/{userID}/send", requireAuth(chatH.Send))

	// Session booking + AI recaps.
	mux.HandleFunc("GET /bookings", requireAuth(bookingH.List))
	mux.HandleFunc("GET /book/{userID}", requireAuth(bookingH.New))
	mux.HandleFunc("POST /book/{userID}", requireAuth(bookingH.Create))
	mux.HandleFunc("POST /bookings/{id}/confirm", requireAuth(bookingH.Confirm))
	mux.HandleFunc("POST /bookings/{id}/cancel", requireAuth(bookingH.Cancel))
	mux.HandleFunc("POST /bookings/{id}/complete", requireAuth(bookingH.Complete))
	mux.HandleFunc("POST /bookings/{id}/recap", requireAuth(bookingH.GenerateRecap))

	// Reviews.
	mux.HandleFunc("GET /reviews/new/{id}", requireAuth(reviewH.New))
	mux.HandleFunc("POST /reviews/new/{id}", requireAuth(reviewH.Create))

	// Community rooms.
	mux.HandleFunc("GET /rooms", requireAuth(roomH.List))
	mux.HandleFunc("POST /rooms", requireAuth(roomH.Create))
	mux.HandleFunc("GET /rooms/{id}", requireAuth(roomH.View))
	mux.HandleFunc("POST /rooms/{id}/join", requireAuth(roomH.Join))
	mux.HandleFunc("POST /rooms/{id}/posts", requireAuth(roomH.Post))

	// Points / leaderboard.
	mux.HandleFunc("GET /leaderboard", requireAuth(leaderboardH.View))

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

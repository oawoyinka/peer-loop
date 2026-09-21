package handlers

import (
	"html/template"
	"net/http"

	"peerloop/internal/middleware"
	"peerloop/internal/models"
	"peerloop/internal/store"
)

type LeaderboardHandlers struct {
	points    *store.PointsStore
	templates *template.Template
}

func NewLeaderboardHandlers(points *store.PointsStore, templates *template.Template) *LeaderboardHandlers {
	return &LeaderboardHandlers{points: points, templates: templates}
}

type leaderboardPage struct {
	Entries   []models.LeaderboardEntry
	MyTotal   int
	MyBadges  []string
}

// View shows the top 20 users by points, plus the logged-in user's own
// total and earned badges.
func (h *LeaderboardHandlers) View(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	entries, err := h.points.Leaderboard(20)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	myTotal, err := h.points.Total(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	render(w, h.templates, "leaderboard.html", leaderboardPage{
		Entries:  entries,
		MyTotal:  myTotal,
		MyBadges: store.Badges(myTotal),
	})
}

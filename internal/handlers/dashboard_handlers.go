package handlers

import (
	"html/template"
	"net/http"

	"peerloop/internal/middleware"
	"peerloop/internal/models"
	"peerloop/internal/store"
)

type DashboardHandlers struct {
	users     *store.UserStore
	skills    *store.SkillStore
	templates *template.Template
}

func NewDashboardHandlers(users *store.UserStore, skills *store.SkillStore, templates *template.Template) *DashboardHandlers {
	return &DashboardHandlers{users: users, skills: skills, templates: templates}
}

type dashboardPage struct {
	User        models.User
	SkillCount  int
	TopMatches  []models.PeerMatch
	TotalMatches int
}

// View shows a summary landing page after login: who you are, how many
// skills you've rated yourself on, and your best peer matches so far.
func (h *DashboardHandlers) View(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	user, err := h.users.GetByID(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	mySkills, err := h.skills.ForUser(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	allMatches, err := h.skills.FindMatches(userID, "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	top := allMatches
	if len(top) > 3 {
		top = top[:3]
	}

	render(w, h.templates, "dashboard.html", dashboardPage{
		User:         user,
		SkillCount:   len(mySkills),
		TopMatches:   top,
		TotalMatches: len(allMatches),
	})
}

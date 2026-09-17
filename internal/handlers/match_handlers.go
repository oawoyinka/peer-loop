package handlers

import (
	"html/template"
	"net/http"

	"peerloop/internal/middleware"
	"peerloop/internal/models"
	"peerloop/internal/store"
)

type MatchHandlers struct {
	skills    *store.SkillStore
	templates *template.Template
}

func NewMatchHandlers(skills *store.SkillStore, templates *template.Template) *MatchHandlers {
	return &MatchHandlers{skills: skills, templates: templates}
}

type peersPage struct {
	Matches     []models.PeerMatch
	AllSkills   []models.Skill
	SkillFilter string
}

// Peers lists candidate peers for the logged-in user, ranked by skill-level
// gap (biggest gaps = clearest teacher/learner pairing). Supports
// ?skill=Name to narrow to one skill.
func (h *MatchHandlers) Peers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	skillFilter := r.URL.Query().Get("skill")

	matches, err := h.skills.FindMatches(userID, skillFilter)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	allSkills, err := h.skills.ListAll()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	render(w, h.templates, "peers.html", peersPage{
		Matches: matches, AllSkills: allSkills, SkillFilter: skillFilter,
	})
}

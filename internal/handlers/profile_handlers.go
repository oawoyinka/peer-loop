package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"peerloop/internal/middleware"
	"peerloop/internal/models"
	"peerloop/internal/store"
)

type ProfileHandlers struct {
	users     *store.UserStore
	skills    *store.SkillStore
	points    *store.PointsStore
	templates *template.Template
}

func NewProfileHandlers(users *store.UserStore, skills *store.SkillStore, points *store.PointsStore, templates *template.Template) *ProfileHandlers {
	return &ProfileHandlers{users: users, skills: skills, points: points, templates: templates}
}

type profilePage struct {
	User       models.User
	MySkills   []models.UserSkill
	AllSkills  []models.Skill
	Error      string
	Success    string
}

func (h *ProfileHandlers) View(w http.ResponseWriter, r *http.Request) {
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
	allSkills, err := h.skills.ListAll()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	page := profilePage{User: user, MySkills: mySkills, AllSkills: allSkills}
	switch r.URL.Query().Get("saved") {
	case "bio":
		page.Success = "Bio saved."
	case "skill":
		page.Success = "Skill saved."
	}
	if r.URL.Query().Get("error") == "badskill" {
		page.Error = "Pick a skill name and a level between 1 and 5."
	}

	render(w, h.templates, "profile.html", page)
}

// UpdateBio saves the free-text bio field.
func (h *ProfileHandlers) UpdateBio(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	bio := strings.TrimSpace(r.FormValue("bio"))
	if err := h.users.UpdateBio(userID, bio); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/profile?saved=bio", http.StatusSeeOther)
}

// AddSkill lets a user rate themselves 1-5 on an existing or new skill.
func (h *ProfileHandlers) AddSkill(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	skillName := strings.TrimSpace(r.FormValue("skill_name"))
	level, err := strconv.Atoi(r.FormValue("level"))
	if skillName == "" || err != nil || level < 1 || level > 5 {
		http.Redirect(w, r, "/profile?error=badskill", http.StatusSeeOther)
		return
	}

	skillID, err := h.skills.GetOrCreateByName(skillName)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.skills.SetUserSkill(userID, skillID, level); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = h.points.Award(userID, store.PointsForSkillAdded)
	http.Redirect(w, r, "/profile?saved=skill", http.StatusSeeOther)
}

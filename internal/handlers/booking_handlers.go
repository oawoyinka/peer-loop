package handlers

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"peerloop/internal/aiclient"
	"peerloop/internal/middleware"
	"peerloop/internal/models"
	"peerloop/internal/store"
)

type BookingHandlers struct {
	bookings  *store.BookingStore
	skills    *store.SkillStore
	points    *store.PointsStore
	ai        *aiclient.Client
	templates *template.Template
}

func NewBookingHandlers(bookings *store.BookingStore, skills *store.SkillStore, points *store.PointsStore, ai *aiclient.Client, templates *template.Template) *BookingHandlers {
	return &BookingHandlers{bookings: bookings, skills: skills, points: points, ai: ai, templates: templates}
}

type bookingsPage struct {
	Bookings []models.Booking
	UserID   int64
	Error    string
	Success  string
}

// List shows every session (pending, confirmed, completed) the logged-in
// user is part of, either side.
func (h *BookingHandlers) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	bookings, err := h.bookings.ForUser(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	page := bookingsPage{Bookings: bookings, UserID: userID}
	switch r.URL.Query().Get("msg") {
	case "proposed":
		page.Success = "Session proposed. Waiting for them to confirm."
	case "confirmed":
		page.Success = "Session confirmed."
	case "cancelled":
		page.Success = "Session cancelled."
	case "completed":
		page.Success = "Marked as completed. You can leave a review or generate a recap now."
	case "recap":
		page.Success = "Recap generated."
	case "reviewed":
		page.Success = "Review submitted. Thanks for the feedback."
	}
	if r.URL.Query().Get("err") == "ai" {
		page.Error = "Couldn't generate a recap right now: " + r.URL.Query().Get("detail")
	}

	render(w, h.templates, "bookings.html", page)
}

// New shows the "propose a session" form for a specific peer.
func (h *BookingHandlers) New(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	peerID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	matched, err := h.skills.IsMatch(userID, peerID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !matched {
		http.Error(w, "You can only book sessions with peers you're matched with.", http.StatusForbidden)
		return
	}

	matches, err := h.skills.FindMatches(userID, "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var sharedSkills []string
	for _, m := range matches {
		if m.User.ID == peerID {
			sharedSkills = append(sharedSkills, m.SkillName)
		}
	}

	render(w, h.templates, "book_session.html", struct {
		PeerID       int64
		SharedSkills []string
	}{PeerID: peerID, SharedSkills: sharedSkills})
}

// Create proposes a new session with a peer.
func (h *BookingHandlers) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	peerID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	matched, err := h.skills.IsMatch(userID, peerID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !matched {
		http.Error(w, "You can only book sessions with peers you're matched with.", http.StatusForbidden)
		return
	}

	skillName := strings.TrimSpace(r.FormValue("skill_name"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	dateStr := strings.TrimSpace(r.FormValue("scheduled_at")) // from <input type="datetime-local">

	scheduledAt, err := time.Parse("2006-01-02T15:04", dateStr)
	if err != nil || skillName == "" {
		http.Redirect(w, r, fmt.Sprintf("/book/%d?err=badform", peerID), http.StatusSeeOther)
		return
	}

	meetLink := generateMeetLink()

	_, err = h.bookings.Create(userID, peerID, skillName, scheduledAt, meetLink, notes)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = h.points.Award(userID, store.PointsForSessionBooked)

	http.Redirect(w, r, "/bookings?msg=proposed", http.StatusSeeOther)
}

// generateMeetLink returns a fresh Google Meet "instant meeting" link.
// meet.google.com/new is a real Google URL that starts a brand-new
// meeting when opened by a signed-in Google user — no Calendar API/OAuth
// needed for a project at this stage. Swap this out for the Calendar API
// later if you want the link auto-created server-side ahead of time.
func generateMeetLink() string {
	return "https://meet.google.com/new"
}

// randomCode is kept for future use if you switch to generating your own
// room-style identifiers instead of relying on meet.google.com/new.
func randomCode(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}

// Confirm accepts a pending session (only the invited peer can confirm).
func (h *BookingHandlers) Confirm(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "confirmed", "confirmed")
}

// Cancel cancels a session (either participant can cancel).
func (h *BookingHandlers) Cancel(w http.ResponseWriter, r *http.Request) {
	h.updateStatus(w, r, "cancelled", "cancelled")
}

// Complete marks a session as completed, unlocking recaps and reviews.
func (h *BookingHandlers) Complete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.bookings.SetStatus(id, userID, "completed"); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = h.points.Award(userID, store.PointsForSessionDone)
	http.Redirect(w, r, "/bookings?msg=completed", http.StatusSeeOther)
}

func (h *BookingHandlers) updateStatus(w http.ResponseWriter, r *http.Request, status, msg string) {
	userID := middleware.UserID(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.bookings.SetStatus(id, userID, status); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/bookings?msg="+msg, http.StatusSeeOther)
}

// GenerateRecap calls the Anthropic API to summarize a completed
// session's notes. Requires ANTHROPIC_API_KEY to be set in the
// environment; otherwise it redirects back with a clear explanation.
func (h *BookingHandlers) GenerateRecap(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	booking, err := h.bookings.Get(id)
	if err == store.ErrBookingNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !booking.IsParticipant(userID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	recap, err := h.ai.GenerateRecap(booking.SkillName, booking.Notes)
	if err != nil {
		msg := err.Error()
		if err == aiclient.ErrNoAPIKey {
			msg = "AI recaps need an ANTHROPIC_API_KEY environment variable set on the server."
		}
		http.Redirect(w, r, fmt.Sprintf("/bookings?err=ai&detail=%s", template.URLQueryEscaper(msg)), http.StatusSeeOther)
		return
	}

	if err := h.bookings.SetRecap(id, recap); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/bookings?msg=recap", http.StatusSeeOther)
}

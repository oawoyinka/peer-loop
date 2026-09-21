package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"peerloop/internal/middleware"
	"peerloop/internal/store"
)

type ReviewHandlers struct {
	reviews   *store.ReviewStore
	bookings  *store.BookingStore
	points    *store.PointsStore
	templates *template.Template
}

func NewReviewHandlers(reviews *store.ReviewStore, bookings *store.BookingStore, points *store.PointsStore, templates *template.Template) *ReviewHandlers {
	return &ReviewHandlers{reviews: reviews, bookings: bookings, points: points, templates: templates}
}

// New shows the "leave a review" form for a completed booking.
func (h *ReviewHandlers) New(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	bookingID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	booking, err := h.bookings.Get(bookingID)
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
	if booking.Status != "completed" {
		http.Error(w, "You can only review sessions that have been marked completed.", http.StatusForbidden)
		return
	}

	already, err := h.reviews.HasReviewed(bookingID, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if already {
		http.Redirect(w, r, "/bookings", http.StatusSeeOther)
		return
	}

	render(w, h.templates, "review_form.html", booking)
}

// Create saves a review for a completed booking. reviewee is inferred as
// "whichever participant isn't the reviewer" — a review is always about
// the other person in that specific session.
func (h *ReviewHandlers) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	bookingID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	booking, err := h.bookings.Get(bookingID)
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
	if booking.Status != "completed" {
		http.Error(w, "session not completed yet", http.StatusForbidden)
		return
	}

	rating, err := strconv.Atoi(r.FormValue("rating"))
	if err != nil || rating < 1 || rating > 5 {
		http.Redirect(w, r, "/bookings", http.StatusSeeOther)
		return
	}
	comment := strings.TrimSpace(r.FormValue("comment"))
	revieweeID := booking.OtherParticipant(userID).ID

	err = h.reviews.Create(bookingID, userID, revieweeID, rating, comment)
	if err == store.ErrAlreadyReviewed {
		http.Redirect(w, r, "/bookings", http.StatusSeeOther)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = h.points.Award(userID, store.PointsForReviewGiven)

	http.Redirect(w, r, "/bookings?msg=reviewed", http.StatusSeeOther)
}

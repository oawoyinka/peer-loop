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

type RoomHandlers struct {
	rooms     *store.RoomStore
	points    *store.PointsStore
	templates *template.Template
}

func NewRoomHandlers(rooms *store.RoomStore, points *store.PointsStore, templates *template.Template) *RoomHandlers {
	return &RoomHandlers{rooms: rooms, points: points, templates: templates}
}

type roomsPage struct {
	Rooms []models.Room
	Error string
}

// List shows every community room / learning circle.
func (h *RoomHandlers) List(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.rooms.ListAll()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	page := roomsPage{Rooms: rooms}
	if r.URL.Query().Get("error") == "nametaken" {
		page.Error = "A room with that name already exists."
	}
	render(w, h.templates, "rooms.html", page)
}

// Create makes a new room (creator is auto-joined).
func (h *RoomHandlers) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		http.Redirect(w, r, "/rooms", http.StatusSeeOther)
		return
	}

	roomID, err := h.rooms.Create(name, description, userID)
	if err == store.ErrRoomNameTaken {
		http.Redirect(w, r, "/rooms?error=nametaken", http.StatusSeeOther)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/rooms/"+strconv.FormatInt(roomID, 10), http.StatusSeeOther)
}

type roomViewPage struct {
	Room     models.Room
	Posts    []models.RoomPost
	IsMember bool
	UserID   int64
}

// View shows a room's description and its shared post feed. Anyone can
// view; only members can post (joining is a one-click action).
func (h *RoomHandlers) View(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	roomID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	room, err := h.rooms.Get(roomID)
	if err == store.ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	posts, err := h.rooms.Feed(roomID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	isMember, err := h.rooms.IsMember(roomID, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	render(w, h.templates, "room_view.html", roomViewPage{
		Room: room, Posts: posts, IsMember: isMember, UserID: userID,
	})
}

// Join adds the logged-in user to a room's membership.
func (h *RoomHandlers) Join(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	roomID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.rooms.Join(roomID, userID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/rooms/"+r.PathValue("id"), http.StatusSeeOther)
}

// Post adds a message to a room's feed (members only).
func (h *RoomHandlers) Post(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)
	roomID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	isMember, err := h.rooms.IsMember(roomID, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "join the room before posting", http.StatusForbidden)
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	if body != "" {
		if err := h.rooms.Post(roomID, userID, body); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = h.points.Award(userID, store.PointsForRoomPost)
	}

	http.Redirect(w, r, "/rooms/"+r.PathValue("id"), http.StatusSeeOther)
}

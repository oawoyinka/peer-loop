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

type ChatHandlers struct {
	users     *store.UserStore
	skills    *store.SkillStore
	messages  *store.MessageStore
	points    *store.PointsStore
	templates *template.Template
}

func NewChatHandlers(users *store.UserStore, skills *store.SkillStore, messages *store.MessageStore, points *store.PointsStore, templates *template.Template) *ChatHandlers {
	return &ChatHandlers{users: users, skills: skills, messages: messages, points: points, templates: templates}
}

type chatInboxPage struct {
	Conversations []models.ConversationSummary
}

// Inbox lists everyone the logged-in user has exchanged messages with.
func (h *ChatHandlers) Inbox(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	convos, err := h.messages.Conversations(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	render(w, h.templates, "chat_inbox.html", chatInboxPage{Conversations: convos})
}

type chatConversationPage struct {
	OtherUser models.User
	Messages  []models.Message
	Error     string
}

// Conversation shows the full thread with one other user. Only allowed
// between matched peers (people who share at least one rated skill).
func (h *ChatHandlers) Conversation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	otherID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if otherID == userID {
		http.NotFound(w, r)
		return
	}

	otherUser, err := h.users.GetByID(otherID)
	if err == store.ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	matched, err := h.skills.IsMatch(userID, otherID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !matched {
		http.Error(w, "You can only message peers you're matched with (you need to share at least one rated skill).", http.StatusForbidden)
		return
	}

	thread, err := h.messages.Thread(userID, otherID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	render(w, h.templates, "chat_conversation.html", chatConversationPage{
		OtherUser: otherUser,
		Messages:  thread,
	})
}

// Send posts a new message into the conversation with the given user.
func (h *ChatHandlers) Send(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r)

	otherID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Redirect(w, r, "/chat/"+r.PathValue("userID"), http.StatusSeeOther)
		return
	}

	matched, err := h.skills.IsMatch(userID, otherID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !matched {
		http.Error(w, "You can only message peers you're matched with.", http.StatusForbidden)
		return
	}

	existing, err := h.messages.Thread(userID, otherID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	isFirstMessage := len(existing) == 0

	if err := h.messages.Send(userID, otherID, body); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if isFirstMessage {
		_ = h.points.Award(userID, store.PointsForFirstMessage)
	}

	http.Redirect(w, r, "/chat/"+r.PathValue("userID"), http.StatusSeeOther)
}

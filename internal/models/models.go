package models

import "time"

// User represents a registered fellow on PeerLoop.
type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Bio          string
	CreatedAt    time.Time
}

// Skill is a predefined or user-added skill tag (e.g. "RAG", "Prompt Engineering").
type Skill struct {
	ID   int64
	Name string
}

// UserSkill links a user to a skill with a self-rated level from 1-5.
type UserSkill struct {
	UserID    int64
	SkillID   int64
	SkillName string
	Level     int
}

// PeerMatch is a candidate peer surfaced by the matching logic, along with
// the shared skill and the level gap that makes them a useful pairing.
type PeerMatch struct {
	User      User
	SkillName string
	TheirLevel int
	MyLevel    int
	Gap        int // TheirLevel - MyLevel; positive means they can teach you
}

// Message is a single chat message between two users.
type Message struct {
	ID          int64
	SenderID    int64
	RecipientID int64
	Body        string
	CreatedAt   time.Time
	Mine        bool // set at query time: true if the viewing user sent it
}

// ConversationSummary is one row in a user's chat inbox: who the other
// person is, and a preview of the most recent message.
type ConversationSummary struct {
	OtherUser  User
	LastBody   string
	LastAt     time.Time
	LastFromMe bool
}

// Booking is a scheduled mentoring session between two matched peers.
type Booking struct {
	ID          int64
	RequesterID int64
	PeerID      int64
	Requester   User
	Peer        User
	SkillName   string
	ScheduledAt time.Time
	MeetLink    string
	Status      string // pending | confirmed | cancelled | completed
	Notes       string
	Recap       string
	CreatedAt   time.Time
}

// IsParticipant reports whether userID is one of the two people in this booking.
func (b Booking) IsParticipant(userID int64) bool {
	return b.RequesterID == userID || b.PeerID == userID
}

// OtherParticipant returns the other user in the booking relative to userID.
func (b Booking) OtherParticipant(userID int64) User {
	if b.RequesterID == userID {
		return b.Peer
	}
	return b.Requester
}

// Room is a community learning circle: a named group with a shared post feed.
type Room struct {
	ID          int64
	Name        string
	Description string
	CreatedBy   int64
	MemberCount int
	CreatedAt   time.Time
}

// RoomPost is one message in a room's shared feed.
type RoomPost struct {
	ID         int64
	RoomID     int64
	UserID     int64
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

// Review is peer feedback left after a completed booking.
type Review struct {
	ID         int64
	BookingID  int64
	ReviewerID int64
	RevieweeID int64
	Rating     int
	Comment    string
	CreatedAt  time.Time
}

// LeaderboardEntry is one row on the points leaderboard.
type LeaderboardEntry struct {
	User  User
	Total int
	Rank  int
}

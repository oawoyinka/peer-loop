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

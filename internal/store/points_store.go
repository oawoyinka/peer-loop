package store

import (
	"database/sql"

	"peerloop/internal/models"
)

// Point values for qualifying actions, exported so handlers can reference
// them (and so it's obvious where to tune the economy).
const (
	PointsForSkillAdded    = 5
	PointsForFirstMessage  = 2
	PointsForSessionBooked = 5
	PointsForSessionDone   = 20
	PointsForRoomPost      = 3
	PointsForReviewGiven   = 10
)

type PointsStore struct {
	db *sql.DB
}

func NewPointsStore(db *sql.DB) *PointsStore {
	return &PointsStore{db: db}
}

// Award adds amount points to userID's running total (creating the row
// if it doesn't exist yet). amount may be negative to deduct, though
// nothing currently does.
func (s *PointsStore) Award(userID int64, amount int) error {
	_, err := s.db.Exec(`
		INSERT INTO points (user_id, total) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET total = points.total + excluded.total
	`, userID, amount)
	return err
}

// Total returns userID's current point total (0 if they have no row yet).
func (s *PointsStore) Total(userID int64) (int, error) {
	var total int
	err := s.db.QueryRow(`SELECT total FROM points WHERE user_id = $1`, userID).Scan(&total)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return total, err
}

// Leaderboard returns the top `limit` users by point total, descending.
func (s *PointsStore) Leaderboard(limit int) ([]models.LeaderboardEntry, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.name, u.email, p.total
		FROM points p
		JOIN users u ON u.id = p.user_id
		WHERE p.total > 0
		ORDER BY p.total DESC, u.name ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e models.LeaderboardEntry
		if err := rows.Scan(&e.User.ID, &e.User.Name, &e.User.Email, &e.Total); err != nil {
			return nil, err
		}
		e.Rank = rank
		rank++
		out = append(out, e)
	}
	return out, rows.Err()
}

// Badges derives simple threshold-based badge names from a point total.
// Badges aren't stored; they're just a presentation of the total.
func Badges(total int) []string {
	var badges []string
	if total >= 10 {
		badges = append(badges, "Getting Started")
	}
	if total >= 50 {
		badges = append(badges, "Active Peer")
	}
	if total >= 100 {
		badges = append(badges, "Community Pillar")
	}
	if total >= 250 {
		badges = append(badges, "PeerLoop Mentor")
	}
	return badges
}

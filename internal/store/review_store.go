package store

import (
	"database/sql"
	"errors"

	"peerloop/internal/models"
)

var ErrAlreadyReviewed = errors.New("you've already reviewed this session")
var ErrBookingNotCompleted = errors.New("this session hasn't been marked completed yet")

type ReviewStore struct {
	db *sql.DB
}

func NewReviewStore(db *sql.DB) *ReviewStore {
	return &ReviewStore{db: db}
}

// Create leaves a review for a completed booking. Call sites are
// responsible for checking the booking is completed and the reviewer
// was a participant — see ReviewHandlers.Create for the actual gate.
func (s *ReviewStore) Create(bookingID, reviewerID, revieweeID int64, rating int, comment string) error {
	_, err := s.db.Exec(`
		INSERT INTO reviews (booking_id, reviewer_id, reviewee_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5)
	`, bookingID, reviewerID, revieweeID, rating, comment)
	if err != nil && isUniqueConstraintErr(err) {
		return ErrAlreadyReviewed
	}
	return err
}

// ForUser returns every review left about userID (their public reputation),
// newest first.
func (s *ReviewStore) ForUser(userID int64) ([]models.Review, error) {
	rows, err := s.db.Query(`
		SELECT id, booking_id, reviewer_id, reviewee_id, rating, comment, created_at
		FROM reviews WHERE reviewee_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Review
	for rows.Next() {
		var rv models.Review
		if err := rows.Scan(&rv.ID, &rv.BookingID, &rv.ReviewerID, &rv.RevieweeID, &rv.Rating, &rv.Comment, &rv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

// AverageRating returns userID's average rating and review count.
func (s *ReviewStore) AverageRating(userID int64) (avg float64, count int, err error) {
	row := s.db.QueryRow(`SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM reviews WHERE reviewee_id = $1`, userID)
	err = row.Scan(&avg, &count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	return avg, count, err
}

// HasReviewed reports whether reviewerID has already reviewed bookingID.
func (s *ReviewStore) HasReviewed(bookingID, reviewerID int64) (bool, error) {
	var exists int
	err := s.db.QueryRow(
		`SELECT 1 FROM reviews WHERE booking_id = $1 AND reviewer_id = $2`,
		bookingID, reviewerID,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

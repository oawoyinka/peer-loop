package store

import (
	"database/sql"
	"errors"

	"peerloop/internal/models"
)

var ErrRoomNameTaken = errors.New("a room with that name already exists")

type RoomStore struct {
	db *sql.DB
}

func NewRoomStore(db *sql.DB) *RoomStore {
	return &RoomStore{db: db}
}

// Create makes a new room and automatically joins the creator to it.
func (s *RoomStore) Create(name, description string, createdBy int64) (int64, error) {
	var roomID int64
	err := s.db.QueryRow(
		`INSERT INTO rooms (name, description, created_by) VALUES ($1, $2, $3) RETURNING id`,
		name, description, createdBy,
	).Scan(&roomID)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return 0, ErrRoomNameTaken
		}
		return 0, err
	}
	if err := s.Join(roomID, createdBy); err != nil {
		return 0, err
	}
	return roomID, nil
}

// ListAll returns every room with its member count, newest first.
func (s *RoomStore) ListAll() ([]models.Room, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.name, r.description, r.created_by, r.created_at,
		       COUNT(rm.user_id) AS member_count
		FROM rooms r
		LEFT JOIN room_members rm ON rm.room_id = r.id
		GROUP BY r.id
		ORDER BY r.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Room
	for rows.Next() {
		var rm models.Room
		if err := rows.Scan(&rm.ID, &rm.Name, &rm.Description, &rm.CreatedBy, &rm.CreatedAt, &rm.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, rows.Err()
}

// Get fetches one room by ID.
func (s *RoomStore) Get(id int64) (models.Room, error) {
	var rm models.Room
	err := s.db.QueryRow(`
		SELECT r.id, r.name, r.description, r.created_by, r.created_at,
		       (SELECT COUNT(*) FROM room_members WHERE room_id = r.id) AS member_count
		FROM rooms r WHERE r.id = $1
	`, id).Scan(&rm.ID, &rm.Name, &rm.Description, &rm.CreatedBy, &rm.CreatedAt, &rm.MemberCount)
	if errors.Is(err, sql.ErrNoRows) {
		return rm, ErrNotFound
	}
	return rm, err
}

// Join adds userID as a member of roomID (idempotent).
func (s *RoomStore) Join(roomID, userID int64) error {
	_, err := s.db.Exec(
		`INSERT INTO room_members (room_id, user_id) VALUES ($1, $2) ON CONFLICT (room_id, user_id) DO NOTHING`,
		roomID, userID,
	)
	return err
}

// IsMember reports whether userID has joined roomID.
func (s *RoomStore) IsMember(roomID, userID int64) (bool, error) {
	var exists int
	err := s.db.QueryRow(
		`SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2`,
		roomID, userID,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Post adds a message to a room's shared feed.
func (s *RoomStore) Post(roomID, userID int64, body string) error {
	_, err := s.db.Exec(
		`INSERT INTO room_posts (room_id, user_id, body) VALUES ($1, $2, $3)`,
		roomID, userID, body,
	)
	return err
}

// Feed returns every post in a room, oldest first, with author names joined in.
func (s *RoomStore) Feed(roomID int64) ([]models.RoomPost, error) {
	rows, err := s.db.Query(`
		SELECT rp.id, rp.room_id, rp.user_id, u.name, rp.body, rp.created_at
		FROM room_posts rp
		JOIN users u ON u.id = rp.user_id
		WHERE rp.room_id = $1
		ORDER BY rp.created_at ASC, rp.id ASC
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.RoomPost
	for rows.Next() {
		var p models.RoomPost
		if err := rows.Scan(&p.ID, &p.RoomID, &p.UserID, &p.AuthorName, &p.Body, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

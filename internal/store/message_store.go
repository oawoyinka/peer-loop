package store

import (
	"database/sql"

	"peerloop/internal/models"
)

type MessageStore struct {
	db *sql.DB
}

func NewMessageStore(db *sql.DB) *MessageStore {
	return &MessageStore{db: db}
}

// Send records a message from senderID to recipientID.
func (s *MessageStore) Send(senderID, recipientID int64, body string) error {
	_, err := s.db.Exec(
		`INSERT INTO messages (sender_id, recipient_id, body) VALUES ($1, $2, $3)`,
		senderID, recipientID, body,
	)
	return err
}

// Thread returns every message exchanged between userID and otherID,
// oldest first, with Mine set relative to userID.
func (s *MessageStore) Thread(userID, otherID int64) ([]models.Message, error) {
	rows, err := s.db.Query(`
		SELECT id, sender_id, recipient_id, body, created_at
		FROM messages
		WHERE (sender_id = $1 AND recipient_id = $2)
		   OR (sender_id = $2 AND recipient_id = $1)
		ORDER BY created_at ASC, id ASC
	`, userID, otherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Mine = m.SenderID == userID
		out = append(out, m)
	}
	return out, rows.Err()
}

// Conversations returns one row per person userID has exchanged messages
// with, newest conversation first, with a preview of the last message.
func (s *MessageStore) Conversations(userID int64) ([]models.ConversationSummary, error) {
	rows, err := s.db.Query(`
		WITH parties AS (
			SELECT recipient_id AS other_id FROM messages WHERE sender_id = $1
			UNION
			SELECT sender_id AS other_id FROM messages WHERE recipient_id = $1
		)
		SELECT
			u.id, u.name, u.email, u.bio,
			m.body, m.created_at, m.sender_id
		FROM parties p
		JOIN users u ON u.id = p.other_id
		JOIN messages m ON m.id = (
			SELECT id FROM messages
			WHERE (sender_id = $1 AND recipient_id = p.other_id)
			   OR (sender_id = p.other_id AND recipient_id = $1)
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		)
		ORDER BY m.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ConversationSummary
	for rows.Next() {
		var c models.ConversationSummary
		var senderID int64
		if err := rows.Scan(
			&c.OtherUser.ID, &c.OtherUser.Name, &c.OtherUser.Email, &c.OtherUser.Bio,
			&c.LastBody, &c.LastAt, &senderID,
		); err != nil {
			return nil, err
		}
		c.LastFromMe = senderID == userID
		out = append(out, c)
	}
	return out, rows.Err()
}

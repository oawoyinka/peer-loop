package store

import (
	"database/sql"

	"peerloop/internal/models"
)

type SkillStore struct {
	db *sql.DB
}

func NewSkillStore(db *sql.DB) *SkillStore {
	return &SkillStore{db: db}
}

// ListAll returns every skill in the system, alphabetically.
func (s *SkillStore) ListAll() ([]models.Skill, error) {
	rows, err := s.db.Query(`SELECT id, name FROM skills ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []models.Skill
	for rows.Next() {
		var sk models.Skill
		if err := rows.Scan(&sk.ID, &sk.Name); err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

// GetOrCreateByName looks up a skill by name, creating it if it doesn't
// exist yet (lets users add a skill that isn't in the seed list).
func (s *SkillStore) GetOrCreateByName(name string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM skills WHERE name = $1`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	err = s.db.QueryRow(`INSERT INTO skills (name) VALUES ($1) RETURNING id`, name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// SetUserSkill upserts a user's self-rated level (1-5) for a skill.
func (s *SkillStore) SetUserSkill(userID, skillID int64, level int) error {
	_, err := s.db.Exec(`
		INSERT INTO user_skills (user_id, skill_id, level)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, skill_id) DO UPDATE SET level = excluded.level
	`, userID, skillID, level)
	return err
}

// ForUser returns everything a given user has rated themselves on.
func (s *SkillStore) ForUser(userID int64) ([]models.UserSkill, error) {
	rows, err := s.db.Query(`
		SELECT us.user_id, us.skill_id, sk.name, us.level
		FROM user_skills us
		JOIN skills sk ON sk.id = us.skill_id
		WHERE us.user_id = $1
		ORDER BY sk.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.UserSkill
	for rows.Next() {
		var us models.UserSkill
		if err := rows.Scan(&us.UserID, &us.SkillID, &us.SkillName, &us.Level); err != nil {
			return nil, err
		}
		out = append(out, us)
	}
	return out, rows.Err()
}

// IsMatch reports whether userID and otherID share at least one rated
// skill, i.e. whether they're a legitimate match (used to gate features
// like chat to people you're actually matched with).
func (s *SkillStore) IsMatch(userID, otherID int64) (bool, error) {
	var exists int
	err := s.db.QueryRow(`
		SELECT 1 FROM user_skills a
		JOIN user_skills b ON a.skill_id = b.skill_id
		WHERE a.user_id = $1 AND b.user_id = $2
		LIMIT 1
	`, userID, otherID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// FindMatches is the core matching logic: for every skill the given user
// has rated themselves on, find other users rated on the same skill,
// ranked so the biggest level gaps (best teacher/learner pairings) come
// first. skillFilter narrows to one skill name; pass "" for all skills.
func (s *SkillStore) FindMatches(userID int64, skillFilter string) ([]models.PeerMatch, error) {
	query := `
		SELECT u.id, u.name, u.email, u.bio, sk.name, other.level, mine.level
		FROM user_skills mine
		JOIN user_skills other
			ON other.skill_id = mine.skill_id AND other.user_id != mine.user_id
		JOIN users u ON u.id = other.user_id
		JOIN skills sk ON sk.id = mine.skill_id
		WHERE mine.user_id = $1
	`
	args := []any{userID}
	if skillFilter != "" {
		query += ` AND sk.name = $2`
		args = append(args, skillFilter)
	}
	query += ` ORDER BY ABS(other.level - mine.level) DESC, sk.name`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []models.PeerMatch
	for rows.Next() {
		var m models.PeerMatch
		if err := rows.Scan(
			&m.User.ID, &m.User.Name, &m.User.Email, &m.User.Bio,
			&m.SkillName, &m.TheirLevel, &m.MyLevel,
		); err != nil {
			return nil, err
		}
		m.Gap = m.TheirLevel - m.MyLevel
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

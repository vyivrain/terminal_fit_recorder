package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// TrainerReview is one weekly review by the AI trainer: guidance for upcoming
// workout generation, written for a single profile.
type TrainerReview struct {
	ID        int
	ProfileID int
	Guidance  string
	CreatedAt time.Time
}

// GetLatestTrainerReview returns the active profile's most recent review, or
// nil when the trainer has not reviewed this profile yet.
func (db *DB) GetLatestTrainerReview() (*TrainerReview, error) {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return nil, err
	}

	var review TrainerReview
	err = db.conn.QueryRow(`
		SELECT id, profile_id, guidance, created_at
		FROM trainer_reviews
		WHERE profile_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, profileID).Scan(&review.ID, &review.ProfileID, &review.Guidance, &review.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// SaveTrainerReview stores new guidance for profileID. The profile is passed
// explicitly so a review started for one profile is never filed under another
// that became active while the AI was thinking.
func (db *DB) SaveTrainerReview(profileID int, guidance string) (*TrainerReview, error) {
	guidance = strings.TrimSpace(guidance)
	if guidance == "" {
		return nil, fmt.Errorf("trainer guidance cannot be empty")
	}

	review := TrainerReview{ProfileID: profileID, Guidance: guidance, CreatedAt: time.Now()}
	result, err := db.conn.Exec(
		`INSERT INTO trainer_reviews (profile_id, guidance, created_at) VALUES (?, ?, ?)`,
		review.ProfileID, review.Guidance, review.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	review.ID = int(id)
	return &review, nil
}

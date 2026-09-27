package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const DefaultProfileName = "mine"

type Profile struct {
	ID        int
	Name      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

func activeProfileID(queryer queryRower) (int, error) {
	var profileID int
	err := queryer.QueryRow(`SELECT id FROM profiles WHERE is_active = 1`).Scan(&profileID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("no active profile")
	}
	if err != nil {
		return 0, err
	}
	return profileID, nil
}

func (db *DB) GetActiveProfile() (*Profile, error) {
	var profile Profile
	err := db.conn.QueryRow(`
		SELECT id, name, is_active, created_at, updated_at
		FROM profiles
		WHERE is_active = 1
	`).Scan(&profile.ID, &profile.Name, &profile.IsActive, &profile.CreatedAt, &profile.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (db *DB) GetProfiles() ([]Profile, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, is_active, created_at, updated_at
		FROM profiles
		ORDER BY is_active DESC, name COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var profile Profile
		if err := rows.Scan(&profile.ID, &profile.Name, &profile.IsActive, &profile.CreatedAt, &profile.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	return profiles, rows.Err()
}

// CreateProfile creates an empty profile and makes it active. Existing data
// remains attached to its current profile.
func (db *DB) CreateProfile(name string) (*Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingID int
	err = tx.QueryRow(`SELECT id FROM profiles WHERE name = ? COLLATE NOCASE`, name).Scan(&existingID)
	if err == nil {
		return nil, fmt.Errorf("profile %q already exists", name)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	if _, err := tx.Exec(`UPDATE profiles SET is_active = 0, updated_at = ? WHERE is_active = 1`, time.Now()); err != nil {
		return nil, err
	}

	now := time.Now()
	result, err := tx.Exec(`
		INSERT INTO profiles (name, is_active, created_at, updated_at)
		VALUES (?, 1, ?, ?)
	`, name, now, now)
	if err != nil {
		return nil, err
	}

	profileID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Profile{
		ID:        int(profileID),
		Name:      name,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (db *DB) UseProfile(name string) (*Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var profile Profile
	err = tx.QueryRow(`
		SELECT id, name, is_active, created_at, updated_at
		FROM profiles
		WHERE name = ? COLLATE NOCASE
	`, name).Scan(&profile.ID, &profile.Name, &profile.IsActive, &profile.CreatedAt, &profile.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile %q does not exist", name)
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if _, err := tx.Exec(`UPDATE profiles SET is_active = 0, updated_at = ? WHERE is_active = 1`, now); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE profiles SET is_active = 1, updated_at = ? WHERE id = ?`, now, profile.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	profile.IsActive = true
	profile.UpdatedAt = now
	return &profile, nil
}

func (db *DB) RenameProfile(profileID int, name string) (*Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("profile name cannot be empty")
	}

	var duplicateID int
	err := db.conn.QueryRow(`SELECT id FROM profiles WHERE name = ? COLLATE NOCASE AND id != ?`, name, profileID).Scan(&duplicateID)
	if err == nil {
		return nil, fmt.Errorf("profile %q already exists", name)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	now := time.Now()
	result, err := db.conn.Exec(`UPDATE profiles SET name = ?, updated_at = ? WHERE id = ?`, name, now, profileID)
	if err != nil {
		return nil, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("profile %d does not exist", profileID)
	}

	var profile Profile
	err = db.conn.QueryRow(`
		SELECT id, name, is_active, created_at, updated_at
		FROM profiles
		WHERE id = ?
	`, profileID).Scan(&profile.ID, &profile.Name, &profile.IsActive, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// DeleteProfile removes a profile and all of its workout data. If it was the
// active profile, mine is preferred as the replacement, then the oldest one.
func (db *DB) DeleteProfile(profileID int) (*Profile, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var profileCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&profileCount); err != nil {
		return nil, err
	}
	if profileCount <= 1 {
		return nil, fmt.Errorf("cannot delete the only profile")
	}

	var wasActive bool
	err = tx.QueryRow(`SELECT is_active FROM profiles WHERE id = ?`, profileID).Scan(&wasActive)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile %d does not exist", profileID)
	}
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(`DELETE FROM exercises WHERE workout_id IN (SELECT id FROM workouts WHERE profile_id = ?)`, profileID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM workouts WHERE profile_id = ?`, profileID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM profiles WHERE id = ?`, profileID); err != nil {
		return nil, err
	}

	if wasActive {
		var replacementID int
		err := tx.QueryRow(`
			SELECT id FROM profiles
			ORDER BY CASE WHEN name = ? COLLATE NOCASE THEN 0 ELSE 1 END, id
			LIMIT 1
		`, DefaultProfileName).Scan(&replacementID)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`UPDATE profiles SET is_active = 1, updated_at = ? WHERE id = ?`, time.Now(), replacementID); err != nil {
			return nil, err
		}
	}

	var active Profile
	err = tx.QueryRow(`
		SELECT id, name, is_active, created_at, updated_at
		FROM profiles
		WHERE is_active = 1
	`).Scan(&active.ID, &active.Name, &active.IsActive, &active.CreatedAt, &active.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &active, nil
}

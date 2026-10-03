package db

import (
	"fmt"
	"time"
)

type Exercise struct {
	ID          int
	Name        string
	Weight      int     // Weight in kg (0 for bodyweight exercises)
	Repetitions int     // Number of repetitions (0 for duration-based exercises)
	Sets        int     // Number of sets
	Duration    float64 // Duration in minutes (required for exercises like planks, run, walk)
	Distance    int     // Distance in meters (required for run/walk exercises)
	// YoutubeURL is a short (<=60s) technique video found in the background
	// after the exercise is created. Empty until a lookup succeeds.
	YoutubeURL string
	WorkoutID  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DurationRequiredKeywords contains exercise name keywords that require duration input
var DurationRequiredKeywords = []string{"plank", "wall sit", "hold", "stretch"}

// DistanceRequiredKeywords contains exercise name keywords that require distance input
var DistanceRequiredKeywords = []string{"run", "walk", "cycling", "cycle", "swim", "rowing", "row"}

func (db *DB) GetAllExercises() ([]Exercise, error) {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT e.id, e.name, e.weight, e.repetitions, e.sets, e.duration, e.distance, e.youtube_url, e.created_at
		FROM exercises e
		JOIN workouts w ON w.id = e.workout_id
		WHERE w.profile_id = ?
		ORDER BY e.created_at DESC`

	rows, err := db.conn.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		err := rows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.Distance, &exercise.YoutubeURL, &exercise.CreatedAt)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, exercise)
	}

	return exercises, rows.Err()
}

// SetExerciseYoutubeURL records the technique video a background lookup
// found for an exercise. Scoped to the active profile like every other write.
func (db *DB) SetExerciseYoutubeURL(exerciseID int, url string) error {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return err
	}

	result, err := db.conn.Exec(`
		UPDATE exercises
		SET youtube_url = ?, updated_at = ?
		WHERE id = ? AND workout_id IN (SELECT id FROM workouts WHERE profile_id = ?)
	`, url, time.Now(), exerciseID, profileID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("exercise does not belong to the active profile")
	}
	return nil
}

// ExerciseRef identifies a single exercise row without loading its workout —
// enough for the catch-up video lookup command to act on.
type ExerciseRef struct {
	ID   int
	Name string
}

// GetExercisesMissingYoutubeURL lists exercises in the active profile that
// have not yet had a technique video found for them.
func (db *DB) GetExercisesMissingYoutubeURL() ([]ExerciseRef, error) {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return nil, err
	}

	rows, err := db.conn.Query(`
		SELECT e.id, e.name
		FROM exercises e
		JOIN workouts w ON w.id = e.workout_id
		WHERE w.profile_id = ? AND e.youtube_url = ''
		ORDER BY e.created_at`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []ExerciseRef
	for rows.Next() {
		var ref ExerciseRef
		if err := rows.Scan(&ref.ID, &ref.Name); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

func (db *DB) GetDistinctExerciseNames() ([]string, error) {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT DISTINCT e.name
		FROM exercises e
		JOIN workouts w ON w.id = e.workout_id
		WHERE w.profile_id = ?
		ORDER BY e.name`

	rows, err := db.conn.Query(query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	return names, rows.Err()
}

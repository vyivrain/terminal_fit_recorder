package db

import "time"

type Exercise struct {
	ID          int
	Name        string
	Weight      int     // Weight in kg (0 for bodyweight exercises)
	Repetitions int     // Number of repetitions (0 for duration-based exercises)
	Sets        int     // Number of sets
	Duration    float64 // Duration in minutes (required for exercises like planks, run, walk)
	Distance    int     // Distance in meters (required for run/walk exercises)
	WorkoutID   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
		SELECT e.id, e.name, e.weight, e.repetitions, e.sets, e.duration, e.distance, e.created_at
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
		err := rows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.Distance, &exercise.CreatedAt)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, exercise)
	}

	return exercises, rows.Err()
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

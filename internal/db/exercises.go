package db

import "time"

type Exercise struct {
	ID          int
	Name        string
	Weight      string
	Repetitions string
	Sets        string
	Duration    float64 // Duration in minutes (required for exercises like planks)
	WorkoutID   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DurationRequiredKeywords contains exercise name keywords that require duration input
var DurationRequiredKeywords = []string{"plank", "run", "walk"}

func (db *DB) GetAllExercises() ([]Exercise, error) {
	query := `SELECT id, name, weight, repetitions, sets, duration, created_at FROM exercises ORDER BY created_at DESC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		err := rows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.CreatedAt)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, exercise)
	}

	return exercises, rows.Err()
}

func (db *DB) GetDistinctExerciseNames() ([]string, error) {
	query := `SELECT DISTINCT name FROM exercises ORDER BY name`

	rows, err := db.conn.Query(query)
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

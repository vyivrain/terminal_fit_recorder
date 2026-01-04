package db

import (
	"database/sql"
	"fmt"
	"time"
)

type Workout struct {
	ID          int
	WorkoutType string
	WorkoutDate time.Time
	Status      string // "planned" or "completed"
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WorkoutWithExercises struct {
	Workout   Workout
	Exercises []Exercise
}

func (db *DB) GetAllWorkouts() ([]WorkoutWithExercises, error) {
	workoutsQuery := `SELECT id, workout_type, workout_date, status, created_at, updated_at FROM workouts ORDER BY workout_date DESC`

	rows, err := db.conn.Query(workoutsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workouts []WorkoutWithExercises
	for rows.Next() {
		var workout Workout
		var createdAt, updatedAt sql.NullTime
		err := rows.Scan(&workout.ID, &workout.WorkoutType, &workout.WorkoutDate, &workout.Status, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}

		// Handle NULL timestamps
		if createdAt.Valid {
			workout.CreatedAt = createdAt.Time
		} else {
			workout.CreatedAt = workout.WorkoutDate
		}
		if updatedAt.Valid {
			workout.UpdatedAt = updatedAt.Time
		} else {
			workout.UpdatedAt = workout.WorkoutDate
		}

		exercisesQuery := `SELECT id, name, weight, repetitions, sets, duration, workout_id, created_at, updated_at FROM exercises WHERE workout_id = ? ORDER BY created_at`
		exerciseRows, err := db.conn.Query(exercisesQuery, workout.ID)
		if err != nil {
			return nil, err
		}

		var exercises []Exercise
		for exerciseRows.Next() {
			var exercise Exercise
			err := exerciseRows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.WorkoutID, &exercise.CreatedAt, &exercise.UpdatedAt)
			if err != nil {
				exerciseRows.Close()
				return nil, err
			}
			exercises = append(exercises, exercise)
		}
		exerciseRows.Close()

		workouts = append(workouts, WorkoutWithExercises{
			Workout:   workout,
			Exercises: exercises,
		})
	}

	return workouts, rows.Err()
}

func (db *DB) GetLastWorkout() (*WorkoutWithExercises, error) {
	workoutQuery := `SELECT id, workout_type, workout_date, status, created_at, updated_at FROM workouts ORDER BY workout_date DESC LIMIT 1`

	var workout Workout
	var createdAt, updatedAt sql.NullTime
	err := db.conn.QueryRow(workoutQuery).Scan(&workout.ID, &workout.WorkoutType, &workout.WorkoutDate, &workout.Status, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle NULL timestamps
	if createdAt.Valid {
		workout.CreatedAt = createdAt.Time
	} else {
		workout.CreatedAt = workout.WorkoutDate
	}
	if updatedAt.Valid {
		workout.UpdatedAt = updatedAt.Time
	} else {
		workout.UpdatedAt = workout.WorkoutDate
	}

	exercisesQuery := `SELECT id, name, weight, repetitions, sets, duration, workout_id, created_at, updated_at FROM exercises WHERE workout_id = ? ORDER BY created_at`
	rows, err := db.conn.Query(exercisesQuery, workout.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		err := rows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.WorkoutID, &exercise.CreatedAt, &exercise.UpdatedAt)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, exercise)
	}

	return &WorkoutWithExercises{
		Workout:   workout,
		Exercises: exercises,
	}, rows.Err()
}

func (db *DB) SaveExercisesForWorkout(workoutID int64, exercises []Exercise) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
	INSERT INTO exercises (name, weight, repetitions, sets, duration, workout_id, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	for _, exercise := range exercises {
		createdAt := exercise.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}

		updatedAt := exercise.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}

		_, err := tx.Exec(query, exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration, workoutID, createdAt, updatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) CreateWorkout(workoutType string, status string, workoutDate ...time.Time) (int64, error) {
	now := time.Now()

	// Use provided date or default to now
	var date time.Time
	if len(workoutDate) > 0 && !workoutDate[0].IsZero() {
		date = workoutDate[0]
	} else {
		date = now
	}

	// Check if a workout already exists for the specified date
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM workouts WHERE DATE(workout_date) = DATE(?)`, date).Scan(&count)
	if err != nil {
		return 0, err
	}

	if count > 0 {
		return 0, fmt.Errorf("a workout already exists for today. Only one workout per day is allowed")
	}

	query := `
	INSERT INTO workouts (workout_type, workout_date, status, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?)`

	result, err := db.conn.Exec(query, workoutType, date, status, now, now)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (db *DB) SaveGeneratedWorkout(workout *WorkoutWithExercises) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert workout with planned status
	query := `INSERT INTO workouts (workout_type, workout_date, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	now := time.Now()
	result, err := tx.Exec(query, workout.Workout.WorkoutType, workout.Workout.WorkoutDate, "planned", now, now)
	if err != nil {
		return err
	}

	workoutID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Insert exercises
	exerciseQuery := `INSERT INTO exercises (name, weight, repetitions, sets, duration, workout_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	for _, exercise := range workout.Exercises {
		_, err := tx.Exec(exerciseQuery, exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration, workoutID, now, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) DeleteLastWorkout() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the last workout ID
	var workoutID int
	err = tx.QueryRow(`SELECT id FROM workouts ORDER BY workout_date DESC LIMIT 1`).Scan(&workoutID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No workouts to delete
		}
		return err
	}

	// Delete exercises associated with the workout
	_, err = tx.Exec(`DELETE FROM exercises WHERE workout_id = ?`, workoutID)
	if err != nil {
		return err
	}

	// Delete the workout
	_, err = tx.Exec(`DELETE FROM workouts WHERE id = ?`, workoutID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) GetWorkoutByDate(date time.Time) (*WorkoutWithExercises, error) {
	workoutQuery := `SELECT id, workout_type, workout_date, status, created_at, updated_at FROM workouts WHERE DATE(workout_date) = DATE(?) LIMIT 1`

	var workout Workout
	var createdAt, updatedAt sql.NullTime
	err := db.conn.QueryRow(workoutQuery, date).Scan(&workout.ID, &workout.WorkoutType, &workout.WorkoutDate, &workout.Status, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle NULL timestamps
	if createdAt.Valid {
		workout.CreatedAt = createdAt.Time
	} else {
		workout.CreatedAt = workout.WorkoutDate
	}
	if updatedAt.Valid {
		workout.UpdatedAt = updatedAt.Time
	} else {
		workout.UpdatedAt = workout.WorkoutDate
	}

	exercisesQuery := `SELECT id, name, weight, repetitions, sets, duration, workout_id, created_at, updated_at FROM exercises WHERE workout_id = ? ORDER BY created_at`
	rows, err := db.conn.Query(exercisesQuery, workout.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		err := rows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.WorkoutID, &exercise.CreatedAt, &exercise.UpdatedAt)
		if err != nil {
			return nil, err
		}
		exercises = append(exercises, exercise)
	}

	return &WorkoutWithExercises{
		Workout:   workout,
		Exercises: exercises,
	}, rows.Err()
}

func (db *DB) DeleteWorkoutByDate(date time.Time) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get workout ID for the specified date
	var workoutID int
	err = tx.QueryRow(`SELECT id FROM workouts WHERE DATE(workout_date) = DATE(?) LIMIT 1`, date).Scan(&workoutID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No workout found for this date
		}
		return err
	}

	// Delete exercises associated with the workout
	_, err = tx.Exec(`DELETE FROM exercises WHERE workout_id = ?`, workoutID)
	if err != nil {
		return err
	}

	// Delete the workout
	_, err = tx.Exec(`DELETE FROM workouts WHERE id = ?`, workoutID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) UpdateWorkout(workoutID int, workoutType string, exercises []Exercise) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update workout type and updated_at
	now := time.Now()
	_, err = tx.Exec(`UPDATE workouts SET workout_type = ?, updated_at = ? WHERE id = ?`, workoutType, now, workoutID)
	if err != nil {
		return err
	}

	// Delete existing exercises
	_, err = tx.Exec(`DELETE FROM exercises WHERE workout_id = ?`, workoutID)
	if err != nil {
		return err
	}

	// Insert new exercises
	query := `INSERT INTO exercises (name, weight, repetitions, sets, duration, workout_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	for _, exercise := range exercises {
		_, err := tx.Exec(query, exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration, workoutID, now, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) UpdateWorkoutDate(oldDate time.Time, newDate time.Time) error {
	// Check if a workout already exists on the new date
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM workouts WHERE DATE(workout_date) = DATE(?)`, newDate).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("a workout already exists for %s", newDate.Format("2006-01-02"))
	}

	// Update the workout date
	result, err := db.conn.Exec(`UPDATE workouts SET workout_date = ?, updated_at = ? WHERE DATE(workout_date) = DATE(?)`, newDate, time.Now(), oldDate)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no workout found for %s", oldDate.Format("2006-01-02"))
	}

	return nil
}

func (db *DB) UpdateLastWorkoutStatus(status string) error {
	// Update the status of the most recent workout
	result, err := db.conn.Exec(`
		UPDATE workouts
		SET status = ?, updated_at = ?
		WHERE id = (SELECT id FROM workouts ORDER BY workout_date DESC LIMIT 1)
	`, status, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no workouts found to update")
	}

	return nil
}

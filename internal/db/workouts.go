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
	// Notes is commentary on this particular workout — typically what the AI
	// generator changed, improved, or reduced and why, filled in automatically
	// when a generated workout is saved, or set externally (e.g. via Hermes).
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WorkoutWithExercises struct {
	Workout   Workout
	Exercises []Exercise
}

func (db *DB) GetAllWorkouts() ([]WorkoutWithExercises, error) {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return nil, err
	}

	workoutsQuery := `SELECT id, workout_type, workout_date, status, notes, created_at, updated_at FROM workouts WHERE profile_id = ? ORDER BY workout_date DESC`

	rows, err := db.conn.Query(workoutsQuery, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workouts []WorkoutWithExercises
	for rows.Next() {
		var workout Workout
		var createdAt, updatedAt sql.NullTime
		err := rows.Scan(&workout.ID, &workout.WorkoutType, &workout.WorkoutDate, &workout.Status, &workout.Notes, &createdAt, &updatedAt)
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

		exercisesQuery := `SELECT id, name, weight, repetitions, sets, duration, distance, youtube_url, workout_id, created_at, updated_at FROM exercises WHERE workout_id = ? ORDER BY created_at`
		exerciseRows, err := db.conn.Query(exercisesQuery, workout.ID)
		if err != nil {
			return nil, err
		}

		var exercises []Exercise
		for exerciseRows.Next() {
			var exercise Exercise
			err := exerciseRows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.Distance, &exercise.YoutubeURL, &exercise.WorkoutID, &exercise.CreatedAt, &exercise.UpdatedAt)
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
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return nil, err
	}

	workoutQuery := `SELECT id, workout_type, workout_date, status, notes, created_at, updated_at FROM workouts WHERE profile_id = ? ORDER BY workout_date DESC LIMIT 1`

	var workout Workout
	var createdAt, updatedAt sql.NullTime
	err = db.conn.QueryRow(workoutQuery, profileID).Scan(&workout.ID, &workout.WorkoutType, &workout.WorkoutDate, &workout.Status, &workout.Notes, &createdAt, &updatedAt)
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

	exercisesQuery := `SELECT id, name, weight, repetitions, sets, duration, distance, youtube_url, workout_id, created_at, updated_at FROM exercises WHERE workout_id = ? ORDER BY created_at`
	rows, err := db.conn.Query(exercisesQuery, workout.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		err := rows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.Distance, &exercise.YoutubeURL, &exercise.WorkoutID, &exercise.CreatedAt, &exercise.UpdatedAt)
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

	profileID, err := activeProfileID(tx)
	if err != nil {
		return err
	}

	var workoutExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM workouts WHERE id = ? AND profile_id = ?)`, workoutID, profileID).Scan(&workoutExists); err != nil {
		return err
	}
	if !workoutExists {
		return fmt.Errorf("workout does not belong to the active profile")
	}

	query := `
	INSERT INTO exercises (name, weight, repetitions, sets, duration, distance, workout_id, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	for index := range exercises {
		exercise := &exercises[index]
		createdAt := exercise.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}

		updatedAt := exercise.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}

		result, err := tx.Exec(query, exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration, exercise.Distance, workoutID, createdAt, updatedAt)
		if err != nil {
			return err
		}
		newID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		exercise.ID = int(newID)
		exercise.WorkoutID = int(workoutID)
	}

	return tx.Commit()
}

func (db *DB) SaveWorkoutsWithExercises(workouts []*WorkoutWithExercises, status string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	profileID, err := activeProfileID(tx)
	if err != nil {
		return err
	}

	seenDates := make(map[string]bool)
	for _, workout := range workouts {
		if workout == nil {
			return fmt.Errorf("workout cannot be nil")
		}

		dateKey := workout.Workout.WorkoutDate.Format("2006-01-02")
		if seenDates[dateKey] {
			return fmt.Errorf("file contains more than one workout for %s. Only one workout per day is allowed", dateKey)
		}
		seenDates[dateKey] = true

		var count int
		err := tx.QueryRow(`SELECT COUNT(*) FROM workouts WHERE profile_id = ? AND DATE(workout_date) = DATE(?)`, profileID, workout.Workout.WorkoutDate).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("a workout already exists for %s. Only one workout per day is allowed", dateKey)
		}
	}

	now := time.Now()
	workoutQuery := `INSERT INTO workouts (workout_type, workout_date, status, notes, profile_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	exerciseQuery := `INSERT INTO exercises (name, weight, repetitions, sets, duration, distance, youtube_url, workout_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	for _, workout := range workouts {
		workoutStatus := status
		if workoutStatus == "" {
			workoutStatus = workout.Workout.Status
		}
		if workoutStatus == "" {
			workoutStatus = "completed"
		}

		result, err := tx.Exec(workoutQuery, workout.Workout.WorkoutType, workout.Workout.WorkoutDate, workoutStatus, workout.Workout.Notes, profileID, now, now)
		if err != nil {
			return err
		}

		workoutID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		for index := range workout.Exercises {
			exercise := &workout.Exercises[index]
			result, err := tx.Exec(exerciseQuery, exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration, exercise.Distance, exercise.YoutubeURL, workoutID, now, now)
			if err != nil {
				return err
			}
			newID, err := result.LastInsertId()
			if err != nil {
				return err
			}
			exercise.ID = int(newID)
			exercise.WorkoutID = int(workoutID)
		}
	}

	return tx.Commit()
}

func (db *DB) CreateWorkout(workoutType string, status string, workoutDate ...time.Time) (int64, error) {
	now := time.Now()
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return 0, err
	}

	// Use provided date or default to now
	var date time.Time
	if len(workoutDate) > 0 && !workoutDate[0].IsZero() {
		date = workoutDate[0]
	} else {
		date = now
	}

	// Check if a workout already exists for the specified date
	var count int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM workouts WHERE profile_id = ? AND DATE(workout_date) = DATE(?)`, profileID, date).Scan(&count)
	if err != nil {
		return 0, err
	}

	if count > 0 {
		return 0, fmt.Errorf("a workout already exists for today. Only one workout per day is allowed")
	}

	query := `
	INSERT INTO workouts (workout_type, workout_date, status, profile_id, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)`

	result, err := db.conn.Exec(query, workoutType, date, status, profileID, now, now)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (db *DB) SaveGeneratedWorkout(workout *WorkoutWithExercises) error {
	return db.SaveWorkoutsWithExercises([]*WorkoutWithExercises{workout}, "planned")
}

func (db *DB) DeleteLastWorkout() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	profileID, err := activeProfileID(tx)
	if err != nil {
		return err
	}

	// Get the last workout ID
	var workoutID int
	err = tx.QueryRow(`SELECT id FROM workouts WHERE profile_id = ? ORDER BY workout_date DESC LIMIT 1`, profileID).Scan(&workoutID)
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
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return nil, err
	}

	workoutQuery := `SELECT id, workout_type, workout_date, status, notes, created_at, updated_at FROM workouts WHERE profile_id = ? AND DATE(workout_date) = DATE(?) LIMIT 1`

	var workout Workout
	var createdAt, updatedAt sql.NullTime
	err = db.conn.QueryRow(workoutQuery, profileID, date).Scan(&workout.ID, &workout.WorkoutType, &workout.WorkoutDate, &workout.Status, &workout.Notes, &createdAt, &updatedAt)
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

	exercisesQuery := `SELECT id, name, weight, repetitions, sets, duration, distance, youtube_url, workout_id, created_at, updated_at FROM exercises WHERE workout_id = ? ORDER BY created_at`
	rows, err := db.conn.Query(exercisesQuery, workout.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exercises []Exercise
	for rows.Next() {
		var exercise Exercise
		err := rows.Scan(&exercise.ID, &exercise.Name, &exercise.Weight, &exercise.Repetitions, &exercise.Sets, &exercise.Duration, &exercise.Distance, &exercise.YoutubeURL, &exercise.WorkoutID, &exercise.CreatedAt, &exercise.UpdatedAt)
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

	profileID, err := activeProfileID(tx)
	if err != nil {
		return err
	}

	// Get workout ID for the specified date
	var workoutID int
	err = tx.QueryRow(`SELECT id FROM workouts WHERE profile_id = ? AND DATE(workout_date) = DATE(?) LIMIT 1`, profileID, date).Scan(&workoutID)
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

	profileID, err := activeProfileID(tx)
	if err != nil {
		return err
	}

	// Update workout type and updated_at
	now := time.Now()
	result, err := tx.Exec(`UPDATE workouts SET workout_type = ?, updated_at = ? WHERE id = ? AND profile_id = ?`, workoutType, now, workoutID, profileID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("workout does not belong to the active profile")
	}

	// Delete existing exercises
	_, err = tx.Exec(`DELETE FROM exercises WHERE workout_id = ?`, workoutID)
	if err != nil {
		return err
	}

	// Insert new exercises
	query := `INSERT INTO exercises (name, weight, repetitions, sets, duration, distance, workout_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for index := range exercises {
		exercise := &exercises[index]
		result, err := tx.Exec(query, exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration, exercise.Distance, workoutID, now, now)
		if err != nil {
			return err
		}
		newID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		exercise.ID = int(newID)
		exercise.WorkoutID = workoutID
		exercise.YoutubeURL = ""
	}

	return tx.Commit()
}

func (db *DB) UpdateWorkoutDate(oldDate time.Time, newDate time.Time) error {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return err
	}

	// Check if a workout already exists on the new date
	var count int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM workouts WHERE profile_id = ? AND DATE(workout_date) = DATE(?)`, profileID, newDate).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return fmt.Errorf("a workout already exists for %s", newDate.Format("2006-01-02"))
	}

	// Update the workout date
	result, err := db.conn.Exec(`UPDATE workouts SET workout_date = ?, updated_at = ? WHERE profile_id = ? AND DATE(workout_date) = DATE(?)`, newDate, time.Now(), profileID, oldDate)
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
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return err
	}

	// Update the status of the most recent workout
	result, err := db.conn.Exec(`
		UPDATE workouts
		SET status = ?, updated_at = ?
		WHERE id = (
			SELECT id FROM workouts
			WHERE profile_id = ?
			ORDER BY workout_date DESC
			LIMIT 1
		)
	`, status, time.Now(), profileID)
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

// SetWorkoutNotes sets the commentary attached to a workout — typically what
// an AI generator changed, improved, or reduced and why. Hermes and other
// automations write here after a workout is planned or completed.
func (db *DB) SetWorkoutNotes(workoutID int, notes string) error {
	profileID, err := activeProfileID(db.conn)
	if err != nil {
		return err
	}

	result, err := db.conn.Exec(`UPDATE workouts SET notes = ?, updated_at = ? WHERE id = ? AND profile_id = ?`, notes, time.Now(), workoutID, profileID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("workout does not belong to the active profile")
	}

	return nil
}

package db_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

func inProjectRoot(t *testing.T) {
	t.Helper()

	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	projectRoot := filepath.Clean(filepath.Join(workingDirectory, "../.."))
	require.NoError(t, os.Chdir(projectRoot))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(workingDirectory))
	})
}

func TestProfilesIsolateWorkoutAndExerciseData(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "profiles.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	active, err := database.GetActiveProfile()
	require.NoError(t, err)
	require.Equal(t, db.DefaultProfileName, active.Name)

	workoutDate := time.Date(2026, time.September, 6, 10, 0, 0, 0, time.UTC)
	mineWorkoutID, err := database.CreateWorkout("strength", "completed", workoutDate)
	require.NoError(t, err)
	require.NoError(t, database.SaveExercisesForWorkout(mineWorkoutID, []db.Exercise{{Name: "Bench Press", Weight: 80, Repetitions: 8, Sets: 3}}))

	guest, err := database.CreateProfile("guest")
	require.NoError(t, err)
	require.True(t, guest.IsActive)

	workouts, err := database.GetAllWorkouts()
	require.NoError(t, err)
	require.Empty(t, workouts)
	names, err := database.GetDistinctExerciseNames()
	require.NoError(t, err)
	require.Empty(t, names)

	guestWorkoutID, err := database.CreateWorkout("cardio", "completed", workoutDate)
	require.NoError(t, err, "the same workout date is allowed in another profile")
	require.NoError(t, database.SaveExercisesForWorkout(guestWorkoutID, []db.Exercise{{Name: "Run", Distance: 5000}}))
	require.Error(t, database.SaveExercisesForWorkout(mineWorkoutID, []db.Exercise{{Name: "Should Not Save"}}))
	require.Error(t, database.UpdateWorkout(int(mineWorkoutID), "cardio", []db.Exercise{{Name: "Should Not Replace"}}))

	_, err = database.CreateWorkout("cardio", "completed", workoutDate)
	require.Error(t, err, "the same profile still allows only one workout per day")
	require.NoError(t, database.DeleteWorkoutByDate(workoutDate))
	workouts, err = database.GetAllWorkouts()
	require.NoError(t, err)
	require.Empty(t, workouts)

	_, err = database.UseProfile(db.DefaultProfileName)
	require.NoError(t, err)
	workouts, err = database.GetAllWorkouts()
	require.NoError(t, err)
	require.Len(t, workouts, 1)
	require.Equal(t, "strength", workouts[0].Workout.WorkoutType)
	require.Equal(t, "Bench Press", workouts[0].Exercises[0].Name)
	names, err = database.GetDistinctExerciseNames()
	require.NoError(t, err)
	require.Equal(t, []string{"Bench Press"}, names)

	_, err = database.CreateProfile("GUEST")
	require.ErrorContains(t, err, "already exists")
	active, err = database.GetActiveProfile()
	require.NoError(t, err)
	require.Equal(t, db.DefaultProfileName, active.Name, "a failed create must not change the active profile")
}

func TestProfileMigrationAssignsExistingHistoryToMine(t *testing.T) {
	inProjectRoot(t)

	databasePath := filepath.Join(t.TempDir(), "existing.sqlite")
	legacy, err := sql.Open("sqlite3", databasePath)
	require.NoError(t, err)

	for version := 1; version <= 7; version++ {
		matches, err := filepath.Glob(fmt.Sprintf("migrations/%03d_*.up.sql", version))
		require.NoError(t, err)
		require.Len(t, matches, 1)

		migration, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		_, err = legacy.Exec(string(migration))
		require.NoError(t, err)
	}

	_, err = legacy.Exec(`
		INSERT INTO workouts (workout_type, workout_date, status, created_at, updated_at)
		VALUES ('strength', '2026-09-01', 'completed', '2026-09-01', '2026-09-01');
		INSERT INTO exercises (name, weight, repetitions, sets, workout_id, created_at, updated_at)
		VALUES ('Squat', 100, 5, 5, last_insert_rowid(), '2026-09-01', '2026-09-01');
		CREATE TABLE schema_migrations (version uint64, dirty bool);
		INSERT INTO schema_migrations (version, dirty) VALUES (7, 0);
	`)
	require.NoError(t, err)
	require.NoError(t, legacy.Close())

	database, err := db.New(databasePath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	active, err := database.GetActiveProfile()
	require.NoError(t, err)
	require.Equal(t, db.DefaultProfileName, active.Name)

	workouts, err := database.GetAllWorkouts()
	require.NoError(t, err)
	require.Len(t, workouts, 1)
	require.Equal(t, "Squat", workouts[0].Exercises[0].Name)

	var unassigned int
	require.NoError(t, database.GetConn().QueryRow(`SELECT COUNT(*) FROM workouts WHERE profile_id IS NULL`).Scan(&unassigned))
	require.Zero(t, unassigned)
}

func TestProfileMigrationCanRollBack(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "rollback.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	_, err = database.CreateWorkout("strength", "completed")
	require.NoError(t, err)

	downMigration, err := os.ReadFile("migrations/008_add_profiles.down.sql")
	require.NoError(t, err)
	_, err = database.GetConn().Exec(string(downMigration))
	require.NoError(t, err)

	var profileTables int
	require.NoError(t, database.GetConn().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'profiles'`).Scan(&profileTables))
	require.Zero(t, profileTables)

	var profileColumns int
	require.NoError(t, database.GetConn().QueryRow(`SELECT COUNT(*) FROM pragma_table_info('workouts') WHERE name = 'profile_id'`).Scan(&profileColumns))
	require.Zero(t, profileColumns)
}

func TestProfileRenameAndDeleteManageOwnedWorkouts(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "profile-management.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	guest, err := database.CreateProfile("guest")
	require.NoError(t, err)
	workoutID, err := database.CreateWorkout("cardio", "completed")
	require.NoError(t, err)
	require.NoError(t, database.SaveExercisesForWorkout(workoutID, []db.Exercise{{Name: "Run", Distance: 5000}}))

	renamed, err := database.RenameProfile(guest.ID, "training partner")
	require.NoError(t, err)
	require.Equal(t, "training partner", renamed.Name)
	require.True(t, renamed.IsActive)

	replacement, err := database.DeleteProfile(guest.ID)
	require.NoError(t, err)
	require.Equal(t, db.DefaultProfileName, replacement.Name)
	require.True(t, replacement.IsActive)

	var workouts int
	require.NoError(t, database.GetConn().QueryRow(`SELECT COUNT(*) FROM workouts WHERE profile_id = ?`, guest.ID).Scan(&workouts))
	require.Zero(t, workouts)
	var exercises int
	require.NoError(t, database.GetConn().QueryRow(`SELECT COUNT(*) FROM exercises WHERE workout_id = ?`, workoutID).Scan(&exercises))
	require.Zero(t, exercises)

	_, err = database.DeleteProfile(replacement.ID)
	require.ErrorContains(t, err, "only profile")
}

func TestGeneratedWorkoutPreservesPerProfileDateInvariant(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "generated-date.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	workoutDate := time.Date(2026, time.September, 6, 0, 0, 0, 0, time.UTC)
	_, err = database.CreateWorkout("strength", "completed", workoutDate)
	require.NoError(t, err)

	generated := &db.WorkoutWithExercises{
		Workout:   db.Workout{WorkoutType: "cardio", WorkoutDate: workoutDate},
		Exercises: []db.Exercise{{Name: "Run", Duration: 30}},
	}
	require.ErrorContains(t, database.SaveGeneratedWorkout(generated), "already exists")

	_, err = database.CreateProfile("partner")
	require.NoError(t, err)
	require.NoError(t, database.SaveGeneratedWorkout(generated), "the date is available in another profile")

	workouts, err := database.GetAllWorkouts()
	require.NoError(t, err)
	require.Len(t, workouts, 1)
	require.Equal(t, "planned", workouts[0].Workout.Status)
}

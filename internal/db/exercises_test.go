package db_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

func TestSaveExercisesForWorkoutAssignsIDs(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "exercise-ids.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	workoutID, err := database.CreateWorkout("strength", "completed", time.Now())
	require.NoError(t, err)

	exercises := []db.Exercise{{Name: "Squat", Weight: 100, Repetitions: 5, Sets: 5}}
	require.NoError(t, database.SaveExercisesForWorkout(workoutID, exercises))
	require.NotZero(t, exercises[0].ID, "SaveExercisesForWorkout must assign the new row's id in place")
	require.Equal(t, int(workoutID), exercises[0].WorkoutID)
}

func TestSaveWorkoutsWithExercisesAssignsIDs(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "workout-exercise-ids.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	workout := &db.WorkoutWithExercises{
		Workout: db.Workout{WorkoutType: "strength", WorkoutDate: time.Now()},
		Exercises: []db.Exercise{{
			Name: "Deadlift", Weight: 120, Repetitions: 5, Sets: 3,
			YoutubeURL: "https://www.youtube.com/watch?v=deadlift",
		}},
	}
	require.NoError(t, database.SaveWorkoutsWithExercises([]*db.WorkoutWithExercises{workout}, "planned"))
	require.NotZero(t, workout.Exercises[0].ID, "SaveWorkoutsWithExercises must assign each exercise's id in place")

	saved, err := database.GetLastWorkout()
	require.NoError(t, err)
	require.Equal(t, "https://www.youtube.com/watch?v=deadlift", saved.Exercises[0].YoutubeURL)
}

func TestSetExerciseYoutubeURLIsScopedToActiveProfile(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "exercise-video.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	workoutID, err := database.CreateWorkout("strength", "completed", time.Now())
	require.NoError(t, err)
	exercises := []db.Exercise{{Name: "Squat", Weight: 100, Repetitions: 5, Sets: 5}}
	require.NoError(t, database.SaveExercisesForWorkout(workoutID, exercises))

	require.NoError(t, database.SetExerciseYoutubeURL(exercises[0].ID, "https://www.youtube.com/watch?v=abc123"))

	all, err := database.GetAllExercises()
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, "https://www.youtube.com/watch?v=abc123", all[0].YoutubeURL)

	_, err = database.CreateProfile("guest")
	require.NoError(t, err)
	err = database.SetExerciseYoutubeURL(exercises[0].ID, "https://www.youtube.com/watch?v=shouldnotapply")
	require.ErrorContains(t, err, "does not belong to the active profile")
}

func TestGetExercisesMissingYoutubeURL(t *testing.T) {
	inProjectRoot(t)

	database, err := db.New(filepath.Join(t.TempDir(), "exercise-missing-video.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	workoutID, err := database.CreateWorkout("strength", "completed", time.Now())
	require.NoError(t, err)
	exercises := []db.Exercise{
		{Name: "Squat", Weight: 100, Repetitions: 5, Sets: 5},
		{Name: "Bench Press", Weight: 80, Repetitions: 8, Sets: 3},
	}
	require.NoError(t, database.SaveExercisesForWorkout(workoutID, exercises))
	require.NoError(t, database.SetExerciseYoutubeURL(exercises[0].ID, "https://www.youtube.com/watch?v=abc123"))

	missing, err := database.GetExercisesMissingYoutubeURL()
	require.NoError(t, err)
	require.Len(t, missing, 1)
	require.Equal(t, "Bench Press", missing[0].Name)
}

package commands

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/testutil"
)

// ShowExerciseTestSuite embeds the shared DBTestSuite
type ShowExerciseTestSuite struct {
	testutil.DBTestSuite
}

func (s *ShowExerciseTestSuite) TestShowLastWorkoutCommand_Name() {
	cmd := NewShowLastWorkoutCommand()
	assert.Equal(s.T(), "show last workout", cmd.Name())
}

func (s *ShowExerciseTestSuite) TestShowLastWorkoutCommand_Validate() {
	cmd := NewShowLastWorkoutCommand()
	assert.NoError(s.T(), cmd.Validate())
}

func (s *ShowExerciseTestSuite) TestShowLastWorkoutCommand_HelpManual() {
	cmd := NewShowLastWorkoutCommand()
	help := cmd.HelpManual()

	assert.NotEmpty(s.T(), help)
	assert.Contains(s.T(), help, "terminal_fit_recorder")
	assert.Contains(s.T(), help, "exercise")
	assert.Contains(s.T(), help, "last")
}

func (s *ShowExerciseTestSuite) TestShowAllWorkoutsCommand_Name() {
	cmd := NewShowAllWorkoutsCommand()
	assert.Equal(s.T(), "show all workouts", cmd.Name())
}

func (s *ShowExerciseTestSuite) TestShowAllWorkoutsCommand_Validate() {
	cmd := NewShowAllWorkoutsCommand()
	assert.NoError(s.T(), cmd.Validate())
}

func (s *ShowExerciseTestSuite) TestShowAllWorkoutsCommand_HelpManual() {
	cmd := NewShowAllWorkoutsCommand()
	help := cmd.HelpManual()

	assert.NotEmpty(s.T(), help)
	assert.Contains(s.T(), help, "terminal_fit_recorder")
	assert.Contains(s.T(), help, "exercise")
	assert.Contains(s.T(), help, "all")
}

func (s *ShowExerciseTestSuite) TestFormatWorkout_WithNilWorkout() {
	result := FormatWorkout(nil)
	assert.Equal(s.T(), "No workout data", result)
}

func (s *ShowExerciseTestSuite) TestFormatWorkout_WithEmptyExercises() {
	workout := &db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: "strength",
			WorkoutDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Status:      "completed",
		},
		Exercises: []db.Exercise{},
	}

	result := FormatWorkout(workout)

	assert.Contains(s.T(), result, "Workout Type: strength")
	assert.Contains(s.T(), result, "2024-01-15")
	assert.Contains(s.T(), result, "Status: completed")
	assert.Contains(s.T(), result, "Exercise")
	assert.Contains(s.T(), result, "Weight")
}

func (s *ShowExerciseTestSuite) TestFormatWorkout_WithExercises() {
	workout := &db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: "strength",
			WorkoutDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Status:      "completed",
		},
		Exercises: []db.Exercise{
			{Name: "Bench Press", Weight: 100, Repetitions: 10, Sets: 3},
			{Name: "Squats", Weight: 120, Repetitions: 8, Sets: 4},
			{Name: "Push-ups", Weight: 0, Repetitions: 20, Sets: 3}, // bodyweight
		},
	}

	result := FormatWorkout(workout)

	// Check header
	assert.Contains(s.T(), result, "Workout Type: strength")
	assert.Contains(s.T(), result, "2024-01-15")
	assert.Contains(s.T(), result, "Status: completed")

	// Check exercises
	assert.Contains(s.T(), result, "Bench Press")
	assert.Contains(s.T(), result, "100 kg")
	assert.Contains(s.T(), result, "Squats")
	assert.Contains(s.T(), result, "120 kg")
	assert.Contains(s.T(), result, "Push-ups")

	// Check that bodyweight exercise shows "-" for weight
	lines := strings.Split(result, "\n")
	var pushupLine string
	for _, line := range lines {
		if strings.Contains(line, "Push-ups") {
			pushupLine = line
			break
		}
	}
	assert.Contains(s.T(), pushupLine, "-")
}

func (s *ShowExerciseTestSuite) TestFormatWorkout_WeightFormatting() {
	workout := &db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: "strength",
			WorkoutDate: time.Now(),
			Status:      "completed",
		},
		Exercises: []db.Exercise{
			{Name: "Heavy Lift", Weight: 150, Repetitions: 5, Sets: 5},
			{Name: "Light Lift", Weight: 25, Repetitions: 15, Sets: 3},
			{Name: "Bodyweight", Weight: 0, Repetitions: 10, Sets: 3},
		},
	}

	result := FormatWorkout(workout)

	// Verify weight formatting
	assert.Contains(s.T(), result, "150 kg")
	assert.Contains(s.T(), result, "25 kg")

	// Find bodyweight exercise line and verify it shows "-"
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Bodyweight") {
			// Split by whitespace and check that weight column has "-"
			assert.Contains(s.T(), line, "-")
			break
		}
	}
}

func (s *ShowExerciseTestSuite) TestDatabaseIntegration_GetLastWorkout() {
	// Create two workouts
	workoutID1, err := s.DB.CreateWorkout("strength", "completed")
	require.NoError(s.T(), err)

	exercises1 := []db.Exercise{
		{Name: "Bench Press", Weight: 100, Repetitions: 10, Sets: 3},
	}
	err = s.DB.SaveExercisesForWorkout(workoutID1, exercises1)
	require.NoError(s.T(), err, "Failed to save exercises for first workout")

	// Create second workout (will be the "last" one)
	s.CleanupSubtest()

	workoutID2, err := s.DB.CreateWorkout("cardio", "completed")
	require.NoError(s.T(), err)

	exercises2 := []db.Exercise{
		{Name: "Running", Weight: 0, Repetitions: 1, Sets: 1, Duration: 30.0},
	}
	err = s.DB.SaveExercisesForWorkout(workoutID2, exercises2)
	require.NoError(s.T(), err)

	// Get last workout
	lastWorkout, err := s.DB.GetLastWorkout()
	require.NoError(s.T(), err)
	require.NotNil(s.T(), lastWorkout)

	// Verify it's the cardio workout
	assert.Equal(s.T(), "cardio", lastWorkout.Workout.WorkoutType)
	assert.Len(s.T(), lastWorkout.Exercises, 1)
	assert.Equal(s.T(), "Running", lastWorkout.Exercises[0].Name)
}

func (s *ShowExerciseTestSuite) TestDatabaseIntegration_GetLastWorkout_NoWorkouts() {
	workout, err := s.DB.GetLastWorkout()
	require.NoError(s.T(), err)
	assert.Nil(s.T(), workout)
}

func (s *ShowExerciseTestSuite) TestDatabaseIntegration_GetAllWorkouts() {
	// Create multiple workouts on different dates
	yesterday := time.Now().AddDate(0, 0, -1)
	today := time.Now()

	// Create first workout (yesterday)
	workoutID1, err := s.DB.CreateWorkout("strength", "completed", yesterday)
	require.NoError(s.T(), err, "Failed to create first workout")

	exercises1 := []db.Exercise{
		{Name: "Bench Press", Weight: 100, Repetitions: 10, Sets: 3},
	}
	err = s.DB.SaveExercisesForWorkout(workoutID1, exercises1)
	require.NoError(s.T(), err, "Failed to save exercises for first workout")

	// Create second workout (today)
	workoutID2, err := s.DB.CreateWorkout("cardio", "planned", today)
	require.NoError(s.T(), err, "Failed to create second workout")

	exercises2 := []db.Exercise{
		{Name: "Running", Weight: 0, Repetitions: 1, Sets: 1, Duration: 30.0},
	}
	err = s.DB.SaveExercisesForWorkout(workoutID2, exercises2)
	require.NoError(s.T(), err, "Failed to save exercises for second workout")

	// Get all workouts
	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err, "Failed to get all workouts")

	assert.Len(s.T(), workouts, 2)

	// Workouts are ordered by date DESC, so cardio (today) should be first
	assert.Equal(s.T(), "cardio", workouts[0].Workout.WorkoutType)
	assert.Equal(s.T(), "strength", workouts[1].Workout.WorkoutType)
}

func (s *ShowExerciseTestSuite) TestDatabaseIntegration_GetAllWorkouts_NoWorkouts() {
	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Empty(s.T(), workouts)
}

// TestShowExerciseCommand runs the test suite
func TestShowExerciseCommand(t *testing.T) {
	suite.Run(t, new(ShowExerciseTestSuite))
}

package commands

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/testutil"
)

// SaveExerciseTestSuite embeds the shared DBTestSuite
type SaveExerciseTestSuite struct {
	testutil.DBTestSuite
}

func (s *SaveExerciseTestSuite) TestName() {
	cmd := NewSaveExerciseCommand()
	assert.Equal(s.T(), "save exercise", cmd.Name())
}

func (s *SaveExerciseTestSuite) TestValidate() {
	cmd := NewSaveExerciseCommand()
	assert.NoError(s.T(), cmd.Validate())
}

func (s *SaveExerciseTestSuite) TestHelpManual() {
	cmd := NewSaveExerciseCommand()
	help := cmd.HelpManual()

	assert.NotEmpty(s.T(), help, "HelpManual should return non-empty string")

	// Verify help text contains expected components
	expectedParts := []string{
		"terminal_fit_recorder",
		"exercise",
		"save",
		"interactive",
		"workout",
	}

	for _, part := range expectedParts {
		assert.Contains(s.T(), help, part, "HelpManual should contain %q", part)
	}
}

func (s *SaveExerciseTestSuite) TestNewSaveExerciseCommand() {
	cmd := NewSaveExerciseCommand()
	assert.NotNil(s.T(), cmd, "NewSaveExerciseCommand should return non-nil instance")
}

func (s *SaveExerciseTestSuite) TestDurationRequiredKeywords() {
	// This test verifies the db.DurationRequiredKeywords that SaveExerciseCommand depends on
	assert.NotEmpty(s.T(), db.DurationRequiredKeywords, "DurationRequiredKeywords should not be empty")

	expectedKeywords := []string{"plank", "run", "walk"}
	assert.ElementsMatch(s.T(), expectedKeywords, db.DurationRequiredKeywords,
		"DurationRequiredKeywords should contain expected keywords")
}

func (s *SaveExerciseTestSuite) TestDatabaseIntegration_GetDistinctExerciseNames() {
	// Create a workout first
	workoutID, err := s.DB.CreateWorkout("strength", "completed")
	require.NoError(s.T(), err, "Failed to create workout")

	// Save some exercises
	exercises := []db.Exercise{
		{Name: "Bench Press", Weight: 100, Repetitions: 10, Sets: 3},
		{Name: "Squats", Weight: 120, Repetitions: 8, Sets: 4},
		{Name: "Bench Press", Weight: 105, Repetitions: 8, Sets: 3}, // Duplicate name
	}

	err = s.DB.SaveExercisesForWorkout(workoutID, exercises)
	require.NoError(s.T(), err, "Failed to save exercises")

	// Test GetDistinctExerciseNames (used by SaveExerciseCommand for autocomplete)
	names, err := s.DB.GetDistinctExerciseNames()
	require.NoError(s.T(), err, "Failed to get distinct exercise names")

	assert.Len(s.T(), names, 2, "Should have 2 distinct exercise names")
	assert.Contains(s.T(), names, "Bench Press")
	assert.Contains(s.T(), names, "Squats")
}

func (s *SaveExerciseTestSuite) TestDatabaseIntegration_CreateWorkout() {
	tests := []struct {
		name        string
		workoutType string
		status      string
	}{
		{
			name:        "create strength workout",
			workoutType: "strength",
			status:      "completed",
		},
		{
			name:        "create cardio workout",
			workoutType: "cardio",
			status:      "completed",
		},
	}

	for _, tt := range tests {
		// Clean up before each subtest (except the first one)
		s.CleanupSubtest()

		s.Run(tt.name, func() {
			workoutID, err := s.DB.CreateWorkout(tt.workoutType, tt.status)

			assert.NoError(s.T(), err)
			assert.Greater(s.T(), workoutID, int64(0), "Workout ID should be positive")
		})
	}
}

func (s *SaveExerciseTestSuite) TestDatabaseIntegration_SaveExercisesWithDuration() {
	// Create a workout
	workoutID, err := s.DB.CreateWorkout("strength", "completed")
	require.NoError(s.T(), err, "Failed to create workout")

	// Save exercises including one with duration
	exercises := []db.Exercise{
		{Name: "Plank", Weight: 0, Repetitions: 1, Sets: 3, Duration: 2.5},   // 2.5 minutes, bodyweight
		{Name: "Push-ups", Weight: 0, Repetitions: 20, Sets: 3, Duration: 0}, // bodyweight
	}

	err = s.DB.SaveExercisesForWorkout(workoutID, exercises)
	require.NoError(s.T(), err, "Failed to save exercises")

	// Verify the exercises were saved correctly
	workout, err := s.DB.GetLastWorkout()
	require.NoError(s.T(), err, "Failed to get last workout")
	require.NotNil(s.T(), workout, "Workout should not be nil")

	assert.Len(s.T(), workout.Exercises, 2, "Should have 2 exercises")

	// Find the plank exercise
	var plankExercise *db.Exercise
	for i := range workout.Exercises {
		if workout.Exercises[i].Name == "Plank" {
			plankExercise = &workout.Exercises[i]
			break
		}
	}

	require.NotNil(s.T(), plankExercise, "Plank exercise should exist")
	assert.Equal(s.T(), 2.5, plankExercise.Duration, "Plank duration should be 2.5 minutes")
}

func (s *SaveExerciseTestSuite) TestDatabaseIntegration_OneWorkoutPerDay() {
	// Create first workout
	_, err := s.DB.CreateWorkout("strength", "completed")
	require.NoError(s.T(), err, "First workout should be created successfully")

	// Try to create second workout on the same day
	_, err = s.DB.CreateWorkout("cardio", "completed")
	assert.Error(s.T(), err, "Should not allow second workout on the same day")
	assert.Contains(s.T(), err.Error(), "already exists for today",
		"Error should mention workout already exists for today")
}

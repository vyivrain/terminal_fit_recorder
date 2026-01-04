package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/testutil"
)

// GenerateCommandTestSuite embeds the shared DBTestSuite
type GenerateCommandTestSuite struct {
	testutil.DBTestSuite
}

// Test GenerateCommandWrapper routing and validation

func (s *GenerateCommandTestSuite) TestGenerateCommandWrapper_Name() {
	cmd := NewGenerateCommandWrapper([]string{})
	assert.Equal(s.T(), "generate", cmd.Name())
}

func (s *GenerateCommandTestSuite) TestGenerateCommandWrapper_HelpManual() {
	cmd := NewGenerateCommandWrapper([]string{})
	helpText := cmd.HelpManual()
	assert.Contains(s.T(), helpText, "generate")
	assert.Contains(s.T(), helpText, "AI")
}

func (s *GenerateCommandTestSuite) TestGenerateCommandWrapper_Validate_NoCount() {
	cmd := NewGenerateCommandWrapper([]string{"terminal_fit_recorder", "exercise", "generate"})
	err := cmd.Validate()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cmd.command)
	assert.Equal(s.T(), 0, cmd.command.ExerciseCount) // 0 means AI decides
}

func (s *GenerateCommandTestSuite) TestGenerateCommandWrapper_Validate_WithValidCount() {
	cmd := NewGenerateCommandWrapper([]string{"terminal_fit_recorder", "exercise", "generate", "5"})
	err := cmd.Validate()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cmd.command)
	assert.Equal(s.T(), 5, cmd.command.ExerciseCount)
}

func (s *GenerateCommandTestSuite) TestGenerateCommandWrapper_Validate_InvalidCount() {
	cmd := NewGenerateCommandWrapper([]string{"terminal_fit_recorder", "exercise", "generate", "invalid"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid exercise count")
}

func (s *GenerateCommandTestSuite) TestGenerateCommandWrapper_Validate_CountTooLow() {
	cmd := NewGenerateCommandWrapper([]string{"terminal_fit_recorder", "exercise", "generate", "0"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "between 1 and 20")
}

func (s *GenerateCommandTestSuite) TestGenerateCommandWrapper_Validate_CountTooHigh() {
	cmd := NewGenerateCommandWrapper([]string{"terminal_fit_recorder", "exercise", "generate", "21"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "between 1 and 20")
}

// Test GenerateCommand

func (s *GenerateCommandTestSuite) TestGenerateCommand_Name() {
	cmd := NewGenerateCommand(5)
	assert.Equal(s.T(), "generate", cmd.Name())
}

func (s *GenerateCommandTestSuite) TestGenerateCommand_Validate_Valid() {
	testCases := []struct {
		name  string
		count int
	}{
		{"Zero count (AI decides)", 0},
		{"Minimum count", 1},
		{"Middle count", 10},
		{"Maximum count", 20},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := NewGenerateCommand(tc.count)
			err := cmd.Validate()
			assert.NoError(s.T(), err)
		})
	}
}

func (s *GenerateCommandTestSuite) TestGenerateCommand_Validate_Invalid() {
	testCases := []struct {
		name  string
		count int
	}{
		{"Negative count", -1},
		{"Count too high", 21},
		{"Count way too high", 100},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := NewGenerateCommand(tc.count)
			err := cmd.Validate()
			assert.Error(s.T(), err)
			assert.Contains(s.T(), err.Error(), "exercise count must be between")
		})
	}
}

func (s *GenerateCommandTestSuite) TestGenerateCommand_HelpManual() {
	cmd := NewGenerateCommand(5)
	_ = cmd.HelpManual() // Just verify it doesn't panic
}

// Test ParseWorkoutResponse with various JSON formats

func (s *GenerateCommandTestSuite) TestParseWorkoutResponse_ValidJSON() {
	jsonResponse := `{
		"type": "strength",
		"date": "2025-01-15",
		"exercises": [
			{
				"name": "Bench Press",
				"weight": 85,
				"reps": 10,
				"sets": 3,
				"duration": 0
			},
			{
				"name": "Squats",
				"weight": 105,
				"reps": 8,
				"sets": 4,
				"duration": 0
			}
		]
	}`

	workout, err := api.ParseWorkoutResponse(jsonResponse)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout)

	assert.Equal(s.T(), "strength", workout.Workout.WorkoutType)
	assert.Equal(s.T(), "2025-01-15", workout.Workout.WorkoutDate.Format("2006-01-02"))
	assert.Len(s.T(), workout.Exercises, 2)

	// Check first exercise
	assert.Equal(s.T(), "Bench Press", workout.Exercises[0].Name)
	assert.Equal(s.T(), 85, workout.Exercises[0].Weight)
	assert.Equal(s.T(), 10, workout.Exercises[0].Repetitions)
	assert.Equal(s.T(), 3, workout.Exercises[0].Sets)
	assert.Equal(s.T(), 0.0, workout.Exercises[0].Duration)

	// Check second exercise
	assert.Equal(s.T(), "Squats", workout.Exercises[1].Name)
	assert.Equal(s.T(), 105, workout.Exercises[1].Weight)
}

func (s *GenerateCommandTestSuite) TestParseWorkoutResponse_WithDuration() {
	jsonResponse := `{
		"type": "cardio",
		"date": "2025-01-15",
		"exercises": [
			{
				"name": "Running",
				"weight": 0,
				"reps": 1,
				"sets": 1,
				"duration": 45
			}
		]
	}`

	workout, err := api.ParseWorkoutResponse(jsonResponse)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout)

	assert.Equal(s.T(), "cardio", workout.Workout.WorkoutType)
	assert.Len(s.T(), workout.Exercises, 1)
	assert.Equal(s.T(), "Running", workout.Exercises[0].Name)
	assert.Equal(s.T(), 45.0, workout.Exercises[0].Duration)
}

func (s *GenerateCommandTestSuite) TestParseWorkoutResponse_JSONWithExtraText() {
	// AI might wrap JSON in text
	response := `Here's your workout plan:

	{
		"type": "strength",
		"date": "2025-01-15",
		"exercises": [
			{
				"name": "Deadlifts",
				"weight": 120,
				"reps": 5,
				"sets": 3,
				"duration": 0
			}
		]
	}

	This workout focuses on building strength.`

	workout, err := api.ParseWorkoutResponse(response)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout)

	assert.Equal(s.T(), "strength", workout.Workout.WorkoutType)
	assert.Len(s.T(), workout.Exercises, 1)
	assert.Equal(s.T(), "Deadlifts", workout.Exercises[0].Name)
}

func (s *GenerateCommandTestSuite) TestParseWorkoutResponse_InvalidJSON() {
	invalidJSON := `This is not JSON at all`

	workout, err := api.ParseWorkoutResponse(invalidJSON)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), workout)
	assert.Contains(s.T(), err.Error(), "no valid JSON found")
}

func (s *GenerateCommandTestSuite) TestParseWorkoutResponse_InvalidDate() {
	jsonResponse := `{
		"type": "strength",
		"date": "invalid-date",
		"exercises": []
	}`

	workout, err := api.ParseWorkoutResponse(jsonResponse)
	assert.Error(s.T(), err)
	assert.Nil(s.T(), workout)
	assert.Contains(s.T(), err.Error(), "invalid date format")
}

func (s *GenerateCommandTestSuite) TestParseWorkoutResponse_EmptyExercises() {
	jsonResponse := `{
		"type": "strength",
		"date": "2025-01-15",
		"exercises": []
	}`

	workout, err := api.ParseWorkoutResponse(jsonResponse)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout)
	assert.Len(s.T(), workout.Exercises, 0)
}

// Test FormatWorkout function (used to display generated workout)

func (s *GenerateCommandTestSuite) TestFormatWorkout_GeneratedWorkout() {
	workout := &db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: "strength",
			WorkoutDate: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Status:      "planned",
		},
		Exercises: []db.Exercise{
			{Name: "Bench Press", Weight: 85, Repetitions: 10, Sets: 3},
			{Name: "Squats", Weight: 105, Repetitions: 8, Sets: 4},
		},
	}

	result := FormatWorkout(workout)

	assert.Contains(s.T(), result, "Workout Type: strength")
	assert.Contains(s.T(), result, "2025-01-15")
	assert.Contains(s.T(), result, "Status: planned")
	assert.Contains(s.T(), result, "Bench Press")
	assert.Contains(s.T(), result, "Squats")
	assert.Contains(s.T(), result, "85 kg")
	assert.Contains(s.T(), result, "105 kg")
}

// TestGenerateCommandTestSuite runs the test suite
func TestGenerateCommandTestSuite(t *testing.T) {
	suite.Run(t, new(GenerateCommandTestSuite))
}

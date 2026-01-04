package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/testutil"
)

// DeleteCommandTestSuite embeds the shared DBTestSuite
type DeleteCommandTestSuite struct {
	testutil.DBTestSuite
}

// Test DeleteCommand routing and validation

func (s *DeleteCommandTestSuite) TestDeleteCommand_Name() {
	cmd := NewDeleteCommand([]string{})
	assert.Equal(s.T(), "delete", cmd.Name())
}

func (s *DeleteCommandTestSuite) TestDeleteCommand_HelpManual() {
	cmd := NewDeleteCommand([]string{})
	helpText := cmd.HelpManual()
	assert.Contains(s.T(), helpText, "delete last")
	assert.Contains(s.T(), helpText, "delete date")
}

func (s *DeleteCommandTestSuite) TestDeleteCommand_Validate_TooFewArgs() {
	cmd := NewDeleteCommand([]string{"terminal_fit_recorder", "exercise", "delete"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "usage:")
}

func (s *DeleteCommandTestSuite) TestDeleteCommand_Validate_DeleteLast() {
	cmd := NewDeleteCommand([]string{"terminal_fit_recorder", "exercise", "delete", "last"})
	err := cmd.Validate()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cmd.command)
	assert.Equal(s.T(), "delete last workout", cmd.command.Name())
}

func (s *DeleteCommandTestSuite) TestDeleteCommand_Validate_DeleteDate_MissingDate() {
	cmd := NewDeleteCommand([]string{"terminal_fit_recorder", "exercise", "delete", "date"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "DD-MM-YY")
}

func (s *DeleteCommandTestSuite) TestDeleteCommand_Validate_DeleteDate_InvalidFormat() {
	cmd := NewDeleteCommand([]string{"terminal_fit_recorder", "exercise", "delete", "date", "2025-01-01"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid date format")
}

func (s *DeleteCommandTestSuite) TestDeleteCommand_Validate_DeleteDate_Valid() {
	cmd := NewDeleteCommand([]string{"terminal_fit_recorder", "exercise", "delete", "date", "01-01-25"})
	err := cmd.Validate()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cmd.command)
	assert.Equal(s.T(), "delete workout by date", cmd.command.Name())
}

func (s *DeleteCommandTestSuite) TestDeleteCommand_Validate_UnknownTarget() {
	cmd := NewDeleteCommand([]string{"terminal_fit_recorder", "exercise", "delete", "unknown"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "unknown delete target")
}

// Test DeleteLastWorkoutCommand

func (s *DeleteCommandTestSuite) TestDeleteLastWorkoutCommand_Name() {
	cmd := NewDeleteLastWorkoutCommand()
	assert.Equal(s.T(), "delete last workout", cmd.Name())
}

func (s *DeleteCommandTestSuite) TestDeleteLastWorkoutCommand_Validate() {
	cmd := NewDeleteLastWorkoutCommand()
	assert.NoError(s.T(), cmd.Validate())
}

func (s *DeleteCommandTestSuite) TestDeleteLastWorkoutCommand_HelpManual() {
	cmd := NewDeleteLastWorkoutCommand()
	_ = cmd.HelpManual() // Just verify it doesn't panic
}

// Test DeleteByDateCommand

func (s *DeleteCommandTestSuite) TestDeleteByDateCommand_Name() {
	date := time.Now()
	cmd := NewDeleteByDateCommand(date)
	assert.Equal(s.T(), "delete workout by date", cmd.Name())
}

func (s *DeleteCommandTestSuite) TestDeleteByDateCommand_Validate() {
	date := time.Now()
	cmd := NewDeleteByDateCommand(date)
	assert.NoError(s.T(), cmd.Validate())
}

func (s *DeleteCommandTestSuite) TestDeleteByDateCommand_HelpManual() {
	date := time.Now()
	cmd := NewDeleteByDateCommand(date)
	_ = cmd.HelpManual() // Just verify it doesn't panic
}

// Database integration tests for DeleteLastWorkout

func (s *DeleteCommandTestSuite) TestDatabaseIntegration_DeleteLastWorkout() {
	// Create multiple workouts
	yesterday := time.Now().AddDate(0, 0, -1)
	today := time.Now()

	workoutID1, err := s.DB.CreateWorkout("strength", "completed", yesterday)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID1, []db.Exercise{
		{Name: "Squats", Weight: 100, Repetitions: 5, Sets: 5},
	})
	require.NoError(s.T(), err)

	workoutID2, err := s.DB.CreateWorkout("cardio", "completed", today)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID2, []db.Exercise{
		{Name: "Running", Weight: 0, Repetitions: 1, Sets: 1, Duration: 30.0},
	})
	require.NoError(s.T(), err)

	// Verify we have 2 workouts
	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Len(s.T(), workouts, 2)

	// Delete the last workout (today's workout)
	err = s.DB.DeleteLastWorkout()
	require.NoError(s.T(), err)

	// Verify we now have 1 workout
	workouts, err = s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Len(s.T(), workouts, 1)
	assert.Equal(s.T(), "strength", workouts[0].Workout.WorkoutType)

	// Verify the last workout is now yesterday's
	lastWorkout, err := s.DB.GetLastWorkout()
	require.NoError(s.T(), err)
	require.NotNil(s.T(), lastWorkout)
	assert.Equal(s.T(), "strength", lastWorkout.Workout.WorkoutType)
}

func (s *DeleteCommandTestSuite) TestDatabaseIntegration_DeleteLastWorkout_NoWorkouts() {
	// Delete when no workouts exist (should not error)
	err := s.DB.DeleteLastWorkout()
	assert.NoError(s.T(), err)

	// Verify no workouts
	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Empty(s.T(), workouts)
}

func (s *DeleteCommandTestSuite) TestDatabaseIntegration_DeleteLastWorkout_SingleWorkout() {
	// Create one workout
	today := time.Now()
	workoutID, err := s.DB.CreateWorkout("strength", "completed", today)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID, []db.Exercise{
		{Name: "Bench Press", Weight: 80, Repetitions: 10, Sets: 3},
	})
	require.NoError(s.T(), err)

	// Delete it
	err = s.DB.DeleteLastWorkout()
	require.NoError(s.T(), err)

	// Verify no workouts remain
	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Empty(s.T(), workouts)
}

// Database integration tests for DeleteWorkoutByDate

func (s *DeleteCommandTestSuite) TestDatabaseIntegration_DeleteWorkoutByDate() {
	// Create multiple workouts on different dates
	yesterday := time.Now().AddDate(0, 0, -1)
	today := time.Now()
	twoDaysAgo := time.Now().AddDate(0, 0, -2)

	workoutID1, err := s.DB.CreateWorkout("strength", "completed", twoDaysAgo)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID1, []db.Exercise{
		{Name: "Deadlifts", Weight: 120, Repetitions: 5, Sets: 3},
	})
	require.NoError(s.T(), err)

	workoutID2, err := s.DB.CreateWorkout("cardio", "completed", yesterday)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID2, []db.Exercise{
		{Name: "Running", Weight: 0, Repetitions: 1, Sets: 1, Duration: 45.0},
	})
	require.NoError(s.T(), err)

	workoutID3, err := s.DB.CreateWorkout("yoga", "completed", today)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID3, []db.Exercise{
		{Name: "Sun Salutation", Weight: 0, Repetitions: 10, Sets: 1},
	})
	require.NoError(s.T(), err)

	// Verify we have 3 workouts
	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Len(s.T(), workouts, 3)

	// Delete yesterday's workout
	err = s.DB.DeleteWorkoutByDate(yesterday)
	require.NoError(s.T(), err)

	// Verify we now have 2 workouts
	workouts, err = s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Len(s.T(), workouts, 2)

	// Verify the remaining workouts are correct
	assert.Equal(s.T(), "yoga", workouts[0].Workout.WorkoutType)     // today (most recent)
	assert.Equal(s.T(), "strength", workouts[1].Workout.WorkoutType) // two days ago
}

func (s *DeleteCommandTestSuite) TestDatabaseIntegration_DeleteWorkoutByDate_NoWorkoutOnDate() {
	// Create a workout today
	today := time.Now()
	workoutID, err := s.DB.CreateWorkout("strength", "completed", today)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID, []db.Exercise{
		{Name: "Bench Press", Weight: 80, Repetitions: 10, Sets: 3},
	})
	require.NoError(s.T(), err)

	// Try to delete a workout from yesterday (doesn't exist)
	yesterday := time.Now().AddDate(0, 0, -1)
	err = s.DB.DeleteWorkoutByDate(yesterday)
	assert.NoError(s.T(), err) // Should not error

	// Verify today's workout still exists
	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Len(s.T(), workouts, 1)
	assert.Equal(s.T(), "strength", workouts[0].Workout.WorkoutType)
}

func (s *DeleteCommandTestSuite) TestDatabaseIntegration_DeleteWorkoutByDate_DeletesExercises() {
	// Create a workout with multiple exercises
	today := time.Now()
	workoutID, err := s.DB.CreateWorkout("strength", "completed", today)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID, []db.Exercise{
		{Name: "Bench Press", Weight: 80, Repetitions: 10, Sets: 3},
		{Name: "Squats", Weight: 100, Repetitions: 5, Sets: 5},
		{Name: "Deadlifts", Weight: 120, Repetitions: 5, Sets: 3},
	})
	require.NoError(s.T(), err)

	// Verify workout and exercises exist
	workout, err := s.DB.GetWorkoutByDate(today)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout)
	assert.Len(s.T(), workout.Exercises, 3)

	// Delete the workout
	err = s.DB.DeleteWorkoutByDate(today)
	require.NoError(s.T(), err)

	// Verify workout and its exercises are gone
	workout, err = s.DB.GetWorkoutByDate(today)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), workout)

	workouts, err := s.DB.GetAllWorkouts()
	require.NoError(s.T(), err)
	assert.Empty(s.T(), workouts)
}

// TestDeleteCommandTestSuite runs the test suite
func TestDeleteCommandTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteCommandTestSuite))
}

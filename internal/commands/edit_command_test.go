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

// EditCommandTestSuite embeds the shared DBTestSuite
type EditCommandTestSuite struct {
	testutil.DBTestSuite
}

// Test EditCommand routing and validation

func (s *EditCommandTestSuite) TestEditCommand_Name() {
	cmd := NewEditCommand([]string{})
	assert.Equal(s.T(), "edit", cmd.Name())
}

func (s *EditCommandTestSuite) TestEditCommand_HelpManual() {
	cmd := NewEditCommand([]string{})
	helpText := cmd.HelpManual()
	assert.Contains(s.T(), helpText, "edit <DD-MM-YY>")
	assert.Contains(s.T(), helpText, "edit date")
	assert.Contains(s.T(), helpText, "edit last status")
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_TooFewArgs() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "usage:")
}

// Test "edit <date>" routing

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditByDate_Valid() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "01-01-25"})
	err := cmd.Validate()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cmd.command)
	assert.Equal(s.T(), "edit workout", cmd.command.Name())
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditByDate_InvalidFormat() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "2025-01-01"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid date format")
}

// Test "edit last status" routing

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditLastStatus_Valid() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "last", "status", "completed"})
	err := cmd.Validate()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cmd.command)
	assert.Equal(s.T(), "edit workout status", cmd.command.Name())
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditLastStatus_MissingArgs() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "last", "status"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "usage:")
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditLastStatus_MissingStatusKeyword() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "last", "something", "completed"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "usage:")
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditLastStatus_InvalidStatus() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "last", "status", "invalid"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "status must be")
}

// Test "edit date <old> <new>" routing

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditDate_Valid() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "date", "01-01-25", "02-01-25"})
	err := cmd.Validate()
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), cmd.command)
	assert.Equal(s.T(), "edit workout date", cmd.command.Name())
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditDate_MissingNewDate() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "date", "01-01-25"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "usage:")
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditDate_InvalidOldDate() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "date", "2025-01-01", "02-01-25"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid old date format")
}

func (s *EditCommandTestSuite) TestEditCommand_Validate_EditDate_InvalidNewDate() {
	cmd := NewEditCommand([]string{"terminal_fit_recorder", "exercise", "edit", "date", "01-01-25", "2025-01-02"})
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid new date format")
}

// Test EditWorkoutStatusCommand

func (s *EditCommandTestSuite) TestEditWorkoutStatusCommand_Name() {
	cmd := NewEditWorkoutStatusCommand("completed")
	assert.Equal(s.T(), "edit workout status", cmd.Name())
}

func (s *EditCommandTestSuite) TestEditWorkoutStatusCommand_Validate_Completed() {
	cmd := NewEditWorkoutStatusCommand("completed")
	assert.NoError(s.T(), cmd.Validate())
}

func (s *EditCommandTestSuite) TestEditWorkoutStatusCommand_Validate_Planned() {
	cmd := NewEditWorkoutStatusCommand("planned")
	assert.NoError(s.T(), cmd.Validate())
}

func (s *EditCommandTestSuite) TestEditWorkoutStatusCommand_Validate_Invalid() {
	cmd := NewEditWorkoutStatusCommand("invalid")
	err := cmd.Validate()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "status must be")
}

func (s *EditCommandTestSuite) TestEditWorkoutStatusCommand_HelpManual() {
	cmd := NewEditWorkoutStatusCommand("completed")
	_ = cmd.HelpManual() // Just verify it doesn't panic
}

// Test EditWorkoutDateCommand

func (s *EditCommandTestSuite) TestEditWorkoutDateCommand_Name() {
	oldDate := time.Now()
	newDate := time.Now().AddDate(0, 0, 1)
	cmd := NewEditWorkoutDateCommand(oldDate, newDate)
	assert.Equal(s.T(), "edit workout date", cmd.Name())
}

func (s *EditCommandTestSuite) TestEditWorkoutDateCommand_Validate() {
	oldDate := time.Now()
	newDate := time.Now().AddDate(0, 0, 1)
	cmd := NewEditWorkoutDateCommand(oldDate, newDate)
	assert.NoError(s.T(), cmd.Validate())
}

func (s *EditCommandTestSuite) TestEditWorkoutDateCommand_HelpManual() {
	oldDate := time.Now()
	newDate := time.Now().AddDate(0, 0, 1)
	cmd := NewEditWorkoutDateCommand(oldDate, newDate)
	_ = cmd.HelpManual() // Just verify it doesn't panic
}

// Database integration tests for UpdateLastWorkoutStatus

func (s *EditCommandTestSuite) TestDatabaseIntegration_UpdateLastWorkoutStatus() {
	// Create a workout with "planned" status
	yesterday := time.Now().AddDate(0, 0, -1)
	today := time.Now()

	workoutID1, err := s.DB.CreateWorkout("strength", "planned", yesterday)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID1, []db.Exercise{
		{Name: "Squats", Weight: 100, Repetitions: 5, Sets: 5},
	})
	require.NoError(s.T(), err)

	workoutID2, err := s.DB.CreateWorkout("cardio", "planned", today)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID2, []db.Exercise{
		{Name: "Running", Weight: 0, Repetitions: 1, Sets: 1, Duration: 30.0},
	})
	require.NoError(s.T(), err)

	// Verify last workout status is "planned"
	lastWorkout, err := s.DB.GetLastWorkout()
	require.NoError(s.T(), err)
	require.NotNil(s.T(), lastWorkout)
	assert.Equal(s.T(), "planned", lastWorkout.Workout.Status)

	// Update status to "completed"
	err = s.DB.UpdateLastWorkoutStatus("completed")
	require.NoError(s.T(), err)

	// Verify status was updated
	lastWorkout, err = s.DB.GetLastWorkout()
	require.NoError(s.T(), err)
	require.NotNil(s.T(), lastWorkout)
	assert.Equal(s.T(), "completed", lastWorkout.Workout.Status)

	// Verify only the last workout was updated
	workout1, err := s.DB.GetWorkoutByDate(yesterday)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout1)
	assert.Equal(s.T(), "planned", workout1.Workout.Status)
}

func (s *EditCommandTestSuite) TestDatabaseIntegration_UpdateLastWorkoutStatus_NoWorkouts() {
	// Try to update when no workouts exist
	err := s.DB.UpdateLastWorkoutStatus("completed")
	assert.Error(s.T(), err) // Should error when no workouts found
	assert.Contains(s.T(), err.Error(), "no workouts found")
}

// Database integration tests for UpdateWorkoutDate

func (s *EditCommandTestSuite) TestDatabaseIntegration_UpdateWorkoutDate() {
	// Create a workout on a specific date
	oldDate := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	newDate := time.Date(2025, 1, 20, 10, 0, 0, 0, time.UTC)

	workoutID, err := s.DB.CreateWorkout("strength", "completed", oldDate)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID, []db.Exercise{
		{Name: "Bench Press", Weight: 80, Repetitions: 10, Sets: 3},
	})
	require.NoError(s.T(), err)

	// Verify workout exists on old date
	workout, err := s.DB.GetWorkoutByDate(oldDate)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout)
	assert.Equal(s.T(), "strength", workout.Workout.WorkoutType)

	// Update the date
	err = s.DB.UpdateWorkoutDate(oldDate, newDate)
	require.NoError(s.T(), err)

	// Verify workout no longer exists on old date
	workout, err = s.DB.GetWorkoutByDate(oldDate)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), workout)

	// Verify workout exists on new date
	workout, err = s.DB.GetWorkoutByDate(newDate)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), workout)
	assert.Equal(s.T(), "strength", workout.Workout.WorkoutType)
	assert.Len(s.T(), workout.Exercises, 1)
	assert.Equal(s.T(), "Bench Press", workout.Exercises[0].Name)
}

func (s *EditCommandTestSuite) TestDatabaseIntegration_UpdateWorkoutDate_NoWorkoutOnOldDate() {
	// Try to update date when no workout exists on old date
	oldDate := time.Now().AddDate(0, 0, -10)
	newDate := time.Now().AddDate(0, 0, -5)

	err := s.DB.UpdateWorkoutDate(oldDate, newDate)
	assert.Error(s.T(), err) // Should error when no workout found
	assert.Contains(s.T(), err.Error(), "no workout found")
}

func (s *EditCommandTestSuite) TestDatabaseIntegration_UpdateWorkoutDate_ConflictWithExisting() {
	// Create two workouts on different dates
	date1 := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	date2 := time.Date(2025, 1, 20, 10, 0, 0, 0, time.UTC)

	workoutID1, err := s.DB.CreateWorkout("strength", "completed", date1)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID1, []db.Exercise{
		{Name: "Squats", Weight: 100, Repetitions: 5, Sets: 5},
	})
	require.NoError(s.T(), err)

	workoutID2, err := s.DB.CreateWorkout("cardio", "completed", date2)
	require.NoError(s.T(), err)
	err = s.DB.SaveExercisesForWorkout(workoutID2, []db.Exercise{
		{Name: "Running", Weight: 0, Repetitions: 1, Sets: 1, Duration: 30.0},
	})
	require.NoError(s.T(), err)

	// Try to update date1 to date2 (which already has a workout)
	err = s.DB.UpdateWorkoutDate(date1, date2)
	assert.Error(s.T(), err) // Should error due to constraint
	assert.Contains(s.T(), err.Error(), "workout already exists")
}

// TestEditCommandTestSuite runs the test suite
func TestEditCommandTestSuite(t *testing.T) {
	suite.Run(t, new(EditCommandTestSuite))
}

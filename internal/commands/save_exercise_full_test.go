package commands

import (
	"testing"

	"terminal_fit_recorder/internal/ui"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSaveExerciseCommand_Execute_FullFlow_SingleExercise(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// Mock the input sequence for saving one exercise
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("strength", false)
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("Bench Press", false)
	mockInput.On("GetInputWithType", "Weight: ", mock.Anything, ui.InputTypeText).Return("80", false)
	mockInput.On("GetInputWithType", "Repetitions: ", mock.Anything, ui.InputTypeText).Return("10", false)
	mockInput.On("GetInputWithType", "Number of sets: ", mock.Anything, ui.InputTypeText).Return("3", false)
	mockInput.On("GetInputWithType", "Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox).Return("yes", false)

	// Create mock Ollama (not used but required by Execute signature)
	mockOllama := new(MockOllamaClient)
	mockOllama.On("GetCustomPrompt").Return("")

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify the workout was saved
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.NotNil(t, workout)
	assert.Equal(t, "strength", workout.Workout.WorkoutType)
	assert.Equal(t, "completed", workout.Workout.Status)
	assert.Len(t, workout.Exercises, 1)

	// Verify exercise details
	exercise := workout.Exercises[0]
	assert.Equal(t, "Bench Press", exercise.Name)
	assert.Equal(t, 80, exercise.Weight)
	assert.Equal(t, 10, exercise.Repetitions)
	assert.Equal(t, 3, exercise.Sets)

	// Verify all mocks were called
	mockInput.AssertExpectations(t)
}

func TestSaveExerciseCommand_Execute_FullFlow_MultipleExercises(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// Mock the input sequence for saving multiple exercises
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("strength", false)

	// First exercise: Bench Press
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("Bench Press", false).Once()
	mockInput.On("GetInputWithType", "Weight: ", mock.Anything, ui.InputTypeText).Return("80", false).Once()
	mockInput.On("GetInputWithType", "Repetitions: ", mock.Anything, ui.InputTypeText).Return("10", false).Once()
	mockInput.On("GetInputWithType", "Number of sets: ", mock.Anything, ui.InputTypeText).Return("3", false).Once()
	mockInput.On("GetInputWithType", "Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox).Return("no", false).Once()

	// Second exercise: Squats
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("Squats", false).Once()
	mockInput.On("GetInputWithType", "Weight: ", mock.Anything, ui.InputTypeText).Return("100", false).Once()
	mockInput.On("GetInputWithType", "Repetitions: ", mock.Anything, ui.InputTypeText).Return("8", false).Once()
	mockInput.On("GetInputWithType", "Number of sets: ", mock.Anything, ui.InputTypeText).Return("4", false).Once()
	mockInput.On("GetInputWithType", "Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox).Return("yes", false).Once()

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify the workout was saved with both exercises
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.NotNil(t, workout)
	assert.Equal(t, "strength", workout.Workout.WorkoutType)
	assert.Len(t, workout.Exercises, 2)

	// Verify first exercise
	assert.Equal(t, "Bench Press", workout.Exercises[0].Name)
	assert.Equal(t, 80, workout.Exercises[0].Weight)
	assert.Equal(t, 10, workout.Exercises[0].Repetitions)
	assert.Equal(t, 3, workout.Exercises[0].Sets)

	// Verify second exercise
	assert.Equal(t, "Squats", workout.Exercises[1].Name)
	assert.Equal(t, 100, workout.Exercises[1].Weight)
	assert.Equal(t, 8, workout.Exercises[1].Repetitions)
	assert.Equal(t, 4, workout.Exercises[1].Sets)

	// Verify all mocks were called
	mockInput.AssertExpectations(t)
}

func TestSaveExerciseCommand_Execute_FullFlow_ExerciseWithDuration(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// Mock the input sequence for an exercise requiring duration (plank)
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("strength", false)
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("Plank", false)
	mockInput.On("GetInputWithType", "Weight: ", mock.Anything, ui.InputTypeText).Return("0", false)
	mockInput.On("GetInputWithType", "Repetitions: ", mock.Anything, ui.InputTypeText).Return("1", false)
	mockInput.On("GetInputWithType", "Number of sets: ", mock.Anything, ui.InputTypeText).Return("3", false)
	mockInput.On("GetInputWithType", "Duration (minutes): ", mock.Anything, ui.InputTypeText).Return("2.5", false)
	mockInput.On("GetInputWithType", "Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox).Return("yes", false)

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify the workout was saved
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.NotNil(t, workout)
	assert.Len(t, workout.Exercises, 1)

	// Verify exercise has duration
	exercise := workout.Exercises[0]
	assert.Equal(t, "Plank", exercise.Name)
	assert.Equal(t, 0, exercise.Weight)
	assert.Equal(t, 1, exercise.Repetitions)
	assert.Equal(t, 3, exercise.Sets)
	assert.Equal(t, 2.5, exercise.Duration)

	// Verify all mocks were called
	mockInput.AssertExpectations(t)
}

func TestSaveExerciseCommand_Execute_UserCancelsAtWorkoutType(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// User cancels at workout type
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("", true)

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify no workout was saved
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.Nil(t, workout)

	// Verify mock was called
	mockInput.AssertExpectations(t)
}

func TestSaveExerciseCommand_Execute_UserCancelsAtExerciseName(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// User cancels at exercise name
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("strength", false)
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("", true)

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify no workout was saved
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.Nil(t, workout)

	// Verify mocks were called
	mockInput.AssertExpectations(t)
}

func TestSaveExerciseCommand_Execute_CardioWorkout(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// Mock the input sequence for cardio workout
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("cardio", false)
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("Running", false)
	mockInput.On("GetInputWithType", "Distance (meters): ", mock.Anything, ui.InputTypeText).Return("1000", false)
	mockInput.On("GetInputWithType", "Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox).Return("yes", false)

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify the workout was saved
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.NotNil(t, workout)
	assert.Equal(t, "cardio", workout.Workout.WorkoutType)

	// Verify exercise
	exercise := workout.Exercises[0]
	assert.Equal(t, "Running", exercise.Name)
	assert.Equal(t, 1000, exercise.Distance)

	// Verify all mocks were called
	mockInput.AssertExpectations(t)
}

func TestSaveExerciseCommand_Execute_DistanceExerciseWithoutDuration(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// Mock the input sequence for distance-based exercise (swimming) without weight/reps/sets
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("cardio", false)
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("Pushups", false)
	mockInput.On("GetInputWithType", "Weight: ", mock.Anything, ui.InputTypeText).Return("0", false)
	mockInput.On("GetInputWithType", "Repetitions: ", mock.Anything, ui.InputTypeText).Return("30", false)
	mockInput.On("GetInputWithType", "Number of sets: ", mock.Anything, ui.InputTypeText).Return("3", false)
	mockInput.On("GetInputWithType", "Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox).Return("yes", false)

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify the workout was saved
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.NotNil(t, workout)
	assert.Equal(t, "cardio", workout.Workout.WorkoutType)
	assert.Len(t, workout.Exercises, 1)

	// Verify exercise has distance but no weight/reps/sets
	exercise := workout.Exercises[0]
	assert.Equal(t, "Pushups", exercise.Name)
	assert.Equal(t, 0, exercise.Weight)
	assert.Equal(t, 30, exercise.Repetitions)
	assert.Equal(t, 3, exercise.Sets)
	assert.Equal(t, 0.0, exercise.Duration)

	// Verify all mocks were called
	mockInput.AssertExpectations(t)
}

func TestSaveExerciseCommand_Execute_DistanceExerciseWithDuration(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// Mock the input sequence for distance-based exercise with duration (rowing)
	mockInput.On("GetInputWithType", "Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox).Return("cardio", false)
	mockInput.On("GetInputWithType", "Exercise name: ", mock.Anything, ui.InputTypeAutocomplete).Return("Wall Hold", false)
	mockInput.On("GetInputWithType", "Weight: ", mock.Anything, ui.InputTypeText).Return("0", false)
	mockInput.On("GetInputWithType", "Repetitions: ", mock.Anything, ui.InputTypeText).Return("1", false)
	mockInput.On("GetInputWithType", "Number of sets: ", mock.Anything, ui.InputTypeText).Return("3", false)
	mockInput.On("GetInputWithType", "Duration (minutes): ", mock.Anything, ui.InputTypeText).Return("15.5", false)
	mockInput.On("GetInputWithType", "Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox).Return("yes", false)

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	// Create command and inject mock
	cmd := NewSaveExerciseCommand()
	cmd.InputProvider = mockInput

	// Execute
	err := cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify the workout was saved
	workout, err := database.GetLastWorkout()
	assert.NoError(t, err)
	assert.NotNil(t, workout)
	assert.Equal(t, "cardio", workout.Workout.WorkoutType)
	assert.Len(t, workout.Exercises, 1)

	// Verify exercise has both distance and duration, but no weight/reps/sets
	exercise := workout.Exercises[0]
	assert.Equal(t, "Wall Hold", exercise.Name)
	assert.Equal(t, 0, exercise.Distance)
	assert.Equal(t, 15.5, exercise.Duration)
	assert.Equal(t, 0, exercise.Weight)
	assert.Equal(t, 1, exercise.Repetitions)
	assert.Equal(t, 3, exercise.Sets)

	// Verify all mocks were called
	mockInput.AssertExpectations(t)
}

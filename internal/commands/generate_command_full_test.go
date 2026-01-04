package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/ui"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// findProjectRoot walks up the directory tree to find the project root
func findProjectRoot(startDir string) string {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// setupTestDB creates an in-memory database with migrations and returns cleanup function
func setupTestDB(t *testing.T) (*db.DB, func()) {
	originalWd, _ := os.Getwd()
	projectRoot := findProjectRoot(originalWd)
	os.Chdir(projectRoot)

	database, err := db.New(":memory:")
	assert.NoError(t, err)

	cleanup := func() {
		if database != nil {
			database.Close()
		}
		os.Chdir(originalWd)
	}

	return database, cleanup
}

// MockOllamaClient is a mock implementation of the OllamaClient interface
type MockOllamaClient struct {
	mock.Mock
}

func (m *MockOllamaClient) SendPromptStream(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockOllamaClient) SendPrompt(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockOllamaClient) GetCustomPrompt() string {
	args := m.Called()
	return args.String(0)
}

// MockInputProvider is a mock implementation of the InputProvider interface
type MockInputProvider struct {
	mock.Mock
}

func (m *MockInputProvider) GetInputWithType(prompt string, suggestions []string, inputType ui.InputType) (string, bool) {
	args := m.Called(prompt, suggestions, inputType)
	return args.String(0), args.Bool(1)
}

func TestGenerateCommand_Execute_FullFlow_UserSavesWorkout(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Seed database with workout data
	workout := db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: "push",
			WorkoutDate: time.Now(),
		},
		Exercises: []db.Exercise{
			{
				Name:        "Bench Press",
				Weight:      80,
				Repetitions: 10,
				Sets:        3,
				Duration:    0,
			},
			{
				Name:        "Push-ups",
				Weight:      0,
				Repetitions: 20,
				Sets:        3,
				Duration:    0,
			},
		},
	}
	err := database.SaveGeneratedWorkout(&workout)
	assert.NoError(t, err)

	// Create mock Ollama client
	mockOllama := new(MockOllamaClient)

	// Mock AI response - this is the JSON that would normally come from Ollama
	aiResponse := `{
		"date": "2026-01-05",
		"type": "push",
		"exercises": [
			{
				"name": "Bench Press",
				"weight": 85,
				"reps": 8,
				"sets": 4,
				"duration": "-"
			},
			{
				"name": "Incline Dumbbell Press",
				"weight": 30,
				"reps": 10,
				"sets": 3,
				"duration": "-"
			},
			{
				"name": "Push-ups",
				"weight": "-",
				"reps": 15,
				"sets": 4,
				"duration": "-"
			}
		]
	}`

	// Set up Ollama mock expectations
	mockOllama.On("GetCustomPrompt").Return("")
	mockOllama.On("SendPromptStream", mock.Anything, mock.Anything).Return(aiResponse, nil)

	// Create mock input provider
	mockInput := new(MockInputProvider)

	// User chooses to save the workout
	mockInput.On("GetInputWithType",
		"\nSave this workout as 'planned'?",
		[]string{"yes", "no"},
		ui.InputTypeCheckbox).Return("yes", false)

	// Create command and inject mocks
	cmd := NewGenerateCommand(3)
	cmd.InputProvider = mockInput

	// Execute the command
	err = cmd.Execute(database, mockOllama)

	// Verify no errors
	assert.NoError(t, err)

	// Verify that SendPromptStream was called
	mockOllama.AssertCalled(t, "SendPromptStream", mock.Anything, mock.Anything)
	mockOllama.AssertCalled(t, "GetCustomPrompt")

	// Verify user input was requested
	mockInput.AssertCalled(t, "GetInputWithType", "\nSave this workout as 'planned'?", []string{"yes", "no"}, ui.InputTypeCheckbox)

	// Verify expectations were met
	mockOllama.AssertExpectations(t)
	mockInput.AssertExpectations(t)

	// Verify the workout was saved to database
	workouts, err := database.GetAllWorkouts()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(workouts), "Should have 2 workouts: original + generated")
}

func TestGenerateCommand_Execute_FullFlow_UserDoesNotSave(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Seed database with workout data
	workout := db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: "legs",
			WorkoutDate: time.Now().AddDate(0, 0, -1),
		},
		Exercises: []db.Exercise{
			{
				Name:        "Squats",
				Weight:      100,
				Repetitions: 8,
				Sets:        4,
				Duration:    0,
			},
		},
	}
	err := database.SaveGeneratedWorkout(&workout)
	assert.NoError(t, err)

	// Create mock Ollama
	mockOllama := new(MockOllamaClient)

	aiResponse := `{
		"date": "2026-01-05",
		"type": "legs",
		"exercises": [
			{
				"name": "Squats",
				"weight": 105,
				"reps": 6,
				"sets": 5,
				"duration": "-"
			}
		]
	}`

	mockOllama.On("GetCustomPrompt").Return("Be brief")
	mockOllama.On("SendPromptStream", mock.Anything, mock.AnythingOfType("string")).Return(aiResponse, nil)

	// Create mock input - user chooses NOT to save
	mockInput := new(MockInputProvider)
	mockInput.On("GetInputWithType",
		"\nSave this workout as 'planned'?",
		[]string{"yes", "no"},
		ui.InputTypeCheckbox).Return("no", false)

	// Create command with mocks
	cmd := NewGenerateCommand(2)
	cmd.InputProvider = mockInput

	// Execute
	err = cmd.Execute(database, mockOllama)
	assert.NoError(t, err)

	// Verify mocks were called
	mockOllama.AssertExpectations(t)
	mockInput.AssertExpectations(t)

	// Verify workout was NOT saved (still only 1 workout)
	workouts, err := database.GetAllWorkouts()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(workouts), "Should still have only 1 workout")
}

func TestGenerateCommand_Execute_FullFlow_UserCancels(t *testing.T) {
	// Setup test database
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// Seed database
	workout := db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: "pull",
			WorkoutDate: time.Now(),
		},
		Exercises: []db.Exercise{
			{Name: "Deadlifts", Weight: 120, Repetitions: 5, Sets: 3},
		},
	}
	err := database.SaveGeneratedWorkout(&workout)
	assert.NoError(t, err)

	// Mock Ollama
	mockOllama := new(MockOllamaClient)
	aiResponse := `{"date": "2026-01-05", "type": "pull", "exercises": [{"name": "Deadlifts", "weight": 125, "reps": 5, "sets": 3, "duration": "-"}]}`

	mockOllama.On("GetCustomPrompt").Return("")
	mockOllama.On("SendPromptStream", mock.Anything, mock.Anything).Return(aiResponse, nil)

	// Mock input - user cancels
	mockInput := new(MockInputProvider)
	mockInput.On("GetInputWithType",
		"\nSave this workout as 'planned'?",
		[]string{"yes", "no"},
		ui.InputTypeCheckbox).Return("", true) // cancelled = true

	// Execute
	cmd := NewGenerateCommand(1)
	cmd.InputProvider = mockInput
	err = cmd.Execute(database, mockOllama)

	assert.NoError(t, err)
	mockOllama.AssertExpectations(t)
	mockInput.AssertExpectations(t)

	// Verify workout was NOT saved
	workouts, err := database.GetAllWorkouts()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(workouts))
}

func TestMockOllamaClient_VerifyResponseStructure(t *testing.T) {
	// This test verifies that our mock response can be parsed correctly
	mockOllama := new(MockOllamaClient)

	aiResponse := `{
		"date": "2026-01-05",
		"type": "pull",
		"exercises": [
			{
				"name": "Deadlifts",
				"weight": 120,
				"reps": 5,
				"sets": 3,
				"duration": "-"
			}
		]
	}`

	mockOllama.On("SendPromptStream", mock.Anything, mock.Anything).Return(aiResponse, nil)

	// Call the mock
	ctx := context.Background()
	response, err := mockOllama.SendPromptStream(ctx, "test prompt")

	assert.NoError(t, err)
	assert.NotEmpty(t, response)

	// Parse the response to verify it's valid
	workout, err := api.ParseWorkoutResponse(response)
	assert.NoError(t, err)
	assert.NotNil(t, workout)
	assert.Equal(t, "pull", workout.Workout.WorkoutType)
	assert.Equal(t, 1, len(workout.Exercises))
	assert.Equal(t, "Deadlifts", workout.Exercises[0].Name)
	assert.Equal(t, 120, workout.Exercises[0].Weight)

	mockOllama.AssertExpectations(t)
}

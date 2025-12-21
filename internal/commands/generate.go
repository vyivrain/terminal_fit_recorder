package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/prompt"
	"terminal_fit_recorder/internal/ui"
)

type GenerateCommand struct {
	ExerciseCount int
}

func NewGenerateCommand(exerciseCount int) *GenerateCommand {
	return &GenerateCommand{
		ExerciseCount: exerciseCount,
	}
}

func (cmd *GenerateCommand) Name() string {
	return "generate"
}

func (cmd *GenerateCommand) Validate() error {
	if cmd.ExerciseCount < 0 || cmd.ExerciseCount > 20 {
		return fmt.Errorf("exercise count must be between 0 and 20, got %d", cmd.ExerciseCount)
	}
	return nil
}

func (cmd *GenerateCommand) HelpManual() string {
	return ""
}

func (cmd *GenerateCommand) Execute(database *db.DB, ollamaClient *api.Client) error {
	// Increased timeout to 5 minutes for longer responses
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("Fetching workout data...")

	// Get all workouts with exercises from database
	workouts, err := database.GetAllWorkouts()
	if err != nil {
		return fmt.Errorf("error fetching workouts: %v", err)
	}

	if len(workouts) == 0 {
		return fmt.Errorf("no workout data found. Please save some workouts first")
	}

	// Build prompt with workout data
	promptText := prompt.BuildPrompt(workouts, cmd.ExerciseCount, ollamaClient.CustomPrompt)

	fmt.Println("Sending request to Ollama...\n")

	// Use streaming to show real-time response and avoid timeouts
	response, err := ollamaClient.SendPromptStream(ctx, promptText)
	if err != nil {
		return fmt.Errorf("error generating response: %v", err)
	}

	// Parse AI response into workout structure
	generatedWorkout, err := api.ParseWorkoutResponse(response)
	if err != nil {
		return fmt.Errorf("error parsing workout response: %v", err)
	}

	// Format and display the workout
	formattedWorkout := FormatWorkout(generatedWorkout)
	fmt.Println(formattedWorkout)

	// Ask user if they want to save the workout
	saveWorkout, cancelled := ui.GetInputWithType("\nSave this workout as 'planned'?", []string{"yes", "no"}, ui.InputTypeCheckbox)
	if cancelled {
		fmt.Println("\nWorkout discarded")
		return nil
	}

	saveWorkout = strings.ToLower(strings.TrimSpace(saveWorkout))
	if saveWorkout == "yes" || saveWorkout == "y" {
		// Save the generated workout with "planned" status
		err = database.SaveGeneratedWorkout(generatedWorkout)
		if err != nil {
			return fmt.Errorf("error saving planned workout: %v", err)
		}
		fmt.Println("✓ Workout saved as 'planned'")
	} else {
		fmt.Println("Workout not saved")
	}

	return nil
}

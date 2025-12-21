package commands

import (
	"fmt"
	"strings"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/ui"
	"terminal_fit_recorder/internal/utils"
)

type SaveExerciseCommand struct{}

func NewSaveExerciseCommand() *SaveExerciseCommand {
	return &SaveExerciseCommand{}
}

func (cmd *SaveExerciseCommand) Name() string {
	return "save exercise"
}

func (cmd *SaveExerciseCommand) Validate() error {
	return nil
}

func (cmd *SaveExerciseCommand) HelpManual() string {
	return "terminal_fit_recorder exercise save\n    Start an interactive session to save a new workout with exercises."
}

func (cmd *SaveExerciseCommand) Execute(database *db.DB, ollamaClient *api.Client) error {
	// Get workout type using checkbox
	workoutType, cancelled := ui.GetInputWithType("Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox)
	if cancelled {
		fmt.Println("\nWorkout cancelled")
		return nil
	}
	workoutType = strings.ToLower(strings.TrimSpace(workoutType))

	// Get existing exercise names for autocomplete
	existingNames, err := database.GetDistinctExerciseNames()
	if err != nil {
		return fmt.Errorf("error fetching exercise names: %v", err)
	}

	// Collect exercises
	fmt.Println("\nEnter exercise details (Ctrl+C to exit, Ctrl+D to cancel workout):")
	var exercises []db.Exercise

	for {
		name, cancelled := ui.GetInputWithType("Exercise name: ", existingNames, ui.InputTypeAutocomplete)
		if cancelled {
			fmt.Println("\nWorkout cancelled")
			return nil
		}

		weight, cancelled := ui.GetInputWithType("Weight: ", nil, ui.InputTypeText)
		if cancelled {
			fmt.Println("\nWorkout cancelled")
			return nil
		}

		reps, cancelled := ui.GetInputWithType("Repetitions: ", nil, ui.InputTypeText)
		if cancelled {
			fmt.Println("\nWorkout cancelled")
			return nil
		}

		sets, cancelled := ui.GetInputWithType("Number of sets: ", nil, ui.InputTypeText)
		if cancelled {
			fmt.Println("\nWorkout cancelled")
			return nil
		}

		exercise := db.Exercise{
			Name:        name,
			Weight:      weight,
			Repetitions: reps,
			Sets:        sets,
		}

		// Check if duration is required for this exercise
		requiresDuration := false
		nameLower := strings.ToLower(name)
		for _, keyword := range db.DurationRequiredKeywords {
			if strings.Contains(nameLower, keyword) {
				requiresDuration = true
				break
			}
		}

		if requiresDuration {
			durationStr, cancelled := ui.GetInputWithType("Duration (minutes): ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nWorkout cancelled")
				return nil
			}

			// Parse duration string to float64
			var duration float64
			_, err := fmt.Sscanf(durationStr, "%f", &duration)
			if err != nil {
				return fmt.Errorf("invalid duration format: %v", err)
			}
			exercise.Duration = duration
		}

		exercises = append(exercises, exercise)

		if exercise.Duration > 0 {
			fmt.Printf("\nRecorded: %s - %s weight, %s reps, %s sets, %.2f minutes\n",
				exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration)
		} else {
			fmt.Printf("\nRecorded: %s - %s weight, %s reps, %s sets\n",
				exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets)
		}

		for {
			finished, cancelled := ui.GetInputWithType("Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox)
			if cancelled {
				fmt.Println("\nWorkout cancelled")
				return nil
			}
			finished = strings.ToLower(strings.TrimSpace(finished))

			if finished == "yes" || finished == "y" {
				// Save to database
				workoutID, err := database.CreateWorkout(workoutType, "completed")
				if err != nil {
					return fmt.Errorf("error creating workout: %v", err)
				}

				err = database.SaveExercisesForWorkout(workoutID, exercises)
				if err != nil {
					return fmt.Errorf("error saving exercises: %v", err)
				}

				fmt.Println("Great workout!")
				return nil
			} else if finished == "no" || finished == "n" {
				utils.ClearScreen()
				fmt.Println("------------ Next Exercise ------------")
				break
			} else if finished == "review" || finished == "r" {
				fmt.Printf("\n========== Current Workout (%d exercises) ==========\n\n", len(exercises))
				utils.PrintExercises(exercises)
				fmt.Println("\n===================================================")
			} else {
				fmt.Println("Please answer 'yes', 'no', or 'review'")
			}
		}
	}
}

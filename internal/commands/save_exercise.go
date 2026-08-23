package commands

import (
	"fmt"
	"os"
	"strings"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/ui"
	"terminal_fit_recorder/internal/utils"
)

type SaveExerciseCommand struct {
	InputProvider ui.InputProvider
	FilePath      string
	Args          []string
}

func NewSaveExerciseCommand(args ...string) *SaveExerciseCommand {
	cmd := &SaveExerciseCommand{
		InputProvider: ui.NewDefaultInputProvider(),
		Args:          args,
	}

	if len(args) == 1 && args[0] != "--file" && args[0] != "-f" {
		cmd.FilePath = args[0]
	} else if len(args) == 2 && (args[0] == "--file" || args[0] == "-f") {
		cmd.FilePath = args[1]
	}

	return cmd
}

func (cmd *SaveExerciseCommand) Name() string {
	return "save exercise"
}

func (cmd *SaveExerciseCommand) Validate() error {
	if len(cmd.Args) > 0 && cmd.FilePath == "" {
		return fmt.Errorf("usage: terminal_fit_recorder exercise save [--file path]")
	}

	if cmd.FilePath == "" {
		return nil
	}

	info, err := os.Stat(cmd.FilePath)
	if err != nil {
		return fmt.Errorf("cannot read workout file %q: %v", cmd.FilePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("workout file %q is a directory", cmd.FilePath)
	}

	return nil
}

func (cmd *SaveExerciseCommand) HelpManual() string {
	return "terminal_fit_recorder exercise save [--file path]\n    Start an interactive session to save a new workout, or import one/many workouts from a JSON file."
}

func (cmd *SaveExerciseCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	if cmd.FilePath != "" {
		return cmd.executeFromFile(database)
	}

	// Get workout type using checkbox
	workoutType, cancelled := cmd.InputProvider.GetInputWithType("Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox)
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
		name, cancelled := cmd.InputProvider.GetInputWithType("Exercise name: ", existingNames, ui.InputTypeAutocomplete)
		if cancelled {
			fmt.Println("\nWorkout cancelled")
			return nil
		}

		// Validate exercise name is not empty
		name = strings.TrimSpace(name)
		if name == "" {
			fmt.Println("Exercise name cannot be empty. Please try again.")
			continue
		}

		// Check if distance is required for this exercise
		requiresDistance := false
		nameLower := strings.ToLower(name)
		for _, keyword := range db.DistanceRequiredKeywords {
			if strings.HasPrefix(nameLower, keyword) {
				requiresDistance = true
				break
			}
		}

		exercise := db.Exercise{
			Name: name,
		}

		// If distance is required, skip weight/reps/sets
		if !requiresDistance {
			weight, cancelled := cmd.InputProvider.GetInputWithType("Weight: ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nWorkout cancelled")
				return nil
			}

			reps, cancelled := cmd.InputProvider.GetInputWithType("Repetitions: ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nWorkout cancelled")
				return nil
			}

			sets, cancelled := cmd.InputProvider.GetInputWithType("Number of sets: ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nWorkout cancelled")
				return nil
			}

			// Convert weight, reps, and sets strings to int
			exercise.Weight = utils.ParseWeight(weight)
			exercise.Repetitions = utils.ParseInt(reps)
			exercise.Sets = utils.ParseInt(sets)
		}

		// Check if duration is required for this exercise
		requiresDuration := false
		for _, keyword := range db.DurationRequiredKeywords {
			if strings.Contains(nameLower, keyword) {
				requiresDuration = true
				break
			}
		}

		if requiresDuration {
			durationStr, cancelled := cmd.InputProvider.GetInputWithType("Duration (minutes): ", nil, ui.InputTypeText)
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

		if requiresDistance {
			distanceStr, cancelled := cmd.InputProvider.GetInputWithType("Distance (meters): ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nWorkout cancelled")
				return nil
			}

			// Parse distance string to int
			distance := utils.ParseInt(distanceStr)
			exercise.Distance = distance
		}

		exercises = append(exercises, exercise)

		// Display recorded exercise based on what fields are set
		if requiresDistance {
			fmt.Printf("\nRecorded: %s - %d meters\n", exercise.Name, exercise.Distance)
		} else {
			if exercise.Duration > 0 {
				fmt.Printf("\nRecorded: %s - %d kg weight, %d reps, %d sets, %.2f minutes\n",
					exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets, exercise.Duration)
			} else {
				fmt.Printf("\nRecorded: %s - %d kg weight, %d reps, %d sets\n",
					exercise.Name, exercise.Weight, exercise.Repetitions, exercise.Sets)
			}
		}

		for {
			finished, cancelled := cmd.InputProvider.GetInputWithType("Finished?", []string{"no", "yes", "review"}, ui.InputTypeCheckbox)
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

func (cmd *SaveExerciseCommand) executeFromFile(database *db.DB) error {
	content, err := os.ReadFile(cmd.FilePath)
	if err != nil {
		return fmt.Errorf("error reading workout file: %v", err)
	}

	workouts, err := api.ParseWorkoutResponses(string(content))
	if err != nil {
		return fmt.Errorf("error parsing workout file: %v", err)
	}

	for _, workout := range workouts {
		workout.Workout.Status = "completed"
	}

	fmt.Printf("\nProcessed %d workout(s) from %s\n", len(workouts), cmd.FilePath)
	for i, workout := range workouts {
		fmt.Printf("\n========== Workout %d of %d ==========\n", i+1, len(workouts))
		fmt.Println(FormatWorkout(workout))
	}

	for {
		choice, cancelled := cmd.InputProvider.GetInputWithType(
			"\nApprove imported workouts?",
			[]string{"approve", "discard", "edit"},
			ui.InputTypeCheckbox,
		)
		if cancelled {
			fmt.Println("\nImport discarded")
			return nil
		}

		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "approve", "yes", "y":
			if err := database.SaveWorkoutsWithExercises(workouts, "completed"); err != nil {
				return fmt.Errorf("error saving imported workouts: %v", err)
			}
			fmt.Printf("Saved %d workout(s)\n", len(workouts))
			return nil
		case "discard", "no", "n":
			fmt.Println("Import discarded")
			return nil
		case "edit", "e":
			fmt.Println("Edit mode for file imports is not implemented yet. Returning to approval prompt.")
		default:
			fmt.Println("Please choose approve, discard, or edit")
		}
	}
}

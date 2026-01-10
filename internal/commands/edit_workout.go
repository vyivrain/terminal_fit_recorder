package commands

import (
	"fmt"
	"strings"
	"time"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/ui"
	"terminal_fit_recorder/internal/utils"
)

type EditWorkoutCommand struct {
	Date time.Time
}

func NewEditWorkoutCommand(date time.Time) *EditWorkoutCommand {
	return &EditWorkoutCommand{Date: date}
}

func (cmd *EditWorkoutCommand) Name() string {
	return "edit workout"
}

func (cmd *EditWorkoutCommand) Validate() error {
	if cmd.Date.IsZero() {
		return fmt.Errorf("date is required")
	}
	return nil
}

func (cmd *EditWorkoutCommand) HelpManual() string {
	return ""
}

func (cmd *EditWorkoutCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	// Get the workout for the specified date
	workout, err := database.GetWorkoutByDate(cmd.Date)
	if err != nil {
		return fmt.Errorf("error fetching workout: %v", err)
	}

	if workout == nil {
		fmt.Printf("No workout found for %s\n", cmd.Date.Format("2006-01-02"))
		return nil
	}

	fmt.Printf("\nEditing workout for %s\n", cmd.Date.Format("2006-01-02"))
	fmt.Printf("Current type: %s\n\n", workout.Workout.WorkoutType)

	// Get workout type using checkbox
	workoutType, cancelled := ui.GetInputWithType("Workout type:", []string{"strength", "cardio"}, ui.InputTypeCheckbox)
	if cancelled {
		fmt.Println("\nEdit cancelled")
		return nil
	}
	workoutType = strings.ToLower(strings.TrimSpace(workoutType))

	// Get existing exercise names for autocomplete
	existingNames, err := database.GetDistinctExerciseNames()
	if err != nil {
		return fmt.Errorf("error fetching exercise names: %v", err)
	}

	// Collect exercises
	fmt.Println("\nEnter exercise details (Ctrl+C to exit, Ctrl+D to cancel editing):")
	var exercises []db.Exercise

	for {
		name, cancelled := ui.GetInputWithType("Exercise name: ", existingNames, ui.InputTypeAutocomplete)
		if cancelled {
			fmt.Println("\nEdit cancelled")
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
			if strings.Contains(nameLower, keyword) {
				requiresDistance = true
				break
			}
		}

		exercise := db.Exercise{
			Name: name,
		}

		// If distance is required, skip weight/reps/sets
		if !requiresDistance {
			weight, cancelled := ui.GetInputWithType("Weight: ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nEdit cancelled")
				return nil
			}

			reps, cancelled := ui.GetInputWithType("Repetitions: ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nEdit cancelled")
				return nil
			}

			sets, cancelled := ui.GetInputWithType("Number of sets: ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nEdit cancelled")
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
			durationStr, cancelled := ui.GetInputWithType("Duration (minutes): ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nEdit cancelled")
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
			distanceStr, cancelled := ui.GetInputWithType("Distance (meters): ", nil, ui.InputTypeText)
			if cancelled {
				fmt.Println("\nEdit cancelled")
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
			finished, cancelled := ui.GetInputWithType("Finished?", []string{"no", "yes"}, ui.InputTypeCheckbox)
			if cancelled {
				fmt.Println("\nEdit cancelled")
				return nil
			}
			finished = strings.ToLower(strings.TrimSpace(finished))

			if finished == "yes" || finished == "y" {
				// Update the workout
				err = database.UpdateWorkout(workout.Workout.ID, workoutType, exercises)
				if err != nil {
					return fmt.Errorf("error updating workout: %v", err)
				}

				fmt.Printf("\nWorkout for %s updated successfully!\n", cmd.Date.Format("2006-01-02"))
				return nil
			} else if finished == "no" || finished == "n" {
				fmt.Println("\n------------ Next Exercise ------------")
				break
			} else {
				fmt.Println("Please answer 'yes' or 'no'")
			}
		}
	}
}

package commands

import (
	"fmt"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type DeleteLastWorkoutCommand struct{}

func NewDeleteLastWorkoutCommand() *DeleteLastWorkoutCommand {
	return &DeleteLastWorkoutCommand{}
}

func (cmd *DeleteLastWorkoutCommand) Name() string {
	return "delete last workout"
}

func (cmd *DeleteLastWorkoutCommand) Validate() error {
	return nil
}

func (cmd *DeleteLastWorkoutCommand) HelpManual() string {
	return ""
}

func (cmd *DeleteLastWorkoutCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	// First, get the last workout to show the date
	workout, err := database.GetLastWorkout()
	if err != nil {
		return fmt.Errorf("error fetching last workout: %v", err)
	}

	if workout == nil {
		fmt.Println("No workouts to delete")
		return nil
	}

	// Delete the workout
	err = database.DeleteLastWorkout()
	if err != nil {
		return fmt.Errorf("error deleting workout: %v", err)
	}

	fmt.Printf("Workout for %s was deleted\n", workout.Workout.WorkoutDate.Format("2006-01-02"))
	return nil
}

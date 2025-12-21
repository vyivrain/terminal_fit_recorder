package commands

import (
	"fmt"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type EditWorkoutStatusCommand struct {
	newStatus string
}

func NewEditWorkoutStatusCommand(status string) *EditWorkoutStatusCommand {
	return &EditWorkoutStatusCommand{
		newStatus: status,
	}
}

func (cmd *EditWorkoutStatusCommand) Name() string {
	return "edit workout status"
}

func (cmd *EditWorkoutStatusCommand) Validate() error {
	if cmd.newStatus != "planned" && cmd.newStatus != "completed" {
		return fmt.Errorf("status must be 'planned' or 'completed', got: %s", cmd.newStatus)
	}
	return nil
}

func (cmd *EditWorkoutStatusCommand) HelpManual() string {
	return ""
}

func (cmd *EditWorkoutStatusCommand) Execute(database *db.DB, ollamaClient *api.Client) error {
	// Get the last workout to show which one we're updating
	lastWorkout, err := database.GetLastWorkout()
	if err != nil {
		return fmt.Errorf("error fetching last workout: %v", err)
	}

	if lastWorkout == nil {
		return fmt.Errorf("no workouts found to update")
	}

	// Update the status
	err = database.UpdateLastWorkoutStatus(cmd.newStatus)
	if err != nil {
		return fmt.Errorf("error updating workout status: %v", err)
	}

	fmt.Printf("✓ Updated workout from %s to '%s'\n",
		lastWorkout.Workout.WorkoutDate.Format("2006-01-02"),
		cmd.newStatus)
	fmt.Printf("  Type: %s\n", lastWorkout.Workout.WorkoutType)
	fmt.Printf("  Exercises: %d\n", len(lastWorkout.Exercises))

	return nil
}

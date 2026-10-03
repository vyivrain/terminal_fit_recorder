package commands

import (
	"fmt"
	"time"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

// EditWorkoutNotesCommand sets the notes on a workout — typically what the AI
// generator changed, improved, or reduced and why. It is the primary way
// automations such as Hermes attach that commentary after a workout is
// planned or completed. A zero Date targets the most recent workout.
type EditWorkoutNotesCommand struct {
	Date  time.Time
	notes string
}

func NewEditWorkoutNotesCommand(date time.Time, notes string) *EditWorkoutNotesCommand {
	return &EditWorkoutNotesCommand{Date: date, notes: notes}
}

func (cmd *EditWorkoutNotesCommand) Name() string {
	return "edit workout notes"
}

func (cmd *EditWorkoutNotesCommand) Validate() error {
	return nil
}

func (cmd *EditWorkoutNotesCommand) HelpManual() string {
	return ""
}

func (cmd *EditWorkoutNotesCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	var workout *db.WorkoutWithExercises
	var err error
	if cmd.Date.IsZero() {
		workout, err = database.GetLastWorkout()
	} else {
		workout, err = database.GetWorkoutByDate(cmd.Date)
	}
	if err != nil {
		return fmt.Errorf("error fetching workout: %v", err)
	}
	if workout == nil {
		return fmt.Errorf("no workout found to update")
	}

	if err := database.SetWorkoutNotes(workout.Workout.ID, cmd.notes); err != nil {
		return fmt.Errorf("error updating workout notes: %v", err)
	}

	fmt.Printf("✓ Updated notes for workout on %s\n", workout.Workout.WorkoutDate.Format("2006-01-02"))
	return nil
}

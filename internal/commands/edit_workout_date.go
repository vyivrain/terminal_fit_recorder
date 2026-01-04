package commands

import (
	"fmt"
	"time"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type EditWorkoutDateCommand struct {
	OldDate time.Time
	NewDate time.Time
}

func NewEditWorkoutDateCommand(oldDate time.Time, newDate time.Time) *EditWorkoutDateCommand {
	return &EditWorkoutDateCommand{
		OldDate: oldDate,
		NewDate: newDate,
	}
}

func (cmd *EditWorkoutDateCommand) Name() string {
	return "edit workout date"
}

func (cmd *EditWorkoutDateCommand) Validate() error {
	return nil
}

func (cmd *EditWorkoutDateCommand) HelpManual() string {
	return ""
}

func (cmd *EditWorkoutDateCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	err := database.UpdateWorkoutDate(cmd.OldDate, cmd.NewDate)
	if err != nil {
		return fmt.Errorf("error updating workout date: %v", err)
	}

	fmt.Printf("Workout date changed from %s to %s\n",
		cmd.OldDate.Format("2006-01-02"),
		cmd.NewDate.Format("2006-01-02"))
	return nil
}

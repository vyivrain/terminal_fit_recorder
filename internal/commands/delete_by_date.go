package commands

import (
	"fmt"
	"time"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type DeleteByDateCommand struct {
	Date time.Time
}

func NewDeleteByDateCommand(date time.Time) *DeleteByDateCommand {
	return &DeleteByDateCommand{Date: date}
}

func (cmd *DeleteByDateCommand) Name() string {
	return "delete workout by date"
}

func (cmd *DeleteByDateCommand) Validate() error {
	return nil
}

func (cmd *DeleteByDateCommand) HelpManual() string {
	return ""
}

func (cmd *DeleteByDateCommand) Execute(database *db.DB, ollamaClient *api.Client) error {
	err := database.DeleteWorkoutByDate(cmd.Date)
	if err != nil {
		return fmt.Errorf("error deleting workout: %v", err)
	}

	fmt.Printf("Workout for %s was deleted\n", cmd.Date.Format("2006-01-02"))
	return nil
}

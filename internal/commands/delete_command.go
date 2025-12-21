package commands

import (
	"fmt"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/utils"
)

// DeleteCommand is a wrapper that routes to specific delete commands
type DeleteCommand struct {
	Args    []string
	command Command
}

func NewDeleteCommand(args []string) *DeleteCommand {
	return &DeleteCommand{Args: args}
}

func (cmd *DeleteCommand) Name() string {
	return "delete"
}

func (cmd *DeleteCommand) Validate() error {
	if len(cmd.Args) < 4 {
		return fmt.Errorf("usage: terminal_fit_recorder exercise delete <last|date <date>>")
	}

	deleteTarget := cmd.Args[3]
	switch deleteTarget {
	case "last":
		cmd.command = NewDeleteLastWorkoutCommand()
		return cmd.command.Validate()
	case "date":
		if len(cmd.Args) < 5 {
			return fmt.Errorf("usage: terminal_fit_recorder exercise delete date <DD-MM-YY>")
		}
		dateStr := cmd.Args[4]
		date, err := utils.ParseEUDate(dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format. Use DD-MM-YY (e.g., 01-10-25): %v", err)
		}
		cmd.command = NewDeleteByDateCommand(date)
		return cmd.command.Validate()
	default:
		return fmt.Errorf("unknown delete target: %s. Use 'last' or 'date <DD-MM-YY>'", deleteTarget)
	}
}

func (cmd *DeleteCommand) Execute(database *db.DB, ollamaClient *api.Client) error {
	return cmd.command.Execute(database, ollamaClient)
}

func (cmd *DeleteCommand) HelpManual() string {
	return "terminal_fit_recorder exercise delete last\n    Delete the most recent workout.\n\nterminal_fit_recorder exercise delete date <DD-MM-YY>\n    Delete workout by specific date (e.g., 01-10-25)."
}

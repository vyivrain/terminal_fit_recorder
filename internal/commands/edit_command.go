package commands

import (
	"fmt"
	"strings"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/utils"
)

// EditCommand is a wrapper that routes to specific edit commands
type EditCommand struct {
	Args    []string
	command Command
}

func NewEditCommand(args []string) *EditCommand {
	return &EditCommand{Args: args}
}

func (cmd *EditCommand) Name() string {
	return "edit"
}

func (cmd *EditCommand) Validate() error {
	if len(cmd.Args) < 4 {
		return fmt.Errorf("usage: terminal_fit_recorder exercise edit <DD-MM-YY|date <old_date> <new_date>|last status <planned|completed>>")
	}

	editTarget := cmd.Args[3]

	// Check if it's "edit last status <planned|completed>"
	if editTarget == "last" {
		if len(cmd.Args) < 6 {
			return fmt.Errorf("usage: terminal_fit_recorder exercise edit last status <planned|completed>")
		}
		if cmd.Args[4] != "status" {
			return fmt.Errorf("usage: terminal_fit_recorder exercise edit last status <planned|completed>")
		}
		newStatus := strings.ToLower(cmd.Args[5])
		cmd.command = NewEditWorkoutStatusCommand(newStatus)
		return cmd.command.Validate()
	}

	// Check if it's "edit date <old> <new>"
	if editTarget == "date" {
		if len(cmd.Args) < 6 {
			return fmt.Errorf("usage: terminal_fit_recorder exercise edit date <DD-MM-YY> <DD-MM-YY>")
		}
		oldDateStr := cmd.Args[4]
		newDateStr := cmd.Args[5]

		oldDate, err := utils.ParseEUDate(oldDateStr)
		if err != nil {
			return fmt.Errorf("invalid old date format. Use DD-MM-YY (e.g., 31-12-25): %v", err)
		}

		newDate, err := utils.ParseEUDate(newDateStr)
		if err != nil {
			return fmt.Errorf("invalid new date format. Use DD-MM-YY (e.g., 31-12-25): %v", err)
		}

		cmd.command = NewEditWorkoutDateCommand(oldDate, newDate)
		return cmd.command.Validate()
	}

	// Otherwise it's "edit <date>" for editing workout content
	date, err := utils.ParseEUDate(editTarget)
	if err != nil {
		return fmt.Errorf("invalid date format. Use DD-MM-YY (e.g., 31-12-25): %v", err)
	}
	cmd.command = NewEditWorkoutCommand(date)
	return cmd.command.Validate()
}

func (cmd *EditCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	return cmd.command.Execute(database, ollamaClient)
}

func (cmd *EditCommand) HelpManual() string {
	return "terminal_fit_recorder exercise edit <DD-MM-YY>\n    Edit a workout by date (e.g., 31-12-25).\n\nterminal_fit_recorder exercise edit date <DD-MM-YY> <DD-MM-YY>\n    Change workout date from old to new date.\n\nterminal_fit_recorder exercise edit last status <planned|completed>\n    Update the status of the last workout."
}

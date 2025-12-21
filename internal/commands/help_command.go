package commands

import (
	"fmt"
	"strings"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type HelpCommand struct{}

func NewHelpCommand() *HelpCommand {
	return &HelpCommand{}
}

func (cmd *HelpCommand) Name() string {
	return "help"
}

func (cmd *HelpCommand) Validate() error {
	return nil
}

func (cmd *HelpCommand) HelpManual() string {
	return "terminal_fit_recorder exercise help\n    Display this help message with all available commands."
}

func (cmd *HelpCommand) Execute(database *db.DB, ollamaClient *api.Client) error {
	var output strings.Builder

	output.WriteString("terminal_fit_recorder - Personal fitness workout recorder tool\n\n")
	output.WriteString("Usage: terminal_fit_recorder exercise <command> [arguments]\n\n")
	output.WriteString("Commands are:\n\n")

	// Create instances of all commands to get their help manuals
	commands := []Command{
		NewSaveExerciseCommand(),
		NewShowLastWorkoutCommand(),
		NewShowAllWorkoutsCommand(),
		NewEditCommand([]string{}),
		NewDeleteCommand([]string{}),
		NewGenerateCommandWrapper([]string{}),
		NewHelpCommand(),
	}

	for _, command := range commands {
		helpText := command.HelpManual()
		if helpText != "" {
			output.WriteString(helpText)
			output.WriteString("\n\n")
		}
	}

	fmt.Print(output.String())
	return nil
}

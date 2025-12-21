package commands

import (
	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/utils"
)

// GenerateCommandWrapper handles argument parsing for generate command
type GenerateCommandWrapper struct {
	Args    []string
	command *GenerateCommand
}

func NewGenerateCommandWrapper(args []string) *GenerateCommandWrapper {
	return &GenerateCommandWrapper{Args: args}
}

func (cmd *GenerateCommandWrapper) Name() string {
	return "generate"
}

func (cmd *GenerateCommandWrapper) Validate() error {
	exerciseCount := 0 // 0 means AI decides
	if len(cmd.Args) >= 4 {
		count, err := utils.ParseExerciseCount(cmd.Args[3])
		if err != nil {
			return err
		}
		exerciseCount = count
	}
	cmd.command = NewGenerateCommand(exerciseCount)
	return cmd.command.Validate()
}

func (cmd *GenerateCommandWrapper) Execute(database *db.DB, ollamaClient *api.Client) error {
	return cmd.command.Execute(database, ollamaClient)
}

func (cmd *GenerateCommandWrapper) HelpManual() string {
	return "terminal_fit_recorder exercise generate [count]\n    Generate a workout plan using AI. Optionally specify number of exercises (default: AI decides)."
}

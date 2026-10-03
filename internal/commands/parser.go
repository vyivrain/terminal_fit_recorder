package commands

import (
	"fmt"
)

// CommandFactory is a function that creates a command from arguments
type CommandFactory func(args []string) Command

// commandMap defines the hierarchical command structure
var commandMap = map[string]map[string]CommandFactory{
	"exercise": {
		"init":     func(args []string) Command { return NewInitCommand() },
		"save":     func(args []string) Command { return NewSaveExerciseCommand(args[3:]...) },
		"last":     func(args []string) Command { return NewShowLastWorkoutCommand() },
		"all":      func(args []string) Command { return NewShowAllWorkoutsCommand() },
		"edit":     func(args []string) Command { return NewEditCommand(args) },
		"delete":   func(args []string) Command { return NewDeleteCommand(args) },
		"generate": func(args []string) Command { return NewGenerateCommandWrapper(args) },
		"plan":     func(args []string) Command { return NewPlanCommand(args[3:]...) },
		"videos":   func(args []string) Command { return NewVideosCommand(args[3:]...) },
		"help":     func(args []string) Command { return NewHelpCommand() },
	},
	"profile": {
		"create":   func(args []string) Command { return NewCreateProfileCommand(args[3:]...) },
		"use":      func(args []string) Command { return NewUseProfileCommand(args[3:]...) },
		"list":     func(args []string) Command { return NewListProfilesCommand(args[3:]...) },
		"describe": func(args []string) Command { return NewDescribeProfileCommand(args[3:]...) },
		"help":     func(args []string) Command { return NewHelpCommand() },
	},
	"ai": {
		"test": func(args []string) Command { return NewAITestCommand(args[3:]...) },
		"help": func(args []string) Command { return NewHelpCommand() },
	},
}

func ParseArgs(args []string) (Command, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("usage: terminal_fit_recorder <exercise|profile|ai> <command>")
	}

	action := args[1]
	target := args[2]

	// Look up command in the map
	subCommands, ok := commandMap[action]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", action)
	}

	factory, ok := subCommands[target]
	if !ok {
		return nil, fmt.Errorf("unknown %s subcommand: %s", action, target)
	}

	// Create command using factory
	cmd := factory(args)

	// Validate command
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	return cmd, nil
}

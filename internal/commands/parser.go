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
		"save":     func(args []string) Command { return NewSaveExerciseCommand() },
		"last":     func(args []string) Command { return NewShowLastWorkoutCommand() },
		"all":      func(args []string) Command { return NewShowAllWorkoutsCommand() },
		"edit":     func(args []string) Command { return NewEditCommand(args) },
		"delete":   func(args []string) Command { return NewDeleteCommand(args) },
		"generate": func(args []string) Command { return NewGenerateCommandWrapper(args) },
		"help":     func(args []string) Command { return NewHelpCommand() },
	},
}

func ParseArgs(args []string) (Command, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("usage: terminal_fit_recorder <command> <subcommand>")
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

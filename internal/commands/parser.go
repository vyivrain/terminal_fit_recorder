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
		"help":     func(args []string) Command { return NewHelpCommand("exercise") },
	},
	"profile": {
		"create":   func(args []string) Command { return NewCreateProfileCommand(args[3:]...) },
		"use":      func(args []string) Command { return NewUseProfileCommand(args[3:]...) },
		"list":     func(args []string) Command { return NewListProfilesCommand(args[3:]...) },
		"describe": func(args []string) Command { return NewDescribeProfileCommand(args[3:]...) },
		"help":     func(args []string) Command { return NewHelpCommand("profile") },
	},
	"trainer": {
		"review": func(args []string) Command { return NewTrainerReviewCommand(args[3:]...) },
		"show":   func(args []string) Command { return NewTrainerShowCommand(args[3:]...) },
		"help":   func(args []string) Command { return NewHelpCommand("trainer") },
	},
	"ai": {
		"test": func(args []string) Command { return NewAITestCommand(args[3:]...) },
		"help": func(args []string) Command { return NewHelpCommand("ai") },
	},
}

const helpHint = "Run 'terminal_fit_recorder help' to see all commands."

func ParseArgs(args []string) (Command, error) {
	if len(args) < 2 {
		return NewHelpCommand(), nil
	}

	action := args[1]
	switch action {
	case "help", "-h", "--help":
		if len(args) > 2 {
			if _, ok := commandMap[args[2]]; ok {
				return NewHelpCommand(args[2]), nil
			}
		}
		return NewHelpCommand(), nil
	}

	// Look up command in the map
	subCommands, ok := commandMap[action]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s\n%s", action, helpHint)
	}

	// A bare group, or a group with -h/--help, shows that group's commands
	if len(args) < 3 || args[2] == "-h" || args[2] == "--help" {
		return NewHelpCommand(action), nil
	}

	target := args[2]
	factory, ok := subCommands[target]
	if !ok {
		return nil, fmt.Errorf("unknown %s subcommand: %s\nRun 'terminal_fit_recorder help %s' to see its commands.", action, target, action)
	}

	// Create command using factory
	cmd := factory(args)

	// Validate command
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	return cmd, nil
}

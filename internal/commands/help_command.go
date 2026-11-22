package commands

import (
	"fmt"
	"strings"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

const appName = "terminal_fit_recorder"

// helpSection groups the commands of one top-level command group
type helpSection struct {
	group    string
	title    string
	commands func() []Command
}

var helpSections = []helpSection{
	{
		group: "exercise",
		title: "Workouts",
		commands: func() []Command {
			return []Command{
				NewInitCommand(),
				NewSaveExerciseCommand(),
				NewShowLastWorkoutCommand(),
				NewShowAllWorkoutsCommand(),
				NewEditCommand([]string{}),
				NewDeleteCommand([]string{}),
				NewGenerateCommandWrapper([]string{}),
				NewPlanCommand(),
				NewVideosCommand(),
			}
		},
	},
	{
		group: "profile",
		title: "Profiles",
		commands: func() []Command {
			return []Command{
				NewCreateProfileCommand(),
				NewUseProfileCommand(),
				NewListProfilesCommand(),
				NewDescribeProfileCommand(),
			}
		},
	},
	{
		group: "ai",
		title: "AI diagnostics",
		commands: func() []Command {
			return []Command{NewAITestCommand()}
		},
	},
}

// HelpCommand prints usage for every group, or only for group when it is set
type HelpCommand struct {
	group string
}

func NewHelpCommand(group ...string) *HelpCommand {
	cmd := &HelpCommand{}
	if len(group) > 0 {
		cmd.group = group[0]
	}
	return cmd
}

func (cmd *HelpCommand) Name() string {
	return "help"
}

func (cmd *HelpCommand) Validate() error {
	return nil
}

func (cmd *HelpCommand) HelpManual() string {
	return appName + " help [exercise|profile|ai]\n    Show this help, or only one command group."
}

func (cmd *HelpCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	fmt.Print(cmd.Render())
	return nil
}

// Render builds the help text; Execute prints it
func (cmd *HelpCommand) Render() string {
	var out strings.Builder

	if cmd.group == "" {
		out.WriteString(appName + " - keyboard-first workout tracker with AI planning\n\n")
		out.WriteString("Usage:\n")
		out.WriteString("  " + appName + "                          Open the interactive app (recommended)\n")
		out.WriteString("  " + appName + " <group> <command> [args]\n")
		out.WriteString("  " + appName + " mcp                      Run an MCP server so AI agents can log workouts\n")
		out.WriteString("  " + appName + " help [group]             Show help for all or one group\n")
	}

	for _, section := range helpSections {
		if cmd.group != "" && cmd.group != section.group {
			continue
		}
		fmt.Fprintf(&out, "\n%s:\n", section.title)
		for _, command := range section.commands() {
			writeManual(&out, command.HelpManual())
		}
	}

	if cmd.group == "" {
		out.WriteString("\nEnvironment (used by 'exercise generate' only):\n")
		out.WriteString("  TERMINAL_FIT_RECORDER_OLLAMA_HOST      Ollama URL, e.g. http://localhost:11434 or https://ollama.com\n")
		out.WriteString("  TERMINAL_FIT_RECORDER_OLLAMA_MODEL     Model name, e.g. glm-5.3-flash:cloud\n")
		out.WriteString("  TERMINAL_FIT_RECORDER_OLLAMA_API_KEY   API key for ollama.com (falls back to OLLAMA_API_KEY)\n")
		out.WriteString("  TERMINAL_FIT_RECORDER_OLLAMA_PROMPT    Replace the built-in coaching prompt\n")
		out.WriteString("\nThe interactive app and 'exercise plan' use your signed-in opencode, codex, or claude CLI instead.\n")
		out.WriteString("\nData: ~/.terminal_fit_recorder/exercises.db\n")
	}

	return out.String()
}

// writeManual prints a HelpManual block without the repeated app name, indented under its section
func writeManual(out *strings.Builder, manual string) {
	if manual == "" {
		return
	}
	for line := range strings.SplitSeq(manual, "\n") {
		if line == "" {
			continue
		}
		if usage, ok := strings.CutPrefix(line, appName+" "); ok {
			out.WriteString("\n  " + usage + "\n")
		} else {
			out.WriteString("  " + line + "\n")
		}
	}
}

package commands

import (
	"fmt"
	"strings"

	"terminal_fit_recorder/internal/api"
	"terminal_fit_recorder/internal/db"
)

type CreateProfileCommand struct {
	args []string
	name string
}

func NewCreateProfileCommand(args ...string) *CreateProfileCommand {
	return &CreateProfileCommand{args: args}
}

func (cmd *CreateProfileCommand) Name() string { return "create profile" }

func (cmd *CreateProfileCommand) Validate() error {
	if len(cmd.args) != 1 || strings.TrimSpace(cmd.args[0]) == "" {
		return fmt.Errorf("usage: terminal_fit_recorder profile create <name>")
	}
	cmd.name = strings.TrimSpace(cmd.args[0])
	return nil
}

func (cmd *CreateProfileCommand) HelpManual() string {
	return "terminal_fit_recorder profile create <name>\n    Create an empty profile and switch to it."
}

func (cmd *CreateProfileCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	profile, err := database.CreateProfile(cmd.name)
	if err != nil {
		return fmt.Errorf("error creating profile: %v", err)
	}
	fmt.Printf("Created and selected profile %q\n", profile.Name)
	return nil
}

type UseProfileCommand struct {
	args []string
	name string
}

func NewUseProfileCommand(args ...string) *UseProfileCommand {
	return &UseProfileCommand{args: args}
}

func (cmd *UseProfileCommand) Name() string { return "use profile" }

func (cmd *UseProfileCommand) Validate() error {
	if len(cmd.args) != 1 || strings.TrimSpace(cmd.args[0]) == "" {
		return fmt.Errorf("usage: terminal_fit_recorder profile use <name>")
	}
	cmd.name = strings.TrimSpace(cmd.args[0])
	return nil
}

func (cmd *UseProfileCommand) HelpManual() string {
	return "terminal_fit_recorder profile use <name>\n    Switch the active profile."
}

func (cmd *UseProfileCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	profile, err := database.UseProfile(cmd.name)
	if err != nil {
		return fmt.Errorf("error selecting profile: %v", err)
	}
	fmt.Printf("Selected profile %q\n", profile.Name)
	return nil
}

type DescribeProfileCommand struct {
	args        []string
	description string
}

func NewDescribeProfileCommand(args ...string) *DescribeProfileCommand {
	return &DescribeProfileCommand{args: args}
}

func (cmd *DescribeProfileCommand) Name() string { return "describe profile" }

func (cmd *DescribeProfileCommand) Validate() error {
	cmd.description = strings.TrimSpace(strings.Join(cmd.args, " "))
	return nil
}

func (cmd *DescribeProfileCommand) HelpManual() string {
	return "terminal_fit_recorder profile describe <text>\n    Set the active profile's description (habits, injuries, preferences) used as AI context. An empty <text> clears it."
}

func (cmd *DescribeProfileCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	profile, err := database.GetActiveProfile()
	if err != nil {
		return fmt.Errorf("error loading active profile: %v", err)
	}
	if profile == nil {
		return fmt.Errorf("no active profile")
	}

	if _, err := database.SetProfileDescription(profile.ID, cmd.description); err != nil {
		return fmt.Errorf("error updating profile description: %v", err)
	}
	fmt.Printf("Updated description for profile %q\n", profile.Name)
	return nil
}

type ListProfilesCommand struct {
	args []string
}

func NewListProfilesCommand(args ...string) *ListProfilesCommand {
	return &ListProfilesCommand{args: args}
}

func (cmd *ListProfilesCommand) Name() string { return "list profiles" }

func (cmd *ListProfilesCommand) Validate() error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("usage: terminal_fit_recorder profile list")
	}
	return nil
}

func (cmd *ListProfilesCommand) HelpManual() string {
	return "terminal_fit_recorder profile list\n    List profiles; the active profile is marked with *."
}

func (cmd *ListProfilesCommand) Execute(database *db.DB, ollamaClient api.OllamaClient) error {
	profiles, err := database.GetProfiles()
	if err != nil {
		return fmt.Errorf("error listing profiles: %v", err)
	}

	for _, profile := range profiles {
		marker := " "
		if profile.IsActive {
			marker = "*"
		}
		fmt.Printf("%s %s\n", marker, profile.Name)
		if description := strings.TrimSpace(profile.Description); description != "" {
			fmt.Printf("    %s\n", description)
		}
	}
	return nil
}

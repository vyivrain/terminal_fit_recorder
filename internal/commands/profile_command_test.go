package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProfileCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want any
	}{
		{name: "create", args: []string{"terminal_fit_recorder", "profile", "create", "guest"}, want: &CreateProfileCommand{}},
		{name: "use", args: []string{"terminal_fit_recorder", "profile", "use", "mine"}, want: &UseProfileCommand{}},
		{name: "list", args: []string{"terminal_fit_recorder", "profile", "list"}, want: &ListProfilesCommand{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command, err := ParseArgs(test.args)
			require.NoError(t, err)
			require.IsType(t, test.want, command)
		})
	}
}

func TestParseProfileCommandsRejectsInvalidArguments(t *testing.T) {
	_, err := ParseArgs([]string{"terminal_fit_recorder", "profile", "create"})
	require.ErrorContains(t, err, "profile create <name>")

	_, err = ParseArgs([]string{"terminal_fit_recorder", "profile", "use", "mine", "extra"})
	require.ErrorContains(t, err, "profile use <name>")

	_, err = ParseArgs([]string{"terminal_fit_recorder", "profile", "list", "extra"})
	require.ErrorContains(t, err, "profile list")
}

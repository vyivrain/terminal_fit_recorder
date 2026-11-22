package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgsHelpAliases(t *testing.T) {
	tests := []struct {
		args  []string
		group string
	}{
		{args: []string{"app", "help"}, group: ""},
		{args: []string{"app", "--help"}, group: ""},
		{args: []string{"app", "-h"}, group: ""},
		{args: []string{"app", "help", "profile"}, group: "profile"},
		{args: []string{"app", "help", "nonsense"}, group: ""},
		{args: []string{"app", "exercise"}, group: "exercise"},
		{args: []string{"app", "exercise", "help"}, group: "exercise"},
		{args: []string{"app", "ai", "--help"}, group: "ai"},
	}

	for _, tt := range tests {
		cmd, err := ParseArgs(tt.args)
		require.NoError(t, err, tt.args)
		help, ok := cmd.(*HelpCommand)
		require.True(t, ok, "%v should parse to help", tt.args)
		assert.Equal(t, tt.group, help.group, tt.args)
	}
}

func TestParseArgsUnknownCommandPointsToHelp(t *testing.T) {
	_, err := ParseArgs([]string{"app", "nope"})
	assert.ErrorContains(t, err, "terminal_fit_recorder help")

	_, err = ParseArgs([]string{"app", "profile", "nope"})
	assert.ErrorContains(t, err, "terminal_fit_recorder help profile")
}

func TestHelpRenderCoversEveryGroup(t *testing.T) {
	full := NewHelpCommand().Render()
	for _, want := range []string{"exercise init", "exercise save", "profile create", "ai test", "mcp", "TERMINAL_FIT_RECORDER_OLLAMA_API_KEY", "exercises.db"} {
		assert.Contains(t, full, want)
	}

	profileOnly := NewHelpCommand("profile").Render()
	assert.Contains(t, profileOnly, "profile create")
	assert.NotContains(t, profileOnly, "exercise save")
	assert.NotContains(t, profileOnly, "TERMINAL_FIT_RECORDER_OLLAMA_HOST")
}

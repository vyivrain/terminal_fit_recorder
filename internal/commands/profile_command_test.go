package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"terminal_fit_recorder/internal/testutil"
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
		{name: "describe", args: []string{"terminal_fit_recorder", "profile", "describe", "Bad", "knee"}, want: &DescribeProfileCommand{}},
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

type DescribeProfileCommandTestSuite struct {
	testutil.DBTestSuite
}

func (s *DescribeProfileCommandTestSuite) TestSetsActiveProfileDescription() {
	cmd := NewDescribeProfileCommand("Bad", "left", "knee", "—", "avoid", "deep", "squats")
	require.NoError(s.T(), cmd.Validate())
	require.NoError(s.T(), cmd.Execute(s.DB, nil))

	active, err := s.DB.GetActiveProfile()
	require.NoError(s.T(), err)
	require.Equal(s.T(), "Bad left knee — avoid deep squats", active.Description)
}

func (s *DescribeProfileCommandTestSuite) TestEmptyTextClearsDescription() {
	require.NoError(s.T(), NewDescribeProfileCommand("Bad knee").Execute(s.DB, nil))
	cmd := NewDescribeProfileCommand()
	require.NoError(s.T(), cmd.Validate())
	require.NoError(s.T(), cmd.Execute(s.DB, nil))

	active, err := s.DB.GetActiveProfile()
	require.NoError(s.T(), err)
	require.Empty(s.T(), active.Description)
}

func TestDescribeProfileCommandTestSuite(t *testing.T) {
	suite.Run(t, new(DescribeProfileCommandTestSuite))
}

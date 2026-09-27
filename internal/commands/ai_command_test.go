package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"terminal_fit_recorder/internal/testutil"
)

type AITestCommandSuite struct {
	testutil.DBTestSuite
}

func (s *AITestCommandSuite) TestName() {
	assert.Equal(s.T(), "ai test", NewAITestCommand().Name())
}

func (s *AITestCommandSuite) TestHelpManualMentionsObservability() {
	help := NewAITestCommand().HelpManual()
	assert.Contains(s.T(), help, "ai test")
	assert.Contains(s.T(), help, "observability")
}

func (s *AITestCommandSuite) TestValidateRejectsExtraArguments() {
	require.Error(s.T(), NewAITestCommand("a", "b").Validate())
	require.NoError(s.T(), NewAITestCommand("opencode-go/glm-5.3").Validate())
	require.NoError(s.T(), NewAITestCommand().Validate())
}

func (s *AITestCommandSuite) TestResolveModelDefaultsToSelected() {
	cmd := NewAITestCommand()
	require.NoError(s.T(), cmd.Validate())

	model, err := cmd.resolveModel(s.DB)
	require.NoError(s.T(), err)
	// Seed data selects GLM 5.3 as the default model.
	assert.True(s.T(), model.IsSelected)
	assert.Equal(s.T(), "opencode-go/glm-5.3", model.ModelID)
}

func (s *AITestCommandSuite) TestResolveModelSelectsNamedModelWithoutChangingSelection() {
	cmd := NewAITestCommand("opencode-go/deepseek-v4-flash")
	require.NoError(s.T(), cmd.Validate())

	model, err := cmd.resolveModel(s.DB)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "opencode-go/deepseek-v4-flash", model.ModelID)
	assert.False(s.T(), model.IsSelected, "testing a model must not change the selection")

	// The persisted selection is untouched.
	selected, err := s.DB.GetSelectedAIModel()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "opencode-go/glm-5.3", selected.ModelID)
}

func (s *AITestCommandSuite) TestResolveModelRejectsUnknownID() {
	cmd := NewAITestCommand("does-not-exist")
	require.NoError(s.T(), cmd.Validate())

	_, err := cmd.resolveModel(s.DB)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "does-not-exist")
}

func TestAITestCommandSuite(t *testing.T) {
	suite.Run(t, new(AITestCommandSuite))
}

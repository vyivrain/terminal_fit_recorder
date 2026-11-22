package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

type fakeTrainerReviewer struct {
	guidance         string
	previousGuidance string
	calls            int
}

func (reviewer *fakeTrainerReviewer) ReviewTraining(_ context.Context, _ db.AIModel, _ []db.WorkoutWithExercises, previousGuidance, _ string) (string, error) {
	reviewer.calls++
	reviewer.previousGuidance = previousGuidance
	return reviewer.guidance, nil
}

func TestTrainerReviewBuildsOnPreviousGuidance(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	first := &fakeTrainerReviewer{guidance: "Week 1: add a cardio day"}
	cmd := NewTrainerReviewCommand()
	cmd.reviewer = first
	require.NoError(t, cmd.Validate())
	require.NoError(t, cmd.Execute(database, nil))
	require.Empty(t, first.previousGuidance, "the first review has nothing to build on")

	second := &fakeTrainerReviewer{guidance: "Week 2: cardio is back, now push bench"}
	cmd = NewTrainerReviewCommand()
	cmd.reviewer = second
	require.NoError(t, cmd.Execute(database, nil))
	require.Equal(t, "Week 1: add a cardio day", second.previousGuidance)

	guidance, err := latestTrainerGuidance(database)
	require.NoError(t, err)
	require.Equal(t, "Week 2: cardio is back, now push bench", guidance)
}

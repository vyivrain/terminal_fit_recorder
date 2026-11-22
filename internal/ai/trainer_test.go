package ai

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

func TestReviewTrainingReturnsGuidance(t *testing.T) {
	model := db.AIModel{Provider: db.ModelProviderCodex, ModelID: "gpt-5-codex", DisplayName: "Codex"}
	executor := &fakeExecutor{t: t, output: `{"accepted":true,"message":"","guidance":"- Bench stalled at 60 kg x 8: move to 62.5 kg x 6"}`}
	generator := &CLIGenerator{executor: executor, now: time.Now}

	guidance, err := generator.ReviewTraining(context.Background(), model, nil, "", "")
	require.NoError(t, err)
	require.Equal(t, "- Bench stalled at 60 kg x 8: move to 62.5 kg x 6", guidance)
	require.Equal(t, trainerResponseSchema, executor.schema)
}

func TestReviewTrainingRejectsEmptyOrDeclinedReviews(t *testing.T) {
	model := db.AIModel{Provider: db.ModelProviderCodex, ModelID: "gpt-5-codex", DisplayName: "Codex"}

	declined := &CLIGenerator{executor: &fakeExecutor{output: `{"accepted":false,"message":"no training data","guidance":""}`}, now: time.Now}
	_, err := declined.ReviewTraining(context.Background(), model, nil, "", "")
	require.True(t, IsRejected(err))

	empty := &CLIGenerator{executor: &fakeExecutor{output: `{"accepted":true,"message":"","guidance":"  "}`}, now: time.Now}
	_, err = empty.ReviewTraining(context.Background(), model, nil, "", "")
	require.ErrorContains(t, err, "no guidance")
}

func TestTrainerPromptIncludesNotesPreviousGuidanceAndProfile(t *testing.T) {
	workout := *sampleWorkout()
	workout.Workout.Notes = "Lower back felt tight after deadlifts"
	now := time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)

	prompt := trainerPrompt([]db.WorkoutWithExercises{workout}, "Add a second cardio day", "Bad left knee", now)
	require.Contains(t, prompt, "Lower back felt tight after deadlifts")
	require.Contains(t, prompt, "PREVIOUS_GUIDANCE\nAdd a second cardio day\nEND_PREVIOUS_GUIDANCE")
	require.Contains(t, prompt, "Bad left knee")
	require.Contains(t, prompt, "Today is 2026-11-22")

	first := trainerPrompt(nil, "", "", now)
	require.Contains(t, first, "this is your first review")
}

func TestGenerationPromptIncludesTrainerGuidance(t *testing.T) {
	prompt := generationPrompt(nil, "2026-09-06", "", "- Swap barbell rows for chest-supported rows")
	require.Contains(t, prompt, "TRAINER_GUIDANCE\n- Swap barbell rows for chest-supported rows\nEND_TRAINER_GUIDANCE")

	without := generationPrompt(nil, "2026-09-06", "", "")
	require.NotContains(t, without, "END_TRAINER_GUIDANCE")
}

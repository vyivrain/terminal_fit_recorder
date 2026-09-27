package ai

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOutcomeForContextClassifiesWhyItStopped(t *testing.T) {
	require.Equal(t, OutcomeTimeout, outcomeForContext(context.DeadlineExceeded))
	require.Equal(t, OutcomeCanceled, outcomeForContext(context.Canceled))
	require.Equal(t, OutcomeFailed, outcomeForContext(nil))
}

func TestSummarizeEnvelopeDescribesTheResponse(t *testing.T) {
	accepted := generationEnvelope{Accepted: true, Workout: &workoutPayload{
		Type:      "strength",
		Exercises: []exercisePayload{{Name: "Squat"}, {Name: "Bench"}},
	}}
	require.Equal(t, "accepted a strength workout with 2 exercise(s)", summarizeEnvelope(accepted))

	declined := generationEnvelope{Accepted: false, Message: "  not a workout  "}
	require.Equal(t, "model declined the request: not a workout", summarizeEnvelope(declined))

	empty := generationEnvelope{Accepted: false}
	require.Equal(t, "model declined the request: (no reason given)", summarizeEnvelope(empty))
}

func TestRenderShowsTimeoutInsteadOfCrypticSignal(t *testing.T) {
	diag := CallDiagnostics{
		Provider:    "opencode",
		ModelID:     "opencode-go/glm-5.3",
		DisplayName: "GLM 5.3 · OpenCode",
		Binary:      "opencode",
		BinaryPath:  "/usr/local/bin/opencode",
		Args:        []string{"run", "--model", "opencode-go/glm-5.3"},
		Timeout:     2 * time.Minute,
		Elapsed:     2 * time.Minute,
		OutputBytes: 42,
		Outcome:     OutcomeTimeout,
		Err:         errors.New("opencode timed out after 2m0s (no response — check the model provider)"),
	}

	report := diag.Render()
	require.Contains(t, report, "GLM 5.3 · OpenCode")
	require.Contains(t, report, "/usr/local/bin/opencode")
	require.Contains(t, report, "--model")
	require.Contains(t, report, "Outcome: timed out")
	require.Contains(t, report, "Elapsed: 2m0s")
	require.Contains(t, report, "no response")
}

func TestRenderReportsMissingBinary(t *testing.T) {
	diag := CallDiagnostics{
		Provider: "opencode",
		ModelID:  "opencode-go/glm-5.3",
		Binary:   "opencode",
		Outcome:  OutcomeBinaryMissing,
		Err:      errors.New(`"opencode" is not installed or not on PATH`),
	}

	report := diag.Render()
	require.Contains(t, report, "Outcome: binary missing")
	require.Contains(t, report, "not installed")
	// No binary path resolved, so it falls back to the bare name.
	require.Contains(t, report, "Binary:   opencode")
}

func TestBriefArgFlattensAndTruncates(t *testing.T) {
	require.Equal(t, `line1\nline2`, briefArg("line1\nline2"))

	long := make([]byte, 260)
	for i := range long {
		long[i] = 'x'
	}
	got := briefArg(string(long))
	require.Contains(t, got, "… (+60 chars)")
	require.Less(t, len(got), 260)
}

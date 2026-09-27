package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"terminal_fit_recorder/internal/db"
)

// CallOutcome is the classified result of a diagnostic test call.
type CallOutcome string

const (
	OutcomeOK            CallOutcome = "ok"
	OutcomeTimeout       CallOutcome = "timed out"
	OutcomeCanceled      CallOutcome = "canceled"
	OutcomeFailed        CallOutcome = "failed"
	OutcomeBinaryMissing CallOutcome = "binary missing"
	OutcomeInvalidJSON   CallOutcome = "invalid response"
	OutcomeSetupFailed   CallOutcome = "setup failed"
)

// CallDiagnostics records everything observed during one test call: the model,
// the exact command, its timing, its raw output, and the classified result. The
// production request path deliberately hides these details; this exposes them so
// a stuck or misbehaving call can be understood at a glance.
type CallDiagnostics struct {
	Provider    string
	ModelID     string
	DisplayName string
	Binary      string
	BinaryPath  string
	Args        []string
	Timeout     time.Duration
	Elapsed     time.Duration
	OutputBytes int
	RawOutput   string
	Outcome     CallOutcome
	Summary     string // human summary of a decoded response when Outcome == OutcomeOK
	Err         error
}

// TestCall runs a single real request against model and returns a full account of
// what happened. It never returns an error — every failure mode is captured in the
// returned CallDiagnostics so the caller can render one consistent report. When
// trace is non-nil the model's output is streamed to it live. An empty prompt uses
// a built-in minimal workout request.
//
// Unlike Generate/Refine/ImportNotes this always spawns the real subprocess (it
// bypasses the injected executor) because observing the actual command is the point.
func (g *CLIGenerator) TestCall(ctx context.Context, model db.AIModel, prompt string, trace io.Writer) CallDiagnostics {
	diag := CallDiagnostics{
		Provider:    model.Provider,
		ModelID:     model.ModelID,
		DisplayName: model.DisplayName,
	}
	if deadline, ok := ctx.Deadline(); ok {
		diag.Timeout = time.Until(deadline)
	}

	if strings.TrimSpace(prompt) == "" {
		prompt = fmt.Sprintf(
			"Return an accepted single strength workout for date %s containing exactly one exercise named \"Diagnostic Squat\" with weight 100, reps 5, and sets 5.",
			g.now().Format("2006-01-02"),
		)
	}

	spec, cleanup, err := prepareRequest(model, prompt, workoutResponseSchema)
	if err != nil {
		diag.Outcome = OutcomeSetupFailed
		diag.Err = err
		return diag
	}
	defer cleanup()
	diag.Binary = spec.Name
	diag.Args = spec.Args

	path, err := exec.LookPath(spec.Name)
	if err != nil {
		diag.Outcome = OutcomeBinaryMissing
		diag.Err = fmt.Errorf("%q is not installed or not on PATH: %w", spec.Name, err)
		return diag
	}
	diag.BinaryPath = path

	output, elapsed, runErr := runCommand(ctx, spec, trace)
	diag.Elapsed = elapsed
	diag.RawOutput = output
	diag.OutputBytes = len(output)
	if runErr != nil {
		diag.Outcome = outcomeForContext(ctx.Err())
		diag.Err = runErr
		return diag
	}

	envelope, decodeErr := decodeEnvelope(output)
	if decodeErr != nil {
		diag.Outcome = OutcomeInvalidJSON
		diag.Err = decodeErr
		return diag
	}
	diag.Outcome = OutcomeOK
	diag.Summary = summarizeEnvelope(envelope)
	return diag
}

// outcomeForContext maps why a killed command ended: a done context means we (or a
// deadline) stopped it; otherwise the command itself failed or was killed externally.
func outcomeForContext(ctxErr error) CallOutcome {
	switch {
	case errors.Is(ctxErr, context.DeadlineExceeded):
		return OutcomeTimeout
	case errors.Is(ctxErr, context.Canceled):
		return OutcomeCanceled
	default:
		return OutcomeFailed
	}
}

func summarizeEnvelope(envelope generationEnvelope) string {
	if !envelope.Accepted {
		reason := strings.TrimSpace(envelope.Message)
		if reason == "" {
			reason = "(no reason given)"
		}
		return "model declined the request: " + reason
	}
	if envelope.Workout == nil {
		return "accepted, but returned no workout"
	}
	return fmt.Sprintf("accepted a %s workout with %d exercise(s)", envelope.Workout.Type, len(envelope.Workout.Exercises))
}

// Render formats the diagnostics as a human-readable report. The raw model output
// is intentionally omitted here — callers stream it live during the call.
func (d CallDiagnostics) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Model:    %s\n", orDash(d.DisplayName))
	fmt.Fprintf(&b, "Provider: %s\n", orDash(d.Provider))
	fmt.Fprintf(&b, "Model ID: %s\n", orDash(d.ModelID))
	binary := d.BinaryPath
	if binary == "" {
		binary = d.Binary
	}
	fmt.Fprintf(&b, "Binary:   %s\n", orDash(binary))
	if d.Timeout > 0 {
		fmt.Fprintf(&b, "Timeout:  %s\n", d.Timeout.Round(time.Second))
	}

	if len(d.Args) > 0 {
		b.WriteString("\nCommand:\n")
		fmt.Fprintf(&b, "  %s\n", d.Binary)
		for _, arg := range d.Args {
			fmt.Fprintf(&b, "    %s\n", briefArg(arg))
		}
	}

	b.WriteString("\nResult:\n")
	fmt.Fprintf(&b, "  Outcome: %s\n", d.Outcome)
	if d.Elapsed > 0 {
		fmt.Fprintf(&b, "  Elapsed: %s\n", d.Elapsed.Round(time.Millisecond))
	}
	if d.OutputBytes > 0 {
		fmt.Fprintf(&b, "  Output:  %d bytes\n", d.OutputBytes)
	}
	if d.Summary != "" {
		fmt.Fprintf(&b, "  Detail:  %s\n", d.Summary)
	}
	if d.Err != nil {
		fmt.Fprintf(&b, "  Error:   %s\n", d.Err)
	}
	return b.String()
}

func orDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

// briefArg makes a single argument printable on one line: newlines become literal
// "\n" and very long arguments (the prompt carries the whole JSON schema) are cut
// so the command stays scannable.
func briefArg(arg string) string {
	arg = strings.ReplaceAll(arg, "\n", `\n`)
	const max = 200
	if len(arg) > max {
		return arg[:max] + fmt.Sprintf("… (+%d chars)", len(arg)-max)
	}
	return arg
}

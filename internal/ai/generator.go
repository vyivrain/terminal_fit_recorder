package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"terminal_fit_recorder/internal/db"
)

const (
	openCodeAgentName = "workout-json"
	// OpenCode reserves the last allowed step for its limit handling. A one-response
	// agent therefore needs two steps; steps: 1 returns only a maximum-steps summary.
	openCodeAgent = `---
description: Convert workout requests into strict JSON without using tools
mode: primary
temperature: 0
steps: 2
tools:
  bash: false
  read: false
  edit: false
  write: false
  glob: false
  grep: false
  webfetch: false
  websearch: false
  task: false
  todowrite: false
  lsp: false
  skill: false
---
You are a single-response structured workout processor.
Never call tools, inspect files, or perform general-purpose planning.
Follow the user prompt's workout constraints and return only its requested JSON object.`
)

type Generator interface {
	Generate(ctx context.Context, model db.AIModel, history []db.WorkoutWithExercises) (*db.WorkoutWithExercises, error)
	Refine(ctx context.Context, model db.AIModel, workout *db.WorkoutWithExercises, instruction string) (*db.WorkoutWithExercises, error)
	ImportNotes(ctx context.Context, model db.AIModel, notes string, history []db.WorkoutWithExercises) ([]*db.WorkoutWithExercises, error)
}

type RejectedError struct {
	Reason string
}

func (e *RejectedError) Error() string {
	if e.Reason == "" {
		return "the AI declined the workout request"
	}
	return e.Reason
}

func IsRejected(err error) bool {
	var rejected *RejectedError
	return errors.As(err, &rejected)
}

type commandExecutor interface {
	Run(ctx context.Context, spec commandSpec) (string, error)
}

type osCommandExecutor struct{}

func (osCommandExecutor) Run(ctx context.Context, spec commandSpec) (string, error) {
	output, _, err := runCommand(ctx, spec, nil)
	if err != nil {
		return "", err
	}
	return output, nil
}

// runCommand executes spec and always reports the full combined output and how
// long it ran, whether or not it succeeded. When trace is non-nil the child's
// stdout and stderr are streamed to it live (as well as captured), so a caller
// can watch a slow or hung command in real time instead of waiting for exit —
// the production path passes nil and only wants the final output.
func runCommand(ctx context.Context, spec commandSpec, trace io.Writer) (string, time.Duration, error) {
	command := exec.CommandContext(ctx, spec.Name, spec.Args...)
	command.Dir = spec.Directory
	command.Stdin = strings.NewReader(spec.Stdin)
	command.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0")

	var captured bytes.Buffer
	var sink io.Writer = &captured
	if trace != nil {
		sink = io.MultiWriter(&captured, trace)
	}
	command.Stdout = sink
	command.Stderr = sink

	started := time.Now()
	err := command.Run()
	elapsed := time.Since(started)
	if err != nil {
		return captured.String(), elapsed, classifyRunError(spec.Name, ctx.Err(), elapsed, captured.Bytes(), err)
	}
	return captured.String(), elapsed, nil
}

// classifyRunError turns a subprocess failure into a message that names the real
// cause. When the context ended first, exec.CommandContext kills the child with
// SIGKILL, so the raw error is an indistinguishable "signal: killed" for both a
// timeout and an external kill. Inspecting ctx.Err() recovers the distinction:
// a deadline means the command hung (e.g. the model provider never responded),
// which is otherwise invisible to the user.
func classifyRunError(name string, ctxErr error, elapsed time.Duration, output []byte, err error) error {
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out after %s (no response — check the model provider)", name, elapsed.Round(time.Second))
	}
	if errors.Is(ctxErr, context.Canceled) {
		return fmt.Errorf("%s was canceled", name)
	}

	detail := strings.TrimSpace(stripANSI(string(output)))
	if len(detail) > 1200 {
		detail = detail[:1200] + "…"
	}
	if detail == "" {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return fmt.Errorf("%s failed: %w\n%s", name, err, detail)
}

type CLIGenerator struct {
	executor commandExecutor
	now      func() time.Time
}

func NewCLIGenerator() *CLIGenerator {
	return &CLIGenerator{
		executor: osCommandExecutor{},
		now:      time.Now,
	}
}

func (g *CLIGenerator) Generate(ctx context.Context, model db.AIModel, history []db.WorkoutWithExercises) (*db.WorkoutWithExercises, error) {
	expectedDate := g.now().Format("2006-01-02")
	output, err := g.request(ctx, model, generationPrompt(history, expectedDate), workoutResponseSchema)
	if err != nil {
		return nil, err
	}
	envelope, err := decodeEnvelope(output)
	if err != nil {
		return nil, fmt.Errorf("%s returned an invalid workout response: %w", model.DisplayName, err)
	}
	return validateEnvelope(envelope, expectedDate, "")
}

func (g *CLIGenerator) Refine(ctx context.Context, model db.AIModel, workout *db.WorkoutWithExercises, instruction string) (*db.WorkoutWithExercises, error) {
	if workout == nil {
		return nil, fmt.Errorf("workout is required")
	}
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return nil, fmt.Errorf("modification request cannot be empty")
	}

	expectedDate := workout.Workout.WorkoutDate.Format("2006-01-02")
	output, err := g.request(ctx, model, refinementPrompt(workout, instruction), workoutResponseSchema)
	if err != nil {
		return nil, err
	}
	envelope, err := decodeEnvelope(output)
	if err != nil {
		return nil, fmt.Errorf("%s returned an invalid workout response: %w", model.DisplayName, err)
	}
	return validateEnvelope(envelope, expectedDate, workout.Workout.WorkoutType)
}

func (g *CLIGenerator) ImportNotes(ctx context.Context, model db.AIModel, notes string, history []db.WorkoutWithExercises) ([]*db.WorkoutWithExercises, error) {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return nil, fmt.Errorf("workout notes cannot be empty")
	}

	schedule := nextWorkoutSchedule(g.now())
	output, err := g.request(ctx, model, importPrompt(history, notes, schedule), importResponseSchema)
	if err != nil {
		return nil, err
	}
	envelope, err := decodeImportEnvelope(output)
	if err != nil {
		return nil, fmt.Errorf("%s returned an invalid workout import response: %w", model.DisplayName, err)
	}
	return validateImportEnvelope(envelope, schedule)
}

func (g *CLIGenerator) request(ctx context.Context, model db.AIModel, prompt, responseSchema string) (string, error) {
	spec, cleanup, err := prepareRequest(model, prompt, responseSchema)
	if err != nil {
		return "", err
	}
	defer cleanup()

	output, err := g.executor.Run(ctx, spec)
	if err != nil {
		return "", err
	}
	return output, nil
}

// prepareRequest builds the isolated workspace (schema file, and for OpenCode the
// tool-less agent) and the exact command to run. The returned cleanup removes the
// workspace and must be called once the command has finished. It is shared by the
// production request path and the diagnostic test call so both invoke a model the
// same way.
func prepareRequest(model db.AIModel, prompt, responseSchema string) (commandSpec, func(), error) {
	workingDirectory, err := os.MkdirTemp("", "terminal-fit-ai-")
	if err != nil {
		return commandSpec{}, nil, fmt.Errorf("could not create isolated AI workspace: %w", err)
	}
	cleanup := func() { os.RemoveAll(workingDirectory) }

	schemaPath := filepath.Join(workingDirectory, "workout-response.schema.json")
	if err := os.WriteFile(schemaPath, []byte(responseSchema), 0600); err != nil {
		cleanup()
		return commandSpec{}, nil, fmt.Errorf("could not prepare workout schema: %w", err)
	}
	if model.Provider == db.ModelProviderOpenCode {
		if err := prepareOpenCodeAgent(workingDirectory); err != nil {
			cleanup()
			return commandSpec{}, nil, err
		}
	}

	spec, err := modelCommand(model, workingDirectory, schemaPath, prompt, responseSchema)
	if err != nil {
		cleanup()
		return commandSpec{}, nil, err
	}
	return spec, cleanup, nil
}

func prepareOpenCodeAgent(directory string) error {
	agentDirectory := filepath.Join(directory, ".opencode", "agents")
	if err := os.MkdirAll(agentDirectory, 0700); err != nil {
		return fmt.Errorf("could not prepare OpenCode workout agent: %w", err)
	}
	agentPath := filepath.Join(agentDirectory, openCodeAgentName+".md")
	if err := os.WriteFile(agentPath, []byte(openCodeAgent), 0600); err != nil {
		return fmt.Errorf("could not prepare OpenCode workout agent: %w", err)
	}
	return nil
}

type commandSpec struct {
	Name      string
	Args      []string
	Stdin     string
	Directory string
}

func modelCommand(model db.AIModel, directory, schemaPath, prompt, responseSchema string) (commandSpec, error) {
	switch model.Provider {
	case db.ModelProviderOpenCode:
		prompt = strings.TrimSpace(prompt) + "\n\nOUTPUT_JSON_SCHEMA\n" + responseSchema + "\nEND_OUTPUT_JSON_SCHEMA"
		return commandSpec{
			Name: "opencode",
			Args: []string{
				"run", "--pure", "--agent", openCodeAgentName,
				"--model", model.ModelID,
				"--title", "Terminal Fit Recorder",
				"--dir", directory,
				prompt,
			},
			Directory: directory,
		}, nil
	case db.ModelProviderCodex:
		return commandSpec{
			Name: "codex",
			Args: []string{
				"exec", "--model", model.ModelID,
				"--sandbox", "read-only",
				"-c", `approval_policy="never"`,
				"--ephemeral", "--ignore-rules", "--skip-git-repo-check",
				"--output-schema", schemaPath,
				"--color", "never",
				"-C", directory, "-",
			},
			Stdin:     prompt,
			Directory: directory,
		}, nil
	case db.ModelProviderClaude:
		return commandSpec{
			Name: "claude",
			Args: []string{
				"--safe-mode", "--print",
				"--model", model.ModelID,
				"--no-session-persistence",
				"--permission-mode", "dontAsk",
				"--tools", "",
				"--json-schema", responseSchema,
				"--output-format", "json",
			},
			Stdin:     prompt,
			Directory: directory,
		}, nil
	default:
		return commandSpec{}, fmt.Errorf("unsupported AI provider %q", model.Provider)
	}
}

type generationEnvelope struct {
	Accepted bool            `json:"accepted"`
	Message  string          `json:"message"`
	Workout  *workoutPayload `json:"workout"`
}

type importEnvelope struct {
	Accepted bool             `json:"accepted"`
	Message  string           `json:"message"`
	Workouts []workoutPayload `json:"workouts"`
}

type workoutPayload struct {
	Date      string            `json:"date"`
	Type      string            `json:"type"`
	Exercises []exercisePayload `json:"exercises"`
}

type exercisePayload struct {
	Name     string  `json:"name"`
	Weight   int     `json:"weight"`
	Reps     int     `json:"reps"`
	Sets     int     `json:"sets"`
	Duration float64 `json:"duration"`
	Distance int     `json:"distance"`
}

func validateEnvelope(envelope generationEnvelope, expectedDate, expectedType string) (*db.WorkoutWithExercises, error) {
	if !envelope.Accepted {
		return nil, &RejectedError{Reason: strings.TrimSpace(envelope.Message)}
	}
	if envelope.Workout == nil {
		return nil, fmt.Errorf("accepted response did not contain a workout")
	}

	return validateWorkoutPayload(envelope.Workout, expectedDate, expectedType)
}

func validateImportEnvelope(envelope importEnvelope, schedule []scheduledWorkoutDate) ([]*db.WorkoutWithExercises, error) {
	if !envelope.Accepted {
		return nil, &RejectedError{Reason: strings.TrimSpace(envelope.Message)}
	}
	if len(envelope.Workouts) == 0 || len(envelope.Workouts) > 7 {
		return nil, fmt.Errorf("workout import must contain between 1 and 7 workouts")
	}

	allowedDates := make(map[string]bool, len(schedule))
	for _, scheduled := range schedule {
		allowedDates[scheduled.Date] = true
	}
	seenDates := make(map[string]bool, len(envelope.Workouts))
	workouts := make([]*db.WorkoutWithExercises, 0, len(envelope.Workouts))
	for index := range envelope.Workouts {
		payload := &envelope.Workouts[index]
		if !allowedDates[payload.Date] {
			return nil, fmt.Errorf("imported workout %d used unexpected date %q", index+1, payload.Date)
		}
		if seenDates[payload.Date] {
			return nil, fmt.Errorf("import returned more than one workout for %s", payload.Date)
		}
		seenDates[payload.Date] = true

		workout, err := validateWorkoutPayload(payload, payload.Date, "")
		if err != nil {
			return nil, fmt.Errorf("imported workout %d: %w", index+1, err)
		}
		workouts = append(workouts, workout)
	}
	return workouts, nil
}

func validateWorkoutPayload(payload *workoutPayload, expectedDate, expectedType string) (*db.WorkoutWithExercises, error) {
	if payload.Date != expectedDate {
		return nil, fmt.Errorf("model changed workout date: expected %s, got %s", expectedDate, payload.Date)
	}
	workoutDate, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid workout date: %w", err)
	}

	payload.Type = strings.ToLower(strings.TrimSpace(payload.Type))
	if payload.Type != "strength" && payload.Type != "cardio" {
		return nil, fmt.Errorf("unsupported workout type %q", payload.Type)
	}
	if expectedType != "" && payload.Type != expectedType {
		return nil, fmt.Errorf("modification changed workout type from %s to %s", expectedType, payload.Type)
	}
	if len(payload.Exercises) == 0 || len(payload.Exercises) > 20 {
		return nil, fmt.Errorf("workout must contain between 1 and 20 exercises")
	}

	exercises := make([]db.Exercise, 0, len(payload.Exercises))
	for index, exercise := range payload.Exercises {
		exercise.Name = strings.TrimSpace(exercise.Name)
		if exercise.Name == "" || len([]rune(exercise.Name)) > 100 {
			return nil, fmt.Errorf("exercise %d has an invalid name", index+1)
		}
		if exercise.Weight < 0 || exercise.Weight > 1000 {
			return nil, fmt.Errorf("exercise %d has invalid weight", index+1)
		}
		if exercise.Reps < 0 || exercise.Reps > 1000 {
			return nil, fmt.Errorf("exercise %d has invalid repetitions", index+1)
		}
		if exercise.Sets < 0 || exercise.Sets > 100 {
			return nil, fmt.Errorf("exercise %d has invalid sets", index+1)
		}
		if exercise.Duration < 0 || exercise.Duration > 1440 {
			return nil, fmt.Errorf("exercise %d has invalid duration", index+1)
		}
		if exercise.Distance < 0 || exercise.Distance > 1_000_000 {
			return nil, fmt.Errorf("exercise %d has invalid distance", index+1)
		}

		exercises = append(exercises, db.Exercise{
			Name:        exercise.Name,
			Weight:      exercise.Weight,
			Repetitions: exercise.Reps,
			Sets:        exercise.Sets,
			Duration:    exercise.Duration,
			Distance:    exercise.Distance,
		})
	}

	return &db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: payload.Type,
			WorkoutDate: workoutDate,
			Status:      "planned",
		},
		Exercises: exercises,
	}, nil
}

var ansiPattern = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\))`)

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

func decodeEnvelope(output string) (generationEnvelope, error) {
	return decodeStructuredResponse[generationEnvelope](output, "workout")
}

func decodeImportEnvelope(output string) (importEnvelope, error) {
	return decodeStructuredResponse[importEnvelope](output, "workouts")
}

func decodeStructuredResponse[T any](output, payloadKey string) (T, error) {
	var zero T
	cleaned := strings.TrimSpace(stripANSI(output))
	if cleaned == "" {
		return zero, fmt.Errorf("empty output")
	}

	lines := strings.Split(cleaned, "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		var value any
		if json.Unmarshal([]byte(lines[index]), &value) == nil {
			if envelope, ok := structuredResponseFromValue[T](value, payloadKey); ok {
				return envelope, nil
			}
		}
	}

	for offset := 0; offset < len(cleaned); {
		relative := strings.Index(cleaned[offset:], "{")
		if relative < 0 {
			break
		}
		start := offset + relative
		decoder := json.NewDecoder(strings.NewReader(cleaned[start:]))
		var value any
		if decoder.Decode(&value) == nil {
			if envelope, ok := structuredResponseFromValue[T](value, payloadKey); ok {
				return envelope, nil
			}
		}
		offset = start + 1
	}

	return zero, fmt.Errorf("no schema-valid JSON object found")
}

func structuredResponseFromValue[T any](value any, payloadKey string) (T, bool) {
	var zero T
	switch typed := value.(type) {
	case map[string]any:
		_, hasAccepted := typed["accepted"]
		_, hasPayload := typed[payloadKey]
		if hasAccepted && hasPayload {
			encoded, err := json.Marshal(typed)
			if err == nil {
				var envelope T
				if json.Unmarshal(encoded, &envelope) == nil {
					return envelope, true
				}
			}
		}
		for _, key := range []string{"structured_output", "result", "output_text", "content", "text", "message"} {
			if nested, exists := typed[key]; exists {
				if envelope, ok := structuredResponseFromValue[T](nested, payloadKey); ok {
					return envelope, true
				}
			}
		}
	case []any:
		for index := len(typed) - 1; index >= 0; index-- {
			if envelope, ok := structuredResponseFromValue[T](typed[index], payloadKey); ok {
				return envelope, true
			}
		}
	case string:
		if strings.Contains(typed, "{") {
			if envelope, err := decodeStructuredResponse[T](typed, payloadKey); err == nil {
				return envelope, true
			}
		}
	}

	return zero, false
}

const workoutObjectSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "date": {"type": "string"},
    "type": {"type": "string", "enum": ["strength", "cardio"]},
    "exercises": {
      "type": "array",
      "minItems": 1,
      "maxItems": 20,
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "name": {"type": "string"},
          "weight": {"type": "integer", "minimum": 0},
          "reps": {"type": "integer", "minimum": 0},
          "sets": {"type": "integer", "minimum": 0},
          "duration": {"type": "number", "minimum": 0},
          "distance": {"type": "integer", "minimum": 0}
        },
        "required": ["name", "weight", "reps", "sets", "duration", "distance"]
      }
    }
  },
  "required": ["date", "type", "exercises"]
}`

const workoutResponseSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "accepted": {"type": "boolean"},
    "message": {"type": "string"},
    "workout": {
      "anyOf": [
        ` + workoutObjectSchema + `,
        {"type": "null"}
      ]
    }
  },
  "required": ["accepted", "message", "workout"]
}`

const importResponseSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "accepted": {"type": "boolean"},
    "message": {"type": "string"},
    "workouts": {
      "type": "array",
      "minItems": 0,
      "maxItems": 7,
      "items": ` + workoutObjectSchema + `
    }
  },
  "required": ["accepted", "message", "workouts"]
}`

// Package mcpserver exposes the workout database to AI agents (Claude, Hermes,
// and other MCP clients) over the Model Context Protocol. Every tool operates
// on the active profile, exactly like the TUI and CLI commands.
package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"terminal_fit_recorder/internal/db"
)

const dateLayout = "2006-01-02"

// Run serves MCP over stdin/stdout until the client disconnects.
func Run(ctx context.Context, database *db.DB) error {
	return New(database).Run(ctx, &mcp.StdioTransport{})
}

// New builds a server with all workout tools registered.
func New(database *db.DB) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "terminal_fit_recorder", Version: "1.0.0"}, &mcp.ServerOptions{
		Instructions: "Workout log for the active profile. Dates are YYYY-MM-DD; at most one workout per day. " +
			"Call list_exercise_names before logging so exercise names stay consistent with history. " +
			"To plan a workout, log it with status \"planned\"; mark it done later with set_workout_status.",
	})
	t := &tools{db: database}
	destructive := true

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_profiles",
		Description: "List profiles with their coaching descriptions. The active profile scopes every other tool.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, t.listProfiles)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "use_profile",
		Description: "Switch the active profile by name. This also changes the profile the TUI opens with.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, t.useProfile)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_workouts",
		Description: "List workouts, newest first, optionally filtered by status and date range.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, t.listWorkouts)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_workout",
		Description: "Get the workout on a date, including exercises and notes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, t.getWorkout)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_exercise_names",
		Description: "List distinct exercise names already used, to keep naming consistent.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, t.listExerciseNames)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "log_workout",
		Description: "Save a completed or planned workout. Fails if a workout already exists on that date.",
	}, t.logWorkout)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_workout_status",
		Description: "Mark the workout on a date as planned or completed.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, t.setWorkoutStatus)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_workout_notes",
		Description: "Replace the notes on the workout for a date (e.g. how it went, what changed and why).",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, t.setWorkoutNotes)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_workout",
		Description: "Permanently delete the workout on a date and its exercises.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: true},
	}, t.deleteWorkout)

	return server
}

type tools struct {
	db *db.DB
}

type exercise struct {
	Name        string  `json:"name" jsonschema:"exercise name in English, e.g. Bench Press"`
	WeightKg    int     `json:"weight_kg,omitempty" jsonschema:"weight in kg; omit for bodyweight"`
	Reps        int     `json:"reps,omitempty" jsonschema:"repetitions per set"`
	Sets        int     `json:"sets,omitempty" jsonschema:"number of sets"`
	DurationMin float64 `json:"duration_min,omitempty" jsonschema:"duration in minutes, for cardio or timed holds"`
	DistanceM   int     `json:"distance_m,omitempty" jsonschema:"distance in metres, for runs, walks, rows"`
	YoutubeURL  string  `json:"youtube_url,omitempty" jsonschema:"technique video link"`
}

type workout struct {
	Date      string     `json:"date"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Notes     string     `json:"notes,omitempty"`
	Exercises []exercise `json:"exercises"`
}

type profile struct {
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	Description string `json:"description,omitempty"`
}

type dateInput struct {
	Date string `json:"date" jsonschema:"workout date, YYYY-MM-DD"`
}

type profilesOutput struct {
	Profiles []profile `json:"profiles"`
}

func (t *tools) listProfiles(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, profilesOutput, error) {
	profiles, err := t.db.GetProfiles()
	if err != nil {
		return nil, profilesOutput{}, err
	}
	out := profilesOutput{Profiles: make([]profile, 0, len(profiles))}
	for _, p := range profiles {
		out.Profiles = append(out.Profiles, profile{Name: p.Name, Active: p.IsActive, Description: p.Description})
	}
	return nil, out, nil
}

type useProfileInput struct {
	Name string `json:"name" jsonschema:"existing profile name (case-insensitive)"`
}

func (t *tools) useProfile(_ context.Context, _ *mcp.CallToolRequest, in useProfileInput) (*mcp.CallToolResult, profile, error) {
	p, err := t.db.UseProfile(in.Name)
	if err != nil {
		return nil, profile{}, err
	}
	return nil, profile{Name: p.Name, Active: true, Description: p.Description}, nil
}

type listWorkoutsInput struct {
	Status string `json:"status,omitempty" jsonschema:"planned or completed; omit for both"`
	From   string `json:"from,omitempty" jsonschema:"earliest date, YYYY-MM-DD, inclusive"`
	To     string `json:"to,omitempty" jsonschema:"latest date, YYYY-MM-DD, inclusive"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum workouts to return; omit for all"`
}

type workoutsOutput struct {
	Workouts []workout `json:"workouts"`
}

func (t *tools) listWorkouts(_ context.Context, _ *mcp.CallToolRequest, in listWorkoutsInput) (*mcp.CallToolResult, workoutsOutput, error) {
	if in.Status != "" {
		if err := validateStatus(in.Status); err != nil {
			return nil, workoutsOutput{}, err
		}
	}
	var from, to string
	if in.From != "" {
		date, err := parseDate(in.From)
		if err != nil {
			return nil, workoutsOutput{}, err
		}
		from = date.Format(dateLayout)
	}
	if in.To != "" {
		date, err := parseDate(in.To)
		if err != nil {
			return nil, workoutsOutput{}, err
		}
		to = date.Format(dateLayout)
	}
	if in.Limit < 0 {
		return nil, workoutsOutput{}, fmt.Errorf("limit cannot be negative")
	}

	all, err := t.db.GetAllWorkouts()
	if err != nil {
		return nil, workoutsOutput{}, err
	}
	out := workoutsOutput{Workouts: make([]workout, 0)}
	for _, w := range all {
		day := w.Workout.WorkoutDate.Format(dateLayout)
		if (in.Status != "" && w.Workout.Status != in.Status) || (from != "" && day < from) || (to != "" && day > to) {
			continue
		}
		out.Workouts = append(out.Workouts, toWorkout(w))
		if in.Limit > 0 && len(out.Workouts) == in.Limit {
			break
		}
	}
	return nil, out, nil
}

func (t *tools) getWorkout(_ context.Context, _ *mcp.CallToolRequest, in dateInput) (*mcp.CallToolResult, workout, error) {
	w, err := t.findWorkout(in.Date)
	if err != nil {
		return nil, workout{}, err
	}
	return nil, toWorkout(*w), nil
}

type exerciseNamesOutput struct {
	Names []string `json:"names"`
}

func (t *tools) listExerciseNames(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, exerciseNamesOutput, error) {
	names, err := t.db.GetDistinctExerciseNames()
	if err != nil {
		return nil, exerciseNamesOutput{}, err
	}
	if names == nil {
		names = []string{}
	}
	return nil, exerciseNamesOutput{Names: names}, nil
}

type logWorkoutInput struct {
	Date      string     `json:"date" jsonschema:"workout date, YYYY-MM-DD"`
	Type      string     `json:"type" jsonschema:"strength or cardio"`
	Status    string     `json:"status,omitempty" jsonschema:"completed (default) or planned"`
	Notes     string     `json:"notes,omitempty" jsonschema:"free-form commentary on this workout"`
	Exercises []exercise `json:"exercises" jsonschema:"at least one exercise"`
}

func (t *tools) logWorkout(_ context.Context, _ *mcp.CallToolRequest, in logWorkoutInput) (*mcp.CallToolResult, workout, error) {
	date, err := parseDate(in.Date)
	if err != nil {
		return nil, workout{}, err
	}
	if in.Type != "strength" && in.Type != "cardio" {
		return nil, workout{}, fmt.Errorf("type must be 'strength' or 'cardio', got: %q", in.Type)
	}
	status := in.Status
	if status == "" {
		status = "completed"
	}
	if err := validateStatus(status); err != nil {
		return nil, workout{}, err
	}
	if len(in.Exercises) == 0 {
		return nil, workout{}, fmt.Errorf("at least one exercise is required")
	}

	record := &db.WorkoutWithExercises{
		Workout: db.Workout{WorkoutType: in.Type, WorkoutDate: date, Status: status, Notes: strings.TrimSpace(in.Notes)},
	}
	for i, e := range in.Exercises {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			return nil, workout{}, fmt.Errorf("exercise %d has no name", i+1)
		}
		if e.WeightKg < 0 || e.Reps < 0 || e.Sets < 0 || e.DurationMin < 0 || e.DistanceM < 0 {
			return nil, workout{}, fmt.Errorf("exercise %q has a negative value", name)
		}
		record.Exercises = append(record.Exercises, db.Exercise{
			Name: name, Weight: e.WeightKg, Repetitions: e.Reps, Sets: e.Sets,
			Duration: e.DurationMin, Distance: e.DistanceM, YoutubeURL: strings.TrimSpace(e.YoutubeURL),
		})
	}

	if err := t.db.SaveWorkoutsWithExercises([]*db.WorkoutWithExercises{record}, status); err != nil {
		return nil, workout{}, err
	}
	return nil, toWorkout(*record), nil
}

type setStatusInput struct {
	Date   string `json:"date" jsonschema:"workout date, YYYY-MM-DD"`
	Status string `json:"status" jsonschema:"planned or completed"`
}

func (t *tools) setWorkoutStatus(_ context.Context, _ *mcp.CallToolRequest, in setStatusInput) (*mcp.CallToolResult, workout, error) {
	w, err := t.findWorkout(in.Date)
	if err != nil {
		return nil, workout{}, err
	}
	if err := t.db.SetWorkoutStatus(w.Workout.ID, in.Status); err != nil {
		return nil, workout{}, err
	}
	w.Workout.Status = in.Status
	return nil, toWorkout(*w), nil
}

type setNotesInput struct {
	Date  string `json:"date" jsonschema:"workout date, YYYY-MM-DD"`
	Notes string `json:"notes" jsonschema:"new notes; empty string clears them"`
}

func (t *tools) setWorkoutNotes(_ context.Context, _ *mcp.CallToolRequest, in setNotesInput) (*mcp.CallToolResult, workout, error) {
	w, err := t.findWorkout(in.Date)
	if err != nil {
		return nil, workout{}, err
	}
	notes := strings.TrimSpace(in.Notes)
	if err := t.db.SetWorkoutNotes(w.Workout.ID, notes); err != nil {
		return nil, workout{}, err
	}
	w.Workout.Notes = notes
	return nil, toWorkout(*w), nil
}

type deleteOutput struct {
	Deleted string `json:"deleted"`
}

func (t *tools) deleteWorkout(_ context.Context, _ *mcp.CallToolRequest, in dateInput) (*mcp.CallToolResult, deleteOutput, error) {
	w, err := t.findWorkout(in.Date)
	if err != nil {
		return nil, deleteOutput{}, err
	}
	if err := t.db.DeleteWorkoutByDate(w.Workout.WorkoutDate); err != nil {
		return nil, deleteOutput{}, err
	}
	return nil, deleteOutput{Deleted: w.Workout.WorkoutDate.Format(dateLayout)}, nil
}

func (t *tools) findWorkout(dateStr string) (*db.WorkoutWithExercises, error) {
	date, err := parseDate(dateStr)
	if err != nil {
		return nil, err
	}
	w, err := t.db.GetWorkoutByDate(date)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, fmt.Errorf("no workout on %s", date.Format(dateLayout))
	}
	return w, nil
}

func parseDate(value string) (time.Time, error) {
	date, err := time.ParseInLocation(dateLayout, strings.TrimSpace(value), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("date must be YYYY-MM-DD, got: %q", value)
	}
	return date, nil
}

func validateStatus(status string) error {
	if status != "planned" && status != "completed" {
		return fmt.Errorf("status must be 'planned' or 'completed', got: %q", status)
	}
	return nil
}

func toWorkout(w db.WorkoutWithExercises) workout {
	out := workout{
		Date:      w.Workout.WorkoutDate.Format(dateLayout),
		Type:      w.Workout.WorkoutType,
		Status:    w.Workout.Status,
		Notes:     w.Workout.Notes,
		Exercises: make([]exercise, 0, len(w.Exercises)),
	}
	for _, e := range w.Exercises {
		out.Exercises = append(out.Exercises, exercise{
			Name: e.Name, WeightKg: e.Weight, Reps: e.Repetitions, Sets: e.Sets,
			DurationMin: e.Duration, DistanceM: e.Distance, YoutubeURL: e.YoutubeURL,
		})
	}
	return out
}

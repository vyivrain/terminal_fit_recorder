package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

func connect(t *testing.T) (*mcp.ClientSession, *db.DB) {
	t.Helper()
	database, err := db.New(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(database).Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { session.Close() })
	return session, database
}

// call invokes a tool and decodes its structured output into out (if non-nil).
func call(t *testing.T, session *mcp.ClientSession, name string, args map[string]any, out any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	if out != nil && !result.IsError {
		raw, err := json.Marshal(result.StructuredContent)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, out))
	}
	return result
}

func errorText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if text, ok := result.Content[0].(*mcp.TextContent); ok {
		return text.Text
	}
	return ""
}

func TestWorkoutLifecycle(t *testing.T) {
	session, _ := connect(t)

	var logged workout
	result := call(t, session, "log_workout", map[string]any{
		"date":   "2026-10-14",
		"type":   "strength",
		"status": "planned",
		"exercises": []map[string]any{
			{"name": " Bench Press ", "weight_kg": 80, "reps": 10, "sets": 3},
			{"name": "Plank", "duration_min": 1.5},
		},
	}, &logged)
	require.False(t, result.IsError, errorText(result))
	require.Equal(t, "planned", logged.Status)
	require.Equal(t, "Bench Press", logged.Exercises[0].Name)

	call(t, session, "log_workout", map[string]any{
		"date": "2026-10-10", "type": "cardio",
		"exercises": []map[string]any{{"name": "Run", "duration_min": 30, "distance_m": 5000}},
	}, nil)

	var planned workoutsOutput
	call(t, session, "list_workouts", map[string]any{"status": "planned"}, &planned)
	require.Len(t, planned.Workouts, 1)
	require.Equal(t, "2026-10-14", planned.Workouts[0].Date)

	var ranged workoutsOutput
	call(t, session, "list_workouts", map[string]any{"from": "2026-10-01", "to": "2026-10-12"}, &ranged)
	require.Len(t, ranged.Workouts, 1)
	require.Equal(t, "completed", ranged.Workouts[0].Status)

	call(t, session, "set_workout_status", map[string]any{"date": "2026-10-14", "status": "completed"}, nil)
	call(t, session, "set_workout_notes", map[string]any{"date": "2026-10-14", "notes": "Felt strong"}, nil)

	var fetched workout
	call(t, session, "get_workout", map[string]any{"date": "2026-10-14"}, &fetched)
	require.Equal(t, "completed", fetched.Status)
	require.Equal(t, "Felt strong", fetched.Notes)
	require.Equal(t, 1.5, fetched.Exercises[1].DurationMin)

	var names exerciseNamesOutput
	call(t, session, "list_exercise_names", nil, &names)
	require.ElementsMatch(t, []string{"Bench Press", "Plank", "Run"}, names.Names)

	call(t, session, "delete_workout", map[string]any{"date": "2026-10-14"}, nil)
	result = call(t, session, "get_workout", map[string]any{"date": "2026-10-14"}, nil)
	require.True(t, result.IsError)
	require.Contains(t, errorText(result), "no workout on 2026-10-14")
}

func TestLogWorkoutRejectsInvalidInput(t *testing.T) {
	session, _ := connect(t)
	valid := []map[string]any{{"name": "Squat", "reps": 5, "sets": 5}}

	cases := map[string]struct {
		args map[string]any
		want string
	}{
		"bad date":       {map[string]any{"date": "14.10.2026", "type": "strength", "exercises": valid}, "YYYY-MM-DD"},
		"bad type":       {map[string]any{"date": "2026-10-14", "type": "yoga", "exercises": valid}, "type must be"},
		"bad status":     {map[string]any{"date": "2026-10-14", "type": "strength", "status": "done", "exercises": valid}, "status must be"},
		"no exercises":   {map[string]any{"date": "2026-10-14", "type": "strength", "exercises": []map[string]any{}}, "at least one exercise"},
		"blank name":     {map[string]any{"date": "2026-10-14", "type": "strength", "exercises": []map[string]any{{"name": " "}}}, "has no name"},
		"negative value": {map[string]any{"date": "2026-10-14", "type": "strength", "exercises": []map[string]any{{"name": "Squat", "sets": -1}}}, "negative"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := call(t, session, "log_workout", tc.args, nil)
			require.True(t, result.IsError)
			require.Contains(t, errorText(result), tc.want)
		})
	}

	ok := map[string]any{"date": "2026-10-14", "type": "strength", "exercises": valid}
	require.False(t, call(t, session, "log_workout", ok, nil).IsError)
	duplicate := call(t, session, "log_workout", ok, nil)
	require.True(t, duplicate.IsError)
	require.Contains(t, errorText(duplicate), "already exists")
}

func TestProfilesScopeWorkouts(t *testing.T) {
	session, database := connect(t)
	call(t, session, "log_workout", map[string]any{
		"date": "2026-10-14", "type": "strength",
		"exercises": []map[string]any{{"name": "Squat"}},
	}, nil)
	_, err := database.CreateProfile("partner") // also makes it active
	require.NoError(t, err)

	var workouts workoutsOutput
	call(t, session, "list_workouts", nil, &workouts)
	require.Empty(t, workouts.Workouts)

	var active profile
	call(t, session, "use_profile", map[string]any{"name": "MINE"}, &active)
	require.Equal(t, db.DefaultProfileName, active.Name)
	call(t, session, "list_workouts", nil, &workouts)
	require.Len(t, workouts.Workouts, 1)

	var profiles profilesOutput
	call(t, session, "list_profiles", nil, &profiles)
	require.Len(t, profiles.Profiles, 2)

	result := call(t, session, "use_profile", map[string]any{"name": "nobody"}, nil)
	require.True(t, result.IsError)
}

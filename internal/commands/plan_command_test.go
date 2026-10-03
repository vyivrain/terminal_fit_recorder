package commands

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

type fakePlanVideoFinder struct {
	urls  map[string]string
	calls []string
}

func (finder *fakePlanVideoFinder) Find(_ context.Context, exerciseName string) (string, error) {
	finder.calls = append(finder.calls, exerciseName)
	return finder.urls[exerciseName], nil
}

func TestPlanCommandDefaultsToPreview(t *testing.T) {
	cmd := NewPlanCommand()
	require.NoError(t, cmd.Validate())
	assert.False(t, cmd.save)
	assert.Equal(t, "plan", cmd.Name())
	assert.Contains(t, cmd.HelpManual(), "--save")
}

func TestPlanCommandAcceptsExplicitSave(t *testing.T) {
	cmd := NewPlanCommand("--save")
	require.NoError(t, cmd.Validate())
	assert.True(t, cmd.save)
}

func TestPlanCommandRejectsUnknownArguments(t *testing.T) {
	cmd := NewPlanCommand("--force")
	err := cmd.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exercise plan [--save]")
}

func TestParserRoutesPlanCommand(t *testing.T) {
	parsed, err := ParseArgs([]string{"terminal_fit_recorder", "exercise", "plan", "--save"})
	require.NoError(t, err)
	cmd, ok := parsed.(*PlanCommand)
	require.True(t, ok)
	assert.True(t, cmd.save)
}

func TestPopulatePlanVideosReusesStoredURLAndLooksUpMissingExercise(t *testing.T) {
	workout := &db.WorkoutWithExercises{
		Workout: db.Workout{WorkoutType: "strength", WorkoutDate: time.Now(), Status: "planned"},
		Exercises: []db.Exercise{
			{Name: "  Bench   Press "},
			{Name: "Bulgarian Split Squat"},
			{Name: "Back Pull"},
		},
	}
	history := []db.WorkoutWithExercises{{
		Exercises: []db.Exercise{{Name: "bench press", YoutubeURL: "https://www.youtube.com/watch?v=stored"}},
	}}
	finder := &fakePlanVideoFinder{urls: map[string]string{
		"Bulgarian Split Squat": "https://www.youtube.com/watch?v=fresh",
	}}

	errs := populatePlanVideos(context.Background(), workout, history, finder)

	require.Empty(t, errs)
	require.Equal(t, "https://www.youtube.com/watch?v=stored", workout.Exercises[0].YoutubeURL)
	require.Equal(t, "https://www.youtube.com/watch?v=fresh", workout.Exercises[1].YoutubeURL)
	require.Equal(t, "https://www.youtube.com/results?search_query=Back+Pull+exercise+technique", workout.Exercises[2].YoutubeURL)
	require.ElementsMatch(t, []string{"Bulgarian Split Squat", "Back Pull"}, finder.calls)
	formatted := FormatWorkout(workout)
	assert.Contains(t, formatted, "https://www.youtube.com/watch?v=stored")
	assert.Contains(t, formatted, "https://www.youtube.com/watch?v=fresh")
	assert.Contains(t, formatted, "https://www.youtube.com/results?search_query=Back+Pull+exercise+technique")
}

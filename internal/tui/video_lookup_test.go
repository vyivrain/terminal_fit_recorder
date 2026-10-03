package tui

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/db"
)

type fakeVideoFinder struct {
	url   string
	err   error
	calls []string
}

func (f *fakeVideoFinder) Find(_ context.Context, exerciseName string) (string, error) {
	f.calls = append(f.calls, exerciseName)
	return f.url, f.err
}

// runBatch executes every child command in a tea.Batch result (as bubbletea's
// runtime would) and returns the resulting messages, matching the pattern
// exercise_editor_test.go already uses for the AI refine flow.
func runBatch(command tea.Cmd) []tea.Msg {
	if command == nil {
		return nil
	}
	msg := command()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var messages []tea.Msg
	for _, child := range batch {
		messages = append(messages, runBatch(child)...)
	}
	return messages
}

func TestEnqueueVideoLookupsSkipsAlreadyFoundAndUnsavedExercises(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)
	finder := &fakeVideoFinder{url: "https://www.youtube.com/watch?v=xyz"}
	app.videoFinder = finder

	workout := &db.WorkoutWithExercises{
		Workout: db.Workout{WorkoutType: "strength", WorkoutDate: time.Now()},
		Exercises: []db.Exercise{
			{ID: 1, Name: "Squat"}, // needs a lookup
			{ID: 2, Name: "Bench Press", YoutubeURL: "https://already/found"}, // already has one
			{ID: 0, Name: "Unsaved Exercise"},                                 // never persisted, no id yet
		},
	}

	command := app.enqueueVideoLookups([]*db.WorkoutWithExercises{workout})
	require.NotNil(t, command)
	runBatch(command)
	require.Equal(t, []string{"Squat"}, finder.calls)
}

func TestEnqueueVideoLookupsReturnsNilWhenNothingToLookUp(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)
	app.videoFinder = &fakeVideoFinder{}

	workout := &db.WorkoutWithExercises{
		Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Now()},
		Exercises: []db.Exercise{{ID: 1, Name: "Squat", YoutubeURL: "https://already/found"}},
	}
	require.Nil(t, app.enqueueVideoLookups([]*db.WorkoutWithExercises{workout}))
}

func TestSavingGeneratedWorkoutTriggersVideoLookupAndPersistsResult(t *testing.T) {
	database := tuiTestDatabase(t)
	app, err := New(database, fakeGenerator{})
	require.NoError(t, err)
	app.videoFinder = &fakeVideoFinder{url: "https://www.youtube.com/watch?v=abc123"}

	workout := &db.WorkoutWithExercises{
		Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Now()},
		Exercises: []db.Exercise{{Name: "Squat", Weight: 100, Repetitions: 5, Sets: 5}},
	}
	app.Update(aiResultMsg{operation: aiGenerate, workout: workout})
	require.Equal(t, screenAIPreview, app.screen)

	_, command := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	require.Equal(t, screenWorkouts, app.screen)

	for _, msg := range runBatch(command) {
		app.Update(msg)
	}

	saved, err := database.GetAllWorkouts()
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.Equal(t, "https://www.youtube.com/watch?v=abc123", saved[0].Exercises[0].YoutubeURL)
}

func TestVideoLookupFailureWarnsOnlyOnce(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)

	app.Update(videoLookupMsg{exerciseID: 1, err: errors.New("yt-dlp: not found")})
	require.True(t, app.videoLookupWarned)
	require.Contains(t, app.statusMessage, "yt-dlp")

	app.statusMessage = "unchanged"
	app.Update(videoLookupMsg{exerciseID: 2, err: errors.New("yt-dlp: not found")})
	require.Equal(t, "unchanged", app.statusMessage, "a second failure must not re-trigger the warning")
}

func TestVideoLookupNoResultIsSilent(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)
	app.statusMessage = "unchanged"

	app.Update(videoLookupMsg{exerciseID: 1, url: ""})
	require.Equal(t, "unchanged", app.statusMessage)
	require.False(t, app.videoLookupWarned)
}

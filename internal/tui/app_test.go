package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/ai"
	"terminal_fit_recorder/internal/db"
)

type fakeGenerator struct{}

func (fakeGenerator) Generate(context.Context, db.AIModel, []db.WorkoutWithExercises, string) (*ai.GenerationResult, error) {
	return nil, errors.New("model endpoint unavailable")
}

func (fakeGenerator) Refine(context.Context, db.AIModel, *db.WorkoutWithExercises, string, string) (*ai.GenerationResult, error) {
	return nil, errors.New("model endpoint unavailable")
}

func (fakeGenerator) ImportNotes(context.Context, db.AIModel, string, []db.WorkoutWithExercises) ([]*db.WorkoutWithExercises, error) {
	return nil, errors.New("model endpoint unavailable")
}

func tuiTestDatabase(t *testing.T) *db.DB {
	t.Helper()

	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(filepath.Clean(filepath.Join(workingDirectory, "../.."))))
	t.Cleanup(func() { require.NoError(t, os.Chdir(workingDirectory)) })

	database, err := db.New(filepath.Join(t.TempDir(), "tui.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func TestTUIWorkoutProfileAndModelNavigation(t *testing.T) {
	database := tuiTestDatabase(t)
	now := time.Now()
	workoutDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	workoutID, err := database.CreateWorkout("strength", "completed", workoutDate)
	require.NoError(t, err)
	require.NoError(t, database.SaveExercisesForWorkout(workoutID, []db.Exercise{{Name: "Bench Press", Weight: 80, Repetitions: 8, Sets: 3}}))

	app, err := New(database, fakeGenerator{})
	require.NoError(t, err)
	require.Equal(t, screenWorkouts, app.screen)
	require.Equal(t, db.DefaultProfileName, app.activeProfile.Name)
	require.Equal(t, "opencode-go/glm-5.3", app.selectedModel.ModelID)
	require.Contains(t, app.View(), "Terminal Fit Recorder")

	sendKey(app, "enter")
	require.Equal(t, screenWorkoutDetail, app.screen)
	require.Contains(t, app.View(), "Bench Press")
	sendKey(app, "esc")

	sendKey(app, "p")
	require.Equal(t, screenProfiles, app.screen)
	sendKey(app, "c")
	require.Equal(t, screenProfileInput, app.screen)
	app.input.SetValue("partner")
	sendKey(app, "enter")
	require.Equal(t, screenWorkouts, app.screen)
	require.Equal(t, "partner", app.activeProfile.Name)
	require.Empty(t, app.workoutList.Items(), "new profiles start with an empty workout list")

	sendKey(app, "p")
	sendKey(app, "e")
	app.input.SetValue("training partner")
	sendKey(app, "enter")
	require.Equal(t, "training partner", app.activeProfile.Name)

	sendKey(app, "p")
	sendKey(app, "d")
	require.Equal(t, confirmDeleteProfile, app.confirm)
	sendKey(app, "y")
	require.Equal(t, screenWorkouts, app.screen)
	require.Equal(t, db.DefaultProfileName, app.activeProfile.Name)
	require.Len(t, app.workoutList.Items(), 1)

	sendKey(app, "m")
	require.Equal(t, screenModels, app.screen)
	app.Update(tea.KeyMsg{Type: tea.KeyDown})
	sendKey(app, "enter")
	require.Equal(t, screenWorkouts, app.screen)
	require.Equal(t, "gpt-5.6-sol", app.selectedModel.ModelID)
}

func TestTUIFiltersWorkoutsWithoutHidingAIHistory(t *testing.T) {
	database := tuiTestDatabase(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local)
	past := today.AddDate(0, 0, -2)
	future := today.AddDate(0, 0, 3)

	addWorkout := func(date time.Time, status, exercise string) {
		workoutID, err := database.CreateWorkout("strength", status, date)
		require.NoError(t, err)
		require.NoError(t, database.SaveExercisesForWorkout(workoutID, []db.Exercise{{Name: exercise, Repetitions: 8, Sets: 3}}))
	}
	addWorkout(past, "completed", "Past Row")
	addWorkout(today, "completed", "Today Row")
	addWorkout(future, "planned", "Future Row")

	app, err := New(database, fakeGenerator{})
	require.NoError(t, err)
	require.Equal(t, []string{today.Format("2006-01-02"), future.Format("2006-01-02")}, visibleWorkoutDates(app))
	require.Contains(t, app.View(), "Upcoming")
	require.Contains(t, app.View(), "All statuses")

	sendKey(app, "a")
	require.Len(t, app.workoutList.Items(), 3)
	require.Contains(t, app.View(), "All dates")
	require.Contains(t, app.View(), "a upcoming")

	sendKey(app, "f")
	require.True(t, app.statusFilterOpen)
	require.Contains(t, app.View(), "Filter workouts by status")
	sendKey(app, "c")
	require.False(t, app.statusFilterOpen)
	require.Equal(t, []string{today.Format("2006-01-02"), past.Format("2006-01-02")}, visibleWorkoutDates(app))
	require.Contains(t, app.View(), "Completed only")

	sendKey(app, "a")
	require.Equal(t, []string{today.Format("2006-01-02")}, visibleWorkoutDates(app))
	require.Contains(t, app.View(), "a all dates")

	sendKey(app, "f")
	sendKey(app, "a")
	require.Equal(t, []string{today.Format("2006-01-02"), future.Format("2006-01-02")}, visibleWorkoutDates(app))

	require.Len(t, app.currentWorkouts(), 3, "date and status filters must not change AI history")
	completed := app.completedWorkouts()
	require.Len(t, completed, 2)
	require.Equal(t, today.Format("2006-01-02"), completed[0].Workout.WorkoutDate.Format("2006-01-02"))
	require.Equal(t, past.Format("2006-01-02"), completed[1].Workout.WorkoutDate.Format("2006-01-02"))
}

func TestTUIShowsAIErrorAsModal(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)

	app.loading = true
	app.Update(aiResultMsg{err: errors.New("model endpoint unavailable")})
	require.False(t, app.loading)
	require.Contains(t, app.modalError, "model endpoint unavailable")
	require.Contains(t, app.View(), "Something went wrong")
	require.Contains(t, app.View(), "model endpoint unavailable")

	sendKey(app, "enter")
	require.Empty(t, app.modalError)
}

func TestTUIShowsSmallTerminalFallback(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)

	app.Update(tea.WindowSizeMsg{Width: 30, Height: 8})
	view := app.View()
	require.Contains(t, view, "Resize terminal")
	require.Contains(t, view, "minimum 60x18")
	require.NotContains(t, view, "enter view")
}

func TestTUIShowsSingleEmptyWorkoutMessage(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)

	require.Equal(t, 1, strings.Count(app.View(), "No workouts"))
}

func TestTUIAcceptsFreeFormWorkoutNotesFile(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)

	notesPath := filepath.Join(t.TempDir(), "training notes.txt")
	require.NoError(t, os.WriteFile(notesPath, []byte("ПОНЕДІЛОК: підтягування з резинкою"), 0600))

	sendKey(app, "i")
	require.Equal(t, screenImportPath, app.screen)
	app.input.SetValue(`"` + notesPath + `"`)
	_, command := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.True(t, app.loading)
	require.NotNil(t, command)
	require.Contains(t, app.loadingText, "training notes.txt")
	app.aiCancel()
}

func TestTUIBrowsesAndSavesImportedWorkoutsAsPlanned(t *testing.T) {
	database := tuiTestDatabase(t)
	app, err := New(database, fakeGenerator{})
	require.NoError(t, err)

	workouts := []*db.WorkoutWithExercises{
		{
			Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Date(2026, time.September, 7, 0, 0, 0, 0, time.Local)},
			Exercises: []db.Exercise{{Name: "Assisted Pull-Up", Repetitions: 15, Sets: 3}},
		},
		{
			Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Date(2026, time.September, 9, 0, 0, 0, 0, time.Local)},
			Exercises: []db.Exercise{{Name: "Dumbbell Hammer Curl", Weight: 10, Repetitions: 10, Sets: 3}},
		},
	}
	app.Update(aiResultMsg{workouts: workouts, operation: aiImport})
	require.Equal(t, screenAIPreview, app.screen)
	require.Contains(t, app.View(), "workout 1 of 2")
	require.Contains(t, app.View(), "Assisted Pull-Up")

	app.Update(tea.KeyMsg{Type: tea.KeyRight})
	require.Contains(t, app.View(), "workout 2 of 2")
	require.Contains(t, app.View(), "Dumbbell Hammer Curl")

	sendKey(app, "s")
	require.Equal(t, screenWorkouts, app.screen)
	require.Contains(t, app.statusMessage, "2 planned workouts")
	saved, err := database.GetAllWorkouts()
	require.NoError(t, err)
	require.Len(t, saved, 2)
	require.Equal(t, "planned", saved[0].Workout.Status)
	require.Equal(t, "planned", saved[1].Workout.Status)
}

func TestTUISavesSuggestedProfileUpdateAlongsideGeneratedWorkout(t *testing.T) {
	database := tuiTestDatabase(t)
	app, err := New(database, fakeGenerator{})
	require.NoError(t, err)
	require.NotNil(t, app.activeProfile)

	workout := &db.WorkoutWithExercises{
		Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Date(2026, time.September, 7, 0, 0, 0, 0, time.Local)},
		Exercises: []db.Exercise{{Name: "Squat", Weight: 100, Repetitions: 5, Sets: 5}},
	}
	app.Update(aiResultMsg{operation: aiGenerate, workout: workout, profileUpdate: "Bad left knee as of Sept 2026 — avoid deep squats and lunges."})
	require.Equal(t, screenAIPreview, app.screen)
	require.Contains(t, app.View(), "Profile update suggested")
	require.Contains(t, app.View(), "Bad left knee")

	sendKey(app, "s")
	require.Equal(t, screenWorkouts, app.screen)
	require.Contains(t, app.statusMessage, "profile description updated")
	require.Empty(t, app.pendingProfileUpdate)

	active, err := database.GetActiveProfile()
	require.NoError(t, err)
	require.Equal(t, "Bad left knee as of Sept 2026 — avoid deep squats and lunges.", active.Description)
}

func TestTUIDiscardingPreviewDropsSuggestedProfileUpdate(t *testing.T) {
	database := tuiTestDatabase(t)
	app, err := New(database, fakeGenerator{})
	require.NoError(t, err)

	workout := &db.WorkoutWithExercises{
		Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Date(2026, time.September, 7, 0, 0, 0, 0, time.Local)},
		Exercises: []db.Exercise{{Name: "Squat", Weight: 100, Repetitions: 5, Sets: 5}},
	}
	app.Update(aiResultMsg{operation: aiGenerate, workout: workout, profileUpdate: "Should not be saved"})
	sendKey(app, "esc")
	require.Empty(t, app.pendingProfileUpdate)

	active, err := database.GetActiveProfile()
	require.NoError(t, err)
	require.Empty(t, active.Description)
}

func TestTUILongImportPreviewCanScroll(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)

	exercises := make([]db.Exercise, 20)
	for index := range exercises {
		exercises[index] = db.Exercise{Name: fmt.Sprintf("Exercise %02d", index+1), Repetitions: 10, Sets: 3}
	}
	app.Update(aiResultMsg{
		operation: aiImport,
		workouts: []*db.WorkoutWithExercises{{
			Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Date(2026, time.September, 7, 0, 0, 0, 0, time.Local)},
			Exercises: exercises,
		}},
	})
	require.NotContains(t, app.View(), "Exercise 20")

	for range 60 {
		app.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	require.Contains(t, app.View(), "Exercise 20")
}

func TestTUIProgramStartsAndQuitsWithTemporaryDatabase(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)

	program := tea.NewProgram(
		app,
		tea.WithInput(strings.NewReader("q")),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
	)
	_, err = program.Run()
	require.NoError(t, err)
}

func TestWorkoutFormBuildsCompleteWorkout(t *testing.T) {
	form := newWorkoutForm(time.Date(2026, time.September, 6, 0, 0, 0, 0, time.Local))
	form.inputs[1].SetValue("Run")
	form.inputs[5].SetValue("30")
	form.inputs[6].SetValue("5000")
	form.workoutType = 1

	workout, err := form.workout()
	require.NoError(t, err)
	require.Equal(t, "cardio", workout.Workout.WorkoutType)
	require.Equal(t, "completed", workout.Workout.Status)
	require.Equal(t, "Run", workout.Exercises[0].Name)
	require.Equal(t, 30.0, workout.Exercises[0].Duration)
	require.Equal(t, 5000, workout.Exercises[0].Distance)
}

func sendKey(app *App, key string) {
	message := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case "enter":
		message = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		message = tea.KeyMsg{Type: tea.KeyEsc}
	}
	app.Update(message)
}

func visibleWorkoutDates(app *App) []string {
	items := app.workoutList.Items()
	dates := make([]string, 0, len(items))
	for _, item := range items {
		workout, ok := item.(workoutItem)
		if ok {
			dates = append(dates, workout.workout.Workout.WorkoutDate.Format("2006-01-02"))
		}
	}
	return dates
}

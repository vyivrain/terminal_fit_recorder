package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"

	"terminal_fit_recorder/internal/ai"
	"terminal_fit_recorder/internal/db"
)

func editorWorkout() *db.WorkoutWithExercises {
	return &db.WorkoutWithExercises{
		Workout:   db.Workout{WorkoutType: "strength", WorkoutDate: time.Now(), Status: "planned"},
		Exercises: []db.Exercise{{Name: "Squat", Weight: 50, Repetitions: 8, Sets: 3}, {Name: "Run", Duration: 20, Distance: 3000}},
	}
}

func TestTUIManualEditsPersistForEachImportedWorkout(t *testing.T) {
	database := tuiTestDatabase(t)
	app, err := New(database, fakeGenerator{})
	require.NoError(t, err)
	first, second := editorWorkout(), editorWorkout()
	second.Workout.WorkoutDate = first.Workout.WorkoutDate.AddDate(0, 0, 1)
	app.Update(aiResultMsg{operation: aiImport, workouts: []*db.WorkoutWithExercises{first, second}})
	sendKey(app, "e")
	require.Equal(t, screenExerciseEditor, app.screen)
	require.Equal(t, first.Exercises, app.exerciseEditor.workout.Exercises)
	for column, value := range []string{"Присідання", "60", "10", "4", "1.5", "100"} {
		app.exerciseEditor.column = column
		app.Update(tea.KeyMsg{Type: tea.KeyEnter})
		app.exerciseEditor.input.SetValue(value)
		app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	}
	app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	require.Equal(t, screenAIPreview, app.screen)
	require.Equal(t, "Squat", first.Exercises[0].Name)
	unsaved, err := database.GetAllWorkouts()
	require.NoError(t, err)
	require.Empty(t, unsaved)

	app.Update(tea.KeyMsg{Type: tea.KeyRight})
	sendKey(app, "e")
	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app.exerciseEditor.input.SetValue("Lunge")
	// Applying the table also commits an active cell.
	app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	require.Equal(t, "Присідання", app.currentPreviewWorkout().Exercises[0].Name)
	sendKey(app, "s")
	require.Empty(t, app.modalError)
	saved, err := database.GetAllWorkouts()
	require.NoError(t, err)
	require.Len(t, saved, 2)
	require.Equal(t, "Lunge", saved[0].Exercises[0].Name)
	exercise := saved[1].Exercises[0]
	require.Equal(t, "Присідання", exercise.Name)
	require.Equal(t, 60, exercise.Weight)
	require.Equal(t, 10, exercise.Repetitions)
	require.Equal(t, 4, exercise.Sets)
	require.Equal(t, 1.5, exercise.Duration)
	require.Equal(t, 100, exercise.Distance)
	require.Equal(t, "Run", saved[1].Exercises[1].Name)
	require.Equal(t, "planned", saved[1].Workout.Status)
	require.Equal(t, first.Workout.WorkoutDate.Format("2006-01-02"), saved[1].Workout.WorkoutDate.Format("2006-01-02"))
}

func TestTUIEditorCancellationPreservesPreview(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)
	app.Update(aiResultMsg{operation: aiGenerate, workout: editorWorkout()})
	sendKey(app, "e")
	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app.exerciseEditor.input.SetValue("Deadlift")
	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.Equal(t, screenExerciseEditor, app.screen)
	require.Equal(t, "Squat", app.exerciseEditor.workout.Exercises[0].Name)
	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app.exerciseEditor.input.SetValue("Deadlift")
	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.Equal(t, screenAIPreview, app.screen)
	require.Equal(t, "Squat", app.currentPreviewWorkout().Exercises[0].Name)
}

func TestExerciseEditorRejectsInvalidValues(t *testing.T) {
	for _, test := range []struct {
		column int
		value  string
	}{{0, " "}, {1, "-1"}, {1, "2.5"}, {1, "1001"}, {2, "abc"}, {2, "1001"}, {3, "101"}, {4, "NaN"}, {4, "+Inf"}, {4, "-2"}, {4, "1441"}, {5, "1000001"}} {
		t.Run(fmt.Sprintf("%d/%s", test.column, test.value), func(t *testing.T) {
			original := editorWorkout()
			editor := newExerciseEditor(original, 88, 28)
			editor.column = test.column
			editor.update(tea.KeyMsg{Type: tea.KeyEnter})
			editor.input.SetValue(test.value)
			outcome, _ := editor.update(tea.KeyMsg{Type: tea.KeyCtrlS})
			require.Equal(t, formContinue, outcome)
			require.True(t, editor.editing)
			require.NotEmpty(t, editor.err)
			require.Equal(t, original.Exercises, editor.workout.Exercises)
		})
	}
}

type refinementRecorder struct {
	fakeGenerator
	received *db.WorkoutWithExercises
	prompt   string
}

func (recorder *refinementRecorder) Refine(_ context.Context, _ db.AIModel, workout *db.WorkoutWithExercises, prompt, _ string) (*ai.GenerationResult, error) {
	recorder.received = workout
	recorder.prompt = prompt
	result := *workout
	result.Exercises = append([]db.Exercise(nil), workout.Exercises...)
	result.Exercises[1].Name = "Walk"
	return &ai.GenerationResult{Workout: &result}, nil
}

func TestTUIAIRefinementUsesManualEditsAndCanBeEditedAgain(t *testing.T) {
	generator := &refinementRecorder{}
	app, err := New(tuiTestDatabase(t), generator)
	require.NoError(t, err)
	app.Update(aiResultMsg{operation: aiImport, workouts: []*db.WorkoutWithExercises{editorWorkout()}})
	sendKey(app, "e")
	app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app.exerciseEditor.input.SetValue("70")
	app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	sendKey(app, "a")
	require.Equal(t, screenRefineInput, app.screen)
	app.input.SetValue("Replace running with walking")
	_, command := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, child := range command().(tea.BatchMsg) {
		if result, ok := child().(aiResultMsg); ok {
			app.Update(result)
		}
	}
	require.Equal(t, 70, generator.received.Exercises[0].Weight)
	require.Equal(t, "Replace running with walking", generator.prompt)
	require.Equal(t, screenAIPreview, app.screen)
	sendKey(app, "e")
	require.Equal(t, 70, app.exerciseEditor.workout.Exercises[0].Weight)
	require.Equal(t, "Walk", app.exerciseEditor.workout.Exercises[1].Name)
}

func TestExerciseEditorScrollsAndFitsMinimumTerminal(t *testing.T) {
	app, err := New(tuiTestDatabase(t), fakeGenerator{})
	require.NoError(t, err)
	workout := editorWorkout()
	for index := 3; index <= 20; index++ {
		workout.Exercises = append(workout.Exercises, db.Exercise{Name: fmt.Sprintf("Exercise %02d", index)})
	}
	app.Update(aiResultMsg{operation: aiImport, workouts: []*db.WorkoutWithExercises{workout}})
	sendKey(app, "e")
	app.Update(tea.WindowSizeMsg{Width: 60, Height: 18})
	for range 19 {
		app.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	require.Equal(t, 19, app.exerciseEditor.table.Cursor())
	require.Contains(t, app.View(), "Exercise 20")
	for _, line := range strings.Split(app.exerciseEditor.view(60), "\n") {
		require.LessOrEqual(t, lipgloss.Width(line), 52)
	}
	require.LessOrEqual(t, lipgloss.Height(app.View()), 18)
}

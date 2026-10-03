package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminal_fit_recorder/internal/ai"
	"terminal_fit_recorder/internal/db"
	"terminal_fit_recorder/internal/youtube"
)

type screen int

const (
	screenWorkouts screen = iota
	screenWorkoutDetail
	screenProfiles
	screenModels
	screenWorkoutForm
	screenProfileInput
	screenImportPath
	screenAIPreview
	screenExerciseEditor
	screenRefineInput
)

type profileInputAction int

const (
	profileCreate profileInputAction = iota
	profileRename
	profileDescribe
)

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDeleteWorkout
	confirmDeleteProfile
)

type aiOperation int

const (
	aiGenerate aiOperation = iota
	aiRefine
	aiImport
)

type workoutDateScope int

const (
	workoutUpcoming workoutDateScope = iota
	workoutAll
)

type workoutStatusFilter int

const (
	statusAll workoutStatusFilter = iota
	statusPlanned
	statusCompleted
)

type aiResultMsg struct {
	workout       *db.WorkoutWithExercises
	workouts      []*db.WorkoutWithExercises
	profileUpdate string
	err           error
	operation     aiOperation
}

// videoLookupMsg carries the outcome of one background technique-video
// search for a single, already-saved exercise.
type videoLookupMsg struct {
	exerciseID int
	url        string
	err        error
}

// enqueueVideoLookups returns a batched command that searches, in the
// background, for a short technique video for every exercise in workouts
// that does not already have one. It returns nil when there is nothing to
// look up or no finder is configured.
func (app *App) enqueueVideoLookups(workouts []*db.WorkoutWithExercises) tea.Cmd {
	if app.videoFinder == nil {
		return nil
	}

	var commands []tea.Cmd
	for _, workout := range workouts {
		if workout == nil {
			continue
		}
		for _, exercise := range workout.Exercises {
			if exercise.ID == 0 || exercise.YoutubeURL != "" || strings.TrimSpace(exercise.Name) == "" {
				continue
			}
			exerciseID := exercise.ID
			name := exercise.Name
			commands = append(commands, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				url, err := app.videoFinder.Find(ctx, name)
				return videoLookupMsg{exerciseID: exerciseID, url: url, err: err}
			})
		}
	}
	if len(commands) == 0 {
		return nil
	}
	return tea.Batch(commands...)
}

// applyVideoLookup records a background technique-video search's result. A
// search failure (e.g. yt-dlp missing) surfaces once as a status message
// rather than once per exercise; finding nothing under a minute is a normal,
// silent outcome, not an error.
func (app *App) applyVideoLookup(result videoLookupMsg) (tea.Model, tea.Cmd) {
	if result.err != nil {
		if !app.videoLookupWarned {
			app.videoLookupWarned = true
			app.statusMessage = "Could not look up technique videos (is yt-dlp installed?)"
		}
		return app, nil
	}
	if result.url == "" {
		return app, nil
	}
	if err := app.database.SetExerciseYoutubeURL(result.exerciseID, result.url); err != nil {
		return app, nil
	}
	if err := app.refreshWorkouts(); err != nil {
		return app, nil
	}
	if app.detailWorkout != nil {
		for index := range app.detailWorkout.Exercises {
			if app.detailWorkout.Exercises[index].ID == result.exerciseID {
				app.detailWorkout.Exercises[index].YoutubeURL = result.url
				if app.screen == screenWorkoutDetail {
					app.setViewedWorkout(app.detailWorkout)
				}
			}
		}
	}
	return app, nil
}

type workoutItem struct{ workout db.WorkoutWithExercises }

func (item workoutItem) FilterValue() string {
	return item.workout.Workout.WorkoutDate.Format("2006-01-02") + " " + item.workout.Workout.WorkoutType
}
func (item workoutItem) Title() string {
	return fmt.Sprintf("%s  ·  %s", item.workout.Workout.WorkoutDate.Format("Mon, 02 Jan 2006"), displayWorkoutType(item.workout.Workout.WorkoutType))
}
func (item workoutItem) Description() string {
	status := "Done"
	if item.workout.Workout.Status == "planned" {
		status = "AI planned"
	}
	return fmt.Sprintf("%s  ·  %d exercises", status, len(item.workout.Exercises))
}

type profileItem struct{ profile db.Profile }

func (item profileItem) FilterValue() string { return item.profile.Name }
func (item profileItem) Title() string       { return item.profile.Name }
func (item profileItem) Description() string {
	status := "Press enter to make active"
	if item.profile.IsActive {
		status = "Active · default"
	}
	if description := strings.TrimSpace(item.profile.Description); description != "" {
		const previewLimit = 60
		if runes := []rune(description); len(runes) > previewLimit {
			description = string(runes[:previewLimit]) + "…"
		}
		status += "  ·  " + description
	}
	return status
}

type modelItem struct{ model db.AIModel }

func (item modelItem) FilterValue() string { return item.model.DisplayName + " " + item.model.ModelID }
func (item modelItem) Title() string       { return item.model.DisplayName }
func (item modelItem) Description() string {
	if item.model.IsSelected {
		return item.model.ModelID + " · selected"
	}
	return item.model.ModelID
}

// exerciseVideoFinder is the subset of *youtube.Finder the TUI depends on,
// so tests can inject a fake instead of shelling out to yt-dlp.
type exerciseVideoFinder interface {
	Find(ctx context.Context, exerciseName string) (string, error)
}

type App struct {
	database    *db.DB
	generator   ai.Generator
	videoFinder exerciseVideoFinder
	// videoLookupWarned keeps a failing technique-video lookup (e.g. yt-dlp
	// not installed) from popping a status message for every exercise.
	videoLookupWarned bool

	screen          screen
	width           int
	height          int
	workoutList     list.Model
	profileList     list.Model
	modelList       list.Model
	workouts        []db.WorkoutWithExercises
	dateScope       workoutDateScope
	statusFilter    workoutStatusFilter
	detailViewport  viewport.Model
	activeProfile   *db.Profile
	selectedModel   *db.AIModel
	detailWorkout   *db.WorkoutWithExercises
	previewWorkouts []*db.WorkoutWithExercises
	previewIndex    int
	previewOrigin   aiOperation
	// pendingProfileUpdate is a profile-description change the AI suggested
	// alongside the current preview. It is shown for review and only written
	// to the profile when the previewed workout(s) are saved.
	pendingProfileUpdate string

	input              textinput.Model
	profileInputAction profileInputAction
	profileInputTarget int
	onboarding         bool
	form               workoutForm
	exerciseEditor     exerciseEditor

	confirm          confirmAction
	confirmText      string
	modalError       string
	statusFilterOpen bool
	statusMessage    string

	spinner      spinner.Model
	loading      bool
	loadingText  string
	aiCancel     context.CancelFunc
	refinePrompt string
}

func New(database *db.DB, generator ai.Generator) (*App, error) {
	app := &App{
		database:    database,
		generator:   generator,
		videoFinder: youtube.NewFinder(3),
		width:       88,
		height:      28,
	}

	app.spinner = spinner.New()
	app.spinner.Spinner = spinner.Dot
	app.spinner.Style = lipgloss.NewStyle().Foreground(accentColor)
	app.input = textinput.New()
	app.input.CharLimit = 500
	app.input.Width = 60

	app.workoutList = newList("workout", "workouts")
	app.profileList = newList("profile", "profiles")
	app.modelList = newList("model", "models")
	app.detailViewport = viewport.New(detailViewportWidth(app.width), detailViewportHeight(app.height))

	if err := app.refreshAll(); err != nil {
		return nil, err
	}
	if app.activeProfile == nil {
		app.startProfileInput(profileCreate, 0, "")
		app.onboarding = true
	} else {
		app.screen = screenWorkouts
	}
	app.resize()
	return app, nil
}

func Run(database *db.DB, generator ai.Generator) error {
	app, err := New(database, generator)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(app, tea.WithAltScreen()).Run()
	return err
}

func newList(singular, plural string) list.Model {
	model := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	model.SetShowTitle(false)
	model.SetShowHelp(false)
	model.SetShowStatusBar(true)
	model.SetFilteringEnabled(true)
	model.SetStatusBarItemName(singular, plural)
	model.DisableQuitKeybindings()
	return model
}

func (app *App) Init() tea.Cmd { return nil }

func (app *App) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := message.(tea.WindowSizeMsg); ok {
		app.width = size.Width
		app.height = size.Height
		app.resize()
		return app, nil
	}

	if result, ok := message.(videoLookupMsg); ok {
		return app.applyVideoLookup(result)
	}

	if result, ok := message.(aiResultMsg); ok {
		app.loading = false
		if app.aiCancel != nil {
			app.aiCancel()
			app.aiCancel = nil
		}
		if result.err != nil {
			prefix := "Generation failed"
			if result.operation == aiImport {
				prefix = "Import failed"
			}
			if result.operation == aiRefine {
				prefix = "Modification failed"
				if ai.IsRejected(result.err) {
					prefix = "Modification refused"
				}
			}
			app.modalError = prefix + "\n\n" + result.err.Error()
			return app, nil
		}
		switch result.operation {
		case aiImport:
			if len(result.workouts) == 0 {
				app.modalError = "Import failed\n\nThe model did not return any workouts."
				return app, nil
			}
			app.previewWorkouts = result.workouts
			app.previewIndex = 0
			app.previewOrigin = aiImport
		case aiRefine:
			if result.workout == nil || len(app.previewWorkouts) == 0 {
				app.modalError = "Modification failed\n\nThe model did not return a workout."
				return app, nil
			}
			app.previewWorkouts[app.previewIndex] = result.workout
		default:
			if result.workout == nil {
				app.modalError = "Generation failed\n\nThe model did not return a workout."
				return app, nil
			}
			app.previewWorkouts = []*db.WorkoutWithExercises{result.workout}
			app.previewIndex = 0
			app.previewOrigin = aiGenerate
		}
		app.pendingProfileUpdate = result.profileUpdate
		app.screen = screenAIPreview
		app.setViewedWorkout(app.currentPreviewWorkout())
		return app, nil
	}

	if app.modalError != "" {
		if key, ok := message.(tea.KeyMsg); ok && (key.Type == tea.KeyEnter || key.Type == tea.KeyEsc) {
			app.modalError = ""
		}
		return app, nil
	}

	if app.confirm != confirmNone {
		return app.updateConfirmation(message)
	}

	if app.loading {
		if key, ok := message.(tea.KeyMsg); ok {
			if key.Type == tea.KeyCtrlC {
				if app.aiCancel != nil {
					app.aiCancel()
				}
				return app, tea.Quit
			}
			if key.Type == tea.KeyEsc && app.aiCancel != nil {
				app.loadingText = "Cancelling…"
				app.aiCancel()
			}
		}
		var command tea.Cmd
		app.spinner, command = app.spinner.Update(message)
		return app, command
	}

	if key, ok := message.(tea.KeyMsg); ok && key.Type == tea.KeyCtrlC {
		return app, tea.Quit
	}
	if app.statusFilterOpen {
		return app.updateStatusFilter(message)
	}

	switch app.screen {
	case screenWorkouts:
		return app.updateWorkouts(message)
	case screenWorkoutDetail:
		return app.updateWorkoutDetail(message)
	case screenProfiles:
		return app.updateProfiles(message)
	case screenModels:
		return app.updateModels(message)
	case screenWorkoutForm:
		return app.updateWorkoutForm(message)
	case screenProfileInput:
		return app.updateProfileInput(message)
	case screenImportPath:
		return app.updateImportPath(message)
	case screenAIPreview:
		return app.updateAIPreview(message)
	case screenExerciseEditor:
		return app.updateExerciseEditor(message)
	case screenRefineInput:
		return app.updateRefineInput(message)
	default:
		return app, nil
	}
}

func (app *App) updateWorkouts(message tea.Msg) (tea.Model, tea.Cmd) {
	if app.workoutList.SettingFilter() {
		var command tea.Cmd
		app.workoutList, command = app.workoutList.Update(message)
		return app, command
	}
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.String() {
		case "q":
			return app, tea.Quit
		case "enter":
			if selected, ok := app.workoutList.SelectedItem().(workoutItem); ok {
				workout := selected.workout
				app.detailWorkout = &workout
				app.screen = screenWorkoutDetail
				app.setViewedWorkout(app.detailWorkout)
			}
			return app, nil
		case "n":
			app.form = newWorkoutForm(time.Now())
			app.screen = screenWorkoutForm
			return app, nil
		case "d":
			if selected, ok := app.workoutList.SelectedItem().(workoutItem); ok {
				app.detailWorkout = &selected.workout
				app.confirm = confirmDeleteWorkout
				app.confirmText = fmt.Sprintf("Delete workout from %s?", selected.workout.Workout.WorkoutDate.Format("02 Jan 2006"))
			}
			return app, nil
		case "g":
			return app.startGeneration(false)
		case "i":
			app.startImportPathInput()
			return app, textinput.Blink
		case "p":
			app.screen = screenProfiles
			return app, nil
		case "m":
			app.screen = screenModels
			return app, nil
		case "a":
			if app.dateScope == workoutAll {
				app.dateScope = workoutUpcoming
			} else {
				app.dateScope = workoutAll
			}
			app.applyWorkoutFilters()
			return app, nil
		case "f":
			app.statusFilterOpen = true
			return app, nil
		case "r":
			if err := app.refreshAll(); err != nil {
				app.modalError = err.Error()
			}
			return app, nil
		}
	}

	var command tea.Cmd
	app.workoutList, command = app.workoutList.Update(message)
	return app, command
}

func (app *App) updateStatusFilter(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return app, nil
	}

	switch key.String() {
	case "esc", "q":
		app.statusFilterOpen = false
		return app, nil
	case "0", "a":
		app.statusFilter = statusAll
	case "1", "p":
		app.statusFilter = statusPlanned
	case "2", "c":
		app.statusFilter = statusCompleted
	default:
		return app, nil
	}

	app.statusFilterOpen = false
	app.applyWorkoutFilters()
	return app, nil
}

func (app *App) updateWorkoutDetail(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return app, nil
	}
	switch key.String() {
	case "esc", "backspace", "q":
		app.screen = screenWorkouts
	case "d":
		if app.detailWorkout != nil {
			app.confirm = confirmDeleteWorkout
			app.confirmText = fmt.Sprintf("Delete workout from %s?", app.detailWorkout.Workout.WorkoutDate.Format("02 Jan 2006"))
		}
	default:
		var command tea.Cmd
		app.detailViewport, command = app.detailViewport.Update(message)
		return app, command
	}
	return app, nil
}

func (app *App) updateProfiles(message tea.Msg) (tea.Model, tea.Cmd) {
	if app.profileList.SettingFilter() {
		var command tea.Cmd
		app.profileList, command = app.profileList.Update(message)
		return app, command
	}
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "q":
			app.screen = screenWorkouts
			return app, nil
		case "enter":
			if selected, ok := app.profileList.SelectedItem().(profileItem); ok {
				if _, err := app.database.UseProfile(selected.profile.Name); err != nil {
					app.modalError = err.Error()
				} else if err := app.refreshAll(); err != nil {
					app.modalError = err.Error()
				} else {
					app.statusMessage = "Default profile changed to " + selected.profile.Name
					app.screen = screenWorkouts
				}
			}
			return app, nil
		case "c":
			app.startProfileInput(profileCreate, 0, "")
			return app, textinput.Blink
		case "e":
			if selected, ok := app.profileList.SelectedItem().(profileItem); ok {
				app.startProfileInput(profileRename, selected.profile.ID, selected.profile.Name)
				return app, textinput.Blink
			}
		case "i":
			if selected, ok := app.profileList.SelectedItem().(profileItem); ok {
				app.startProfileInput(profileDescribe, selected.profile.ID, selected.profile.Description)
				return app, textinput.Blink
			}
		case "d":
			if selected, ok := app.profileList.SelectedItem().(profileItem); ok {
				app.profileInputTarget = selected.profile.ID
				app.confirm = confirmDeleteProfile
				app.confirmText = fmt.Sprintf("Delete profile %q and all of its workouts?", selected.profile.Name)
			}
			return app, nil
		}
	}

	var command tea.Cmd
	app.profileList, command = app.profileList.Update(message)
	return app, command
}

func (app *App) updateModels(message tea.Msg) (tea.Model, tea.Cmd) {
	if app.modelList.SettingFilter() {
		var command tea.Cmd
		app.modelList, command = app.modelList.Update(message)
		return app, command
	}
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "q":
			app.screen = screenWorkouts
			return app, nil
		case "enter":
			if selected, ok := app.modelList.SelectedItem().(modelItem); ok {
				model, err := app.database.SelectAIModel(selected.model.ID)
				if err != nil {
					app.modalError = err.Error()
				} else {
					app.selectedModel = model
					if err := app.refreshModels(); err != nil {
						app.modalError = err.Error()
					} else {
						app.statusMessage = "AI model changed to " + model.DisplayName
						app.screen = screenWorkouts
					}
				}
			}
			return app, nil
		}
	}

	var command tea.Cmd
	app.modelList, command = app.modelList.Update(message)
	return app, command
}

func (app *App) updateWorkoutForm(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return app, nil
	}
	outcome, workout, command := app.form.update(key)
	switch outcome {
	case formCancelled:
		app.screen = screenWorkouts
	case formSave:
		if err := app.database.SaveWorkoutsWithExercises([]*db.WorkoutWithExercises{workout}, "completed"); err != nil {
			app.modalError = err.Error()
			return app, nil
		}
		lookupCmd := app.enqueueVideoLookups([]*db.WorkoutWithExercises{workout})
		if err := app.refreshWorkouts(); err != nil {
			app.modalError = err.Error()
			return app, nil
		}
		app.statusMessage = "Workout saved"
		app.screen = screenWorkouts
		return app, tea.Batch(command, lookupCmd)
	}
	return app, command
}

func (app *App) startProfileInput(action profileInputAction, profileID int, value string) {
	app.profileInputAction = action
	app.profileInputTarget = profileID
	app.input = textinput.New()
	if action == profileDescribe {
		app.input.CharLimit = 300
		app.input.Width = 68
		app.input.Placeholder = "Habits, injuries, preferences for the AI coach"
	} else {
		app.input.CharLimit = 80
		app.input.Width = 48
		app.input.Placeholder = "Profile name"
	}
	app.input.SetValue(value)
	app.input.Focus()
	app.screen = screenProfileInput
}

func (app *App) updateProfileInput(message tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEsc:
			if app.onboarding {
				return app, tea.Quit
			}
			app.screen = screenProfiles
			return app, nil
		case tea.KeyEnter:
			value := strings.TrimSpace(app.input.Value())
			var err error
			switch app.profileInputAction {
			case profileCreate:
				_, err = app.database.CreateProfile(value)
			case profileDescribe:
				_, err = app.database.SetProfileDescription(app.profileInputTarget, value)
			default:
				_, err = app.database.RenameProfile(app.profileInputTarget, value)
			}
			if err != nil {
				app.modalError = err.Error()
				return app, nil
			}
			app.onboarding = false
			if err := app.refreshAll(); err != nil {
				app.modalError = err.Error()
				return app, nil
			}
			app.statusMessage = "Profile saved"
			app.screen = screenWorkouts
			return app, nil
		}
	}
	var command tea.Cmd
	app.input, command = app.input.Update(message)
	return app, command
}

func (app *App) startImportPathInput() {
	app.input = textinput.New()
	app.input.CharLimit = 1000
	app.input.Width = 68
	app.input.Placeholder = "~/Downloads/training_1.txt"
	app.input.Focus()
	app.screen = screenImportPath
}

func (app *App) updateImportPath(message tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEsc:
			app.screen = screenWorkouts
			return app, nil
		case tea.KeyEnter:
			notes, path, err := readWorkoutNotesFile(app.input.Value())
			if err != nil {
				app.modalError = err.Error()
				return app, nil
			}
			return app.startImport(notes, path)
		}
	}
	var command tea.Cmd
	app.input, command = app.input.Update(message)
	return app, command
}

func (app *App) updateAIPreview(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return app, nil
	}
	switch key.String() {
	case "esc", "q":
		app.clearPreview()
		app.screen = screenWorkouts
	case "left", "h":
		if app.previewIndex > 0 {
			app.previewIndex--
			app.setViewedWorkout(app.currentPreviewWorkout())
		}
	case "right", "l":
		if app.previewIndex+1 < len(app.previewWorkouts) {
			app.previewIndex++
			app.setViewedWorkout(app.currentPreviewWorkout())
		}
	case "s":
		if len(app.previewWorkouts) == 0 {
			return app, nil
		}
		count := len(app.previewWorkouts)
		origin := app.previewOrigin
		profileUpdate := app.pendingProfileUpdate
		if err := app.database.SaveWorkoutsWithExercises(app.previewWorkouts, "planned"); err != nil {
			app.modalError = err.Error()
			return app, nil
		}
		if profileUpdate != "" && app.activeProfile != nil {
			if _, err := app.database.SetProfileDescription(app.activeProfile.ID, profileUpdate); err != nil {
				app.modalError = err.Error()
				return app, nil
			}
		}
		lookupCmd := app.enqueueVideoLookups(app.previewWorkouts)
		if err := app.refreshAll(); err != nil {
			app.modalError = err.Error()
			return app, nil
		}
		app.clearPreview()
		switch {
		case origin == aiImport:
			noun := "workout"
			if count != 1 {
				noun = "workouts"
			}
			app.statusMessage = fmt.Sprintf("Imported %d planned %s", count, noun)
		case profileUpdate != "":
			app.statusMessage = "AI workout saved as planned · profile description updated"
		default:
			app.statusMessage = "AI workout saved as planned"
		}
		app.screen = screenWorkouts
		return app, lookupCmd
	case "e":
		if workout := app.currentPreviewWorkout(); workout != nil {
			app.exerciseEditor = newExerciseEditor(workout, app.width, app.height)
			app.screen = screenExerciseEditor
		}
	case "a":
		if app.currentPreviewWorkout() == nil {
			return app, nil
		}
		app.input = textinput.New()
		app.input.CharLimit = 500
		app.input.Width = 68
		app.input.Placeholder = "e.g. Replace squats with a knee-friendly leg exercise"
		app.input.Focus()
		app.screen = screenRefineInput
		return app, textinput.Blink
	default:
		var command tea.Cmd
		app.detailViewport, command = app.detailViewport.Update(message)
		return app, command
	}
	return app, nil
}

func (app *App) currentPreviewWorkout() *db.WorkoutWithExercises {
	if app.previewIndex < 0 || app.previewIndex >= len(app.previewWorkouts) {
		return nil
	}
	return app.previewWorkouts[app.previewIndex]
}

func (app *App) updateExerciseEditor(message tea.Msg) (tea.Model, tea.Cmd) {
	outcome, command := app.exerciseEditor.update(message)
	switch outcome {
	case formSave:
		workout := app.exerciseEditor.workout
		app.previewWorkouts[app.previewIndex] = &workout
		app.setViewedWorkout(app.currentPreviewWorkout())
		app.screen = screenAIPreview
	case formCancelled:
		app.screen = screenAIPreview
	}
	return app, command
}

func (app *App) clearPreview() {
	app.previewWorkouts = nil
	app.previewIndex = 0
	app.previewOrigin = aiGenerate
	app.pendingProfileUpdate = ""
}

func (app *App) setViewedWorkout(workout *db.WorkoutWithExercises) {
	app.resizeDetailViewport()
	app.detailViewport.SetContent(workoutDetailView(workout, app.width))
	app.detailViewport.GotoTop()
}

func (app *App) updateRefineInput(message tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := message.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEsc:
			app.screen = screenAIPreview
			return app, nil
		case tea.KeyEnter:
			app.refinePrompt = strings.TrimSpace(app.input.Value())
			if app.refinePrompt == "" {
				app.modalError = "Describe a workout-related modification first."
				return app, nil
			}
			return app.startGeneration(true)
		}
	}
	var command tea.Cmd
	app.input, command = app.input.Update(message)
	return app, command
}

func (app *App) startGeneration(refine bool) (tea.Model, tea.Cmd) {
	if app.generator == nil {
		app.modalError = "AI generator is unavailable"
		return app, nil
	}
	if app.selectedModel == nil {
		app.modalError = "Select an AI model first"
		return app, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	app.aiCancel = cancel
	app.loading = true
	app.loadingText = "Generating workout with " + app.selectedModel.DisplayName + "…"
	if refine {
		app.loadingText = "Applying workout modification with " + app.selectedModel.DisplayName + "…"
	}

	selectedModel := *app.selectedModel
	preview := app.currentPreviewWorkout()
	prompt := app.refinePrompt
	history := app.currentWorkouts()
	var profileDescription string
	if app.activeProfile != nil {
		profileDescription = app.activeProfile.Description
	}
	command := func() tea.Msg {
		var result *ai.GenerationResult
		var err error
		if refine {
			result, err = app.generator.Refine(ctx, selectedModel, preview, prompt, profileDescription)
		} else {
			result, err = app.generator.Generate(ctx, selectedModel, history, profileDescription)
		}
		operation := aiGenerate
		if refine {
			operation = aiRefine
		}
		var workout *db.WorkoutWithExercises
		var profileUpdate string
		if result != nil {
			workout = result.Workout
			profileUpdate = result.ProfileUpdate
		}
		return aiResultMsg{workout: workout, profileUpdate: profileUpdate, err: err, operation: operation}
	}
	return app, tea.Batch(app.spinner.Tick, command)
}

func (app *App) startImport(notes, path string) (tea.Model, tea.Cmd) {
	if app.generator == nil {
		app.modalError = "AI generator is unavailable"
		return app, nil
	}
	if app.selectedModel == nil {
		app.modalError = "Select an AI model first"
		return app, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	app.aiCancel = cancel
	app.loading = true
	app.loadingText = "Importing " + filepath.Base(path) + " with " + app.selectedModel.DisplayName + "…"

	selectedModel := *app.selectedModel
	history := app.completedWorkouts()
	command := func() tea.Msg {
		workouts, err := app.generator.ImportNotes(ctx, selectedModel, notes, history)
		return aiResultMsg{workouts: workouts, err: err, operation: aiImport}
	}
	return app, tea.Batch(app.spinner.Tick, command)
}

func (app *App) updateConfirmation(message tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := message.(tea.KeyMsg)
	if !ok {
		return app, nil
	}
	if key.String() == "n" || key.Type == tea.KeyEsc {
		app.confirm = confirmNone
		return app, nil
	}
	if key.String() != "y" {
		return app, nil
	}

	var err error
	switch app.confirm {
	case confirmDeleteWorkout:
		if app.detailWorkout != nil {
			err = app.database.DeleteWorkoutByDate(app.detailWorkout.Workout.WorkoutDate)
			if err == nil {
				app.detailWorkout = nil
				app.screen = screenWorkouts
				app.statusMessage = "Workout deleted"
			}
		}
	case confirmDeleteProfile:
		_, err = app.database.DeleteProfile(app.profileInputTarget)
		if err == nil {
			app.screen = screenWorkouts
			app.statusMessage = "Profile and its workouts deleted"
		}
	}
	app.confirm = confirmNone
	if err != nil {
		app.modalError = err.Error()
		return app, nil
	}
	if err := app.refreshAll(); err != nil {
		app.modalError = err.Error()
	}
	return app, nil
}

func (app *App) refreshAll() error {
	active, err := app.database.GetActiveProfile()
	if err != nil {
		return err
	}
	app.activeProfile = active
	if err := app.refreshWorkouts(); err != nil && active != nil {
		return err
	}
	if err := app.refreshProfiles(); err != nil {
		return err
	}
	return app.refreshModels()
}

func (app *App) refreshWorkouts() error {
	if app.activeProfile == nil {
		app.workouts = nil
		app.workoutList.SetItems(nil)
		app.workoutList.SetShowStatusBar(false)
		return nil
	}
	workouts, err := app.database.GetAllWorkouts()
	if err != nil {
		return err
	}
	app.workouts = workouts
	app.applyWorkoutFilters()
	return nil
}

func (app *App) applyWorkoutFilters() {
	today := time.Now().Format("2006-01-02")
	visible := make([]db.WorkoutWithExercises, 0, len(app.workouts))
	for _, workout := range app.workouts {
		status := workout.Workout.Status
		if status != "planned" && status != "completed" {
			continue
		}
		if app.dateScope == workoutUpcoming && workout.Workout.WorkoutDate.Format("2006-01-02") < today {
			continue
		}
		if app.statusFilter == statusPlanned && status != "planned" {
			continue
		}
		if app.statusFilter == statusCompleted && status != "completed" {
			continue
		}
		visible = append(visible, workout)
	}

	if app.dateScope == workoutUpcoming {
		sort.SliceStable(visible, func(i, j int) bool {
			left := visible[i].Workout.WorkoutDate.Format("2006-01-02")
			right := visible[j].Workout.WorkoutDate.Format("2006-01-02")
			return left < right
		})
	}

	items := make([]list.Item, 0, len(visible))
	for _, workout := range visible {
		items = append(items, workoutItem{workout: workout})
	}
	searchText := app.workoutList.FilterInput.Value()
	searchActive := app.workoutList.FilterState() != list.Unfiltered
	app.workoutList.SetItems(items)
	app.workoutList.SetShowStatusBar(len(items) > 0)
	app.workoutList.ResetSelected()
	if searchActive {
		app.workoutList.SetFilterText(searchText)
	}
}

func (app *App) refreshProfiles() error {
	profiles, err := app.database.GetProfiles()
	if err != nil {
		return err
	}
	items := make([]list.Item, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, profileItem{profile: profile})
	}
	app.profileList.SetItems(items)
	return nil
}

func (app *App) refreshModels() error {
	models, err := app.database.GetAIModels()
	if err != nil {
		return err
	}
	items := make([]list.Item, 0, len(models))
	app.selectedModel = nil
	for _, model := range models {
		items = append(items, modelItem{model: model})
		if model.IsSelected {
			selected := model
			app.selectedModel = &selected
		}
	}
	app.modelList.SetItems(items)
	return nil
}

func (app *App) currentWorkouts() []db.WorkoutWithExercises {
	return append([]db.WorkoutWithExercises(nil), app.workouts...)
}

func (app *App) completedWorkouts() []db.WorkoutWithExercises {
	workouts := app.currentWorkouts()
	completed := make([]db.WorkoutWithExercises, 0, len(workouts))
	for _, workout := range workouts {
		if workout.Workout.Status == "completed" {
			completed = append(completed, workout)
		}
	}
	return completed
}

func (app *App) resize() {
	width := contentWidth(app.width)
	height := contentHeight(app.height)
	app.workoutList.SetSize(width, height)
	app.profileList.SetSize(width, height)
	app.modelList.SetSize(width, height)
	app.resizeDetailViewport()
	if app.screen == screenExerciseEditor {
		app.exerciseEditor.resize(app.width, app.height)
	}
	if app.screen == screenWorkoutDetail {
		app.detailViewport.SetContent(workoutDetailView(app.detailWorkout, app.width))
	}
	if app.screen == screenAIPreview {
		app.detailViewport.SetContent(workoutDetailView(app.currentPreviewWorkout(), app.width))
	}
}

func (app *App) resizeDetailViewport() {
	height := detailViewportHeight(app.height)
	if app.screen == screenAIPreview && app.previewOrigin == aiImport {
		height -= 2
		if height < 1 {
			height = 1
		}
	}
	app.detailViewport.Width = detailViewportWidth(app.width)
	app.detailViewport.Height = height
}

func detailViewportWidth(width int) int {
	width -= 4
	if width < 1 {
		return 1
	}
	if width > 98 {
		return 98
	}
	return width
}

func detailViewportHeight(height int) int {
	height -= 8
	if height < 1 {
		return 1
	}
	return height
}

func (app *App) View() string {
	if terminalTooSmall(app.width, app.height) {
		return fmt.Sprintf(
			"Resize terminal\nminimum %dx%d\ncurrent %dx%d\n\nq quit",
			minTerminalWidth,
			minTerminalHeight,
			app.width,
			app.height,
		)
	}
	if app.modalError != "" {
		return app.modal("Something went wrong", app.modalError, "enter/esc close")
	}
	if app.confirm != confirmNone {
		return app.modal("Please confirm", app.confirmText, "y delete  ·  n/esc cancel")
	}
	if app.statusFilterOpen {
		message := "1 / p  Planned\n2 / c  Completed\n0 / a  All statuses\n\nCurrent: " + app.workoutStatusLabel()
		return app.modal("Filter workouts by status", message, "choose a status  ·  esc cancel")
	}
	if app.loading {
		return app.frame(lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(app.spinner.View()+" "+app.loadingText),
			"",
			mutedStyle.Render("esc cancel  ·  ctrl+c quit"),
		))
	}

	var body string
	var help string
	switch app.screen {
	case screenWorkouts:
		dateToggleHelp := "a all dates"
		if app.dateScope == workoutAll {
			dateToggleHelp = "a upcoming"
		}
		body = mutedStyle.Render(app.workoutFilterSummary()) + "\n\n" + app.workoutList.View()
		help = "enter view  ·  n new  ·  i import  ·  d delete  ·  g generate  ·  " + dateToggleHelp + "  ·  f status  ·  p profiles  ·  m models  ·  / search  ·  q quit"
	case screenWorkoutDetail:
		body = app.detailViewport.View()
		help = "↑/↓ scroll  ·  d delete  ·  esc back"
	case screenProfiles:
		body = app.profileList.View()
		help = "enter set default  ·  c create  ·  e rename  ·  i describe  ·  d delete  ·  / filter  ·  esc back"
	case screenModels:
		body = app.modelList.View()
		help = "enter select  ·  / filter  ·  esc back"
	case screenWorkoutForm:
		body = app.form.view(app.width)
		help = "tab next  ·  ctrl+a add exercise  ·  ctrl+s save  ·  esc cancel"
	case screenProfileInput:
		title := "Create profile"
		switch app.profileInputAction {
		case profileRename:
			title = "Rename profile"
		case profileDescribe:
			title = "Edit profile description"
		}
		if app.onboarding {
			title = "Welcome — create your first profile"
		}
		body = panelStyle.Width(contentWidth(app.width)).Render(sectionStyle.Render(title) + "\n\n" + app.input.View())
		help = "enter save  ·  esc cancel"
	case screenImportPath:
		descriptionWidth := contentWidth(app.width) - 4
		body = panelStyle.Width(contentWidth(app.width)).Render(
			sectionStyle.Render("Import workout notes") + "\n\n" +
				app.input.View() + "\n\n" +
				mutedStyle.Width(descriptionWidth).Render("Ukrainian and free-form text are supported. The selected AI model will translate and enrich the exercises."),
		)
		help = "enter analyze  ·  esc cancel"
	case screenAIPreview:
		body = app.detailViewport.View()
		if app.pendingProfileUpdate != "" {
			width := contentWidth(app.width)
			callout := successStyle.Bold(true).Render("Profile update suggested") + "\n" +
				mutedStyle.Width(width).Render(app.pendingProfileUpdate) + "\n" +
				mutedStyle.Render("Saved together with this workout (s)")
			body = callout + "\n\n" + body
		}
		if app.previewOrigin == aiImport {
			counter := fmt.Sprintf("Import preview  ·  workout %d of %d", app.previewIndex+1, len(app.previewWorkouts))
			body = titleStyle.Render(counter) + "\n\n" + body
		}
		help = "↑/↓ scroll  ·  e edit table  ·  a AI modify  ·  s save planned  ·  esc discard"
		if len(app.previewWorkouts) > 1 {
			help = "←/→ browse  ·  " + help
		}
	case screenExerciseEditor:
		body = app.exerciseEditor.view(app.width)
		help = "↑/↓ row  ·  ←/→ or tab column  ·  enter edit  ·  ctrl+s apply to preview  ·  esc cancel edits"
		if app.exerciseEditor.editing {
			help = "enter accept cell  ·  tab accept / next column  ·  ctrl+s apply to preview  ·  esc cancel cell"
		}
	case screenRefineInput:
		body = panelStyle.Width(contentWidth(app.width)).Render(sectionStyle.Render("Modify exercises") + "\n\n" + app.input.View() + "\n\n" + mutedStyle.Render("Only exercise selection and exercise parameters are accepted."))
		help = "enter apply  ·  esc back"
	}

	if app.statusMessage != "" && app.screen == screenWorkouts {
		body = successStyle.Render("✓ "+app.statusMessage) + "\n\n" + body
	}
	return app.frame(body + "\n" + mutedStyle.Width(contentWidth(app.width)).Render(help))
}

func (app *App) workoutFilterSummary() string {
	dateLabel := "Upcoming"
	if app.dateScope == workoutAll {
		dateLabel = "All dates"
	}
	return dateLabel + "  ·  " + app.workoutStatusLabel()
}

func (app *App) workoutStatusLabel() string {
	switch app.statusFilter {
	case statusPlanned:
		return "Planned only"
	case statusCompleted:
		return "Completed only"
	default:
		return "All statuses"
	}
}

func (app *App) frame(body string) string {
	profile := "No profile"
	if app.activeProfile != nil {
		profile = "@" + app.activeProfile.Name
	}
	model := "No AI model"
	if app.selectedModel != nil {
		model = app.selectedModel.DisplayName
	}
	header := titleStyle.Render("Terminal Fit Recorder") + "  " + headerMetaStyle.Render(profile+"  ·  "+model)
	return lipgloss.NewStyle().Padding(1, 2).Render(header + "\n\n" + body)
}

func (app *App) modal(title, message, help string) string {
	width := contentWidth(app.width)
	if width > 72 {
		width = 72
	}
	content := titleStyle.Render(title) + "\n\n" + lipgloss.NewStyle().Width(width-4).Render(message) + "\n\n" + mutedStyle.Render(help)
	return lipgloss.NewStyle().Padding(2, 4).Render(modalStyle.Width(width).Render(content))
}

func workoutDetailView(workout *db.WorkoutWithExercises, width int) string {
	if workout == nil {
		return panelStyle.Width(contentWidth(width)).Render(mutedStyle.Render("No workout selected"))
	}
	status := "Done"
	if workout.Workout.Status == "planned" {
		status = "AI planned"
	}
	rows := []string{
		sectionStyle.Render(displayWorkoutType(workout.Workout.WorkoutType) + " workout"),
		mutedStyle.Render(workout.Workout.WorkoutDate.Format("Monday, 02 January 2006") + "  ·  " + status),
		"",
	}
	if notes := strings.TrimSpace(workout.Workout.Notes); notes != "" {
		rows = append(rows,
			successStyle.Bold(true).Render("Notes"),
			mutedStyle.Width(contentWidth(width)).Render(notes),
			"",
		)
	}
	for index, exercise := range workout.Exercises {
		rows = append(rows,
			fmt.Sprintf("%s  %s", titleStyle.Render(fmt.Sprintf("%02d", index+1)), sectionStyle.Render(exercise.Name)),
			"    "+mutedStyle.Render(exerciseSummary(exercise)),
		)
		if exercise.YoutubeURL != "" {
			rows = append(rows, "    "+mutedStyle.Render("▶ "+exercise.YoutubeURL))
		}
		if index < len(workout.Exercises)-1 {
			rows = append(rows, "")
		}
	}
	return panelStyle.Width(contentWidth(width)).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func displayWorkoutType(workoutType string) string {
	switch strings.ToLower(strings.TrimSpace(workoutType)) {
	case "strength":
		return "Strength"
	case "cardio":
		return "Cardio"
	default:
		return workoutType
	}
}

func exerciseSummary(exercise db.Exercise) string {
	parts := make([]string, 0, 4)
	if exercise.Weight > 0 {
		parts = append(parts, fmt.Sprintf("%d kg", exercise.Weight))
	}
	if exercise.Repetitions > 0 || exercise.Sets > 0 {
		parts = append(parts, fmt.Sprintf("%d reps × %d sets", exercise.Repetitions, exercise.Sets))
	}
	if exercise.Duration > 0 {
		parts = append(parts, fmt.Sprintf("%.1f min", exercise.Duration))
	}
	if exercise.Distance > 0 {
		parts = append(parts, fmt.Sprintf("%d m", exercise.Distance))
	}
	if len(parts) == 0 {
		return "No metrics"
	}
	return strings.Join(parts, "  ·  ")
}

package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminal_fit_recorder/internal/db"
)

const (
	formFocusType = iota
	formFocusDate
	formFocusName
	formFocusWeight
	formFocusReps
	formFocusSets
	formFocusDuration
	formFocusDistance
	formFocusAdd
	formFocusSave
	formFocusCount
)

type formOutcome int

const (
	formContinue formOutcome = iota
	formCancelled
	formSave
)

type workoutForm struct {
	workoutType int
	focus       int
	inputs      []textinput.Model
	exercises   []db.Exercise
	err         string
}

func newWorkoutForm(now time.Time) workoutForm {
	placeholders := []string{
		now.Format("2006-01-02"),
		"Bench Press",
		"0",
		"0",
		"0",
		"0",
		"0",
	}
	inputs := make([]textinput.Model, len(placeholders))
	for index, placeholder := range placeholders {
		input := textinput.New()
		input.Placeholder = placeholder
		input.CharLimit = 100
		input.Width = 32
		inputs[index] = input
	}
	inputs[0].SetValue(now.Format("2006-01-02"))

	form := workoutForm{inputs: inputs}
	form.setFocus(formFocusType)
	return form
}

func (form *workoutForm) setFocus(focus int) tea.Cmd {
	if focus < 0 {
		focus = formFocusCount - 1
	}
	if focus >= formFocusCount {
		focus = 0
	}
	form.focus = focus

	var command tea.Cmd
	for index := range form.inputs {
		if index+1 == focus {
			command = form.inputs[index].Focus()
		} else {
			form.inputs[index].Blur()
		}
	}
	return command
}

func (form *workoutForm) update(message tea.KeyMsg) (formOutcome, *db.WorkoutWithExercises, tea.Cmd) {
	form.err = ""

	switch message.Type {
	case tea.KeyEsc:
		return formCancelled, nil, nil
	case tea.KeyCtrlS:
		workout, err := form.workout()
		if err != nil {
			form.err = err.Error()
			return formContinue, nil, nil
		}
		return formSave, workout, nil
	case tea.KeyCtrlA:
		if err := form.addExercise(); err != nil {
			form.err = err.Error()
			return formContinue, nil, nil
		}
		return formContinue, nil, form.setFocus(formFocusName)
	case tea.KeyTab:
		return formContinue, nil, form.setFocus(form.focus + 1)
	case tea.KeyShiftTab:
		return formContinue, nil, form.setFocus(form.focus - 1)
	case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
		if form.focus == formFocusType {
			form.workoutType = 1 - form.workoutType
			return formContinue, nil, nil
		}
	case tea.KeyEnter:
		switch form.focus {
		case formFocusAdd:
			if err := form.addExercise(); err != nil {
				form.err = err.Error()
				return formContinue, nil, nil
			}
			return formContinue, nil, form.setFocus(formFocusName)
		case formFocusSave:
			workout, err := form.workout()
			if err != nil {
				form.err = err.Error()
				return formContinue, nil, nil
			}
			return formSave, workout, nil
		default:
			return formContinue, nil, form.setFocus(form.focus + 1)
		}
	}

	if form.focus >= formFocusDate && form.focus <= formFocusDistance {
		inputIndex := form.focus - 1
		var command tea.Cmd
		form.inputs[inputIndex], command = form.inputs[inputIndex].Update(message)
		return formContinue, nil, command
	}
	return formContinue, nil, nil
}

func (form *workoutForm) addExercise() error {
	name := strings.TrimSpace(form.inputs[1].Value())
	if name == "" {
		return fmt.Errorf("exercise name is required")
	}

	weight, err := parseOptionalInt(form.inputs[2].Value(), "weight")
	if err != nil {
		return err
	}
	reps, err := parseOptionalInt(form.inputs[3].Value(), "repetitions")
	if err != nil {
		return err
	}
	sets, err := parseOptionalInt(form.inputs[4].Value(), "sets")
	if err != nil {
		return err
	}
	duration, err := parseOptionalFloat(form.inputs[5].Value(), "duration")
	if err != nil {
		return err
	}
	distance, err := parseOptionalInt(form.inputs[6].Value(), "distance")
	if err != nil {
		return err
	}

	form.exercises = append(form.exercises, db.Exercise{
		Name:        name,
		Weight:      weight,
		Repetitions: reps,
		Sets:        sets,
		Duration:    duration,
		Distance:    distance,
	})
	for index := 1; index < len(form.inputs); index++ {
		form.inputs[index].SetValue("")
	}
	return nil
}

func (form *workoutForm) workout() (*db.WorkoutWithExercises, error) {
	if strings.TrimSpace(form.inputs[1].Value()) != "" {
		if err := form.addExercise(); err != nil {
			return nil, err
		}
	}
	if len(form.exercises) == 0 {
		return nil, fmt.Errorf("add at least one exercise")
	}

	workoutDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(form.inputs[0].Value()), time.Local)
	if err != nil {
		return nil, fmt.Errorf("date must use YYYY-MM-DD")
	}
	workoutType := "strength"
	if form.workoutType == 1 {
		workoutType = "cardio"
	}

	return &db.WorkoutWithExercises{
		Workout: db.Workout{
			WorkoutType: workoutType,
			WorkoutDate: workoutDate,
			Status:      "completed",
		},
		Exercises: append([]db.Exercise(nil), form.exercises...),
	}, nil
}

func parseOptionalInt(value, field string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative whole number", field)
	}
	return parsed, nil
}

func parseOptionalFloat(value, field string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative number", field)
	}
	return parsed, nil
}

func (form workoutForm) view(width int) string {
	types := []string{"Strength", "Cardio"}
	typeParts := make([]string, len(types))
	for index, workoutType := range types {
		style := choiceStyle
		if form.workoutType == index {
			style = selectedChoiceStyle
		}
		if form.focus == formFocusType && form.workoutType == index {
			style = focusedChoiceStyle
		}
		typeParts[index] = style.Render(workoutType)
	}

	labels := []string{"Date", "Exercise", "Weight (kg)", "Repetitions", "Sets", "Duration (min)", "Distance (m)"}
	rows := []string{fieldLabelStyle.Render("Workout type") + "  " + lipgloss.JoinHorizontal(lipgloss.Left, typeParts...)}
	for index, label := range labels {
		labelStyle := fieldLabelStyle
		if form.focus == index+1 {
			labelStyle = focusedLabelStyle
		}
		rows = append(rows, labelStyle.Render(label)+"  "+form.inputs[index].View())
	}

	addStyle := buttonStyle
	saveStyle := buttonStyle
	if form.focus == formFocusAdd {
		addStyle = focusedButtonStyle
	}
	if form.focus == formFocusSave {
		saveStyle = focusedButtonStyle
	}
	rows = append(rows, "", lipgloss.JoinHorizontal(lipgloss.Left,
		addStyle.Render("Add exercise"), "  ", saveStyle.Render("Save workout"),
	))

	if len(form.exercises) > 0 {
		rows = append(rows, "", sectionStyle.Render(fmt.Sprintf("Exercises · %d", len(form.exercises))))
		for index, exercise := range form.exercises {
			rows = append(rows, mutedStyle.Render(fmt.Sprintf("%d. %s  %s", index+1, exercise.Name, exerciseSummary(exercise))))
		}
	}
	if form.err != "" {
		rows = append(rows, "", errorStyle.Render(form.err))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return panelStyle.Width(contentWidth(width)).Render(content)
}

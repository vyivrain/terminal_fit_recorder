package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminal_fit_recorder/internal/db"
)

var exerciseColumnLabels = []string{"Exercise name", "Weight (kg)", "Repetitions", "Sets", "Duration (min)", "Distance (m)"}

// The editor owns a copy so cancelling cannot change the pending import.
type exerciseEditor struct {
	workout db.WorkoutWithExercises
	table   table.Model
	column  int
	input   textinput.Model
	editing bool
	err     string
}

func newExerciseEditor(workout *db.WorkoutWithExercises, width, height int) exerciseEditor {
	editor := exerciseEditor{workout: *workout, input: textinput.New()}
	editor.workout.Exercises = append([]db.Exercise(nil), workout.Exercises...)
	editor.input.CharLimit = 100
	editor.table = table.New(table.WithFocused(true))
	styles := table.DefaultStyles()
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("0")).Background(accentColor)
	editor.table.SetStyles(styles)
	editor.resize(width, height)
	editor.refreshRows()
	return editor
}

func (editor *exerciseEditor) resize(width, height int) {
	width = contentWidth(width)
	editor.table.SetColumns([]table.Column{
		{Title: "Exercise", Width: max(1, width-40)},
		{Title: "kg", Width: 5},
		{Title: "Reps", Width: 4},
		{Title: "Sets", Width: 4},
		{Title: "Min", Width: 7},
		{Title: "Metres", Width: 8},
	})
	editor.table.SetWidth(width)
	editor.table.SetHeight(max(3, height-14))
	editor.input.Width = max(1, width-2)
}

func exerciseValues(exercise db.Exercise) table.Row {
	return table.Row{
		exercise.Name, strconv.Itoa(exercise.Weight), strconv.Itoa(exercise.Repetitions),
		strconv.Itoa(exercise.Sets), strconv.FormatFloat(exercise.Duration, 'f', -1, 64), strconv.Itoa(exercise.Distance),
	}
}

func (editor *exerciseEditor) refreshRows() {
	rows := make([]table.Row, len(editor.workout.Exercises))
	for index, exercise := range editor.workout.Exercises {
		rows[index] = exerciseValues(exercise)
	}
	editor.table.SetRows(rows)
}

func (editor *exerciseEditor) commitCell() error {
	value := strings.TrimSpace(editor.input.Value())
	exercise := &editor.workout.Exercises[editor.table.Cursor()]
	label := exerciseColumnLabels[editor.column]
	switch editor.column {
	case 0:
		if value == "" || len([]rune(value)) > 100 {
			return fmt.Errorf("exercise name must contain 1–100 characters")
		}
		exercise.Name = value
	case 4:
		duration, err := parseOptionalFloat(value, label)
		if err != nil {
			return err
		}
		if math.IsNaN(duration) || math.IsInf(duration, 0) || duration > 1440 {
			return fmt.Errorf("duration must be a finite number between 0 and 1440 minutes")
		}
		exercise.Duration = duration
	default:
		number, err := parseOptionalInt(value, label)
		if err != nil {
			return err
		}
		limits := []int{0, 1000, 1000, 100, 0, 1_000_000}
		if number > limits[editor.column] {
			return fmt.Errorf("%s must be at most %d", label, limits[editor.column])
		}
		switch editor.column {
		case 1:
			exercise.Weight = number
		case 2:
			exercise.Repetitions = number
		case 3:
			exercise.Sets = number
		case 5:
			exercise.Distance = number
		}
	}
	editor.editing = false
	editor.input.Blur()
	editor.err = ""
	editor.refreshRows()
	return nil
}

func (editor *exerciseEditor) update(message tea.Msg) (formOutcome, tea.Cmd) {
	if key, ok := message.(tea.KeyMsg); ok {
		if editor.editing {
			switch key.Type {
			case tea.KeyEsc:
				editor.editing = false
				editor.err = ""
				editor.input.Blur()
				return formContinue, nil
			case tea.KeyEnter, tea.KeyTab, tea.KeyShiftTab, tea.KeyCtrlS:
				if err := editor.commitCell(); err != nil {
					editor.err = err.Error()
					return formContinue, nil
				}
				if key.Type == tea.KeyCtrlS {
					return formSave, nil
				}
				if key.Type == tea.KeyTab {
					editor.column = (editor.column + 1) % len(exerciseColumnLabels)
				} else if key.Type == tea.KeyShiftTab {
					editor.column = (editor.column + len(exerciseColumnLabels) - 1) % len(exerciseColumnLabels)
				}
				return formContinue, nil
			}
		} else {
			switch key.Type {
			case tea.KeyEsc:
				return formCancelled, nil
			case tea.KeyCtrlS:
				return formSave, nil
			case tea.KeyTab, tea.KeyRight:
				editor.column = (editor.column + 1) % len(exerciseColumnLabels)
				return formContinue, nil
			case tea.KeyShiftTab, tea.KeyLeft:
				editor.column = (editor.column + len(exerciseColumnLabels) - 1) % len(exerciseColumnLabels)
				return formContinue, nil
			case tea.KeyEnter:
				if len(editor.workout.Exercises) > 0 {
					editor.input.SetValue(exerciseValues(editor.workout.Exercises[editor.table.Cursor()])[editor.column])
					editor.input.CursorEnd()
					editor.editing = true
					return formContinue, editor.input.Focus()
				}
			}
		}
	}
	var command tea.Cmd
	if editor.editing {
		editor.input, command = editor.input.Update(message)
	} else {
		editor.table, command = editor.table.Update(message)
	}
	return formContinue, command
}

func (editor exerciseEditor) view(width int) string {
	title := fmt.Sprintf("Edit exercises · %s · %s", editor.workout.Workout.WorkoutDate.Format("2006-01-02"), editor.workout.Workout.WorkoutType)
	value := ""
	if len(editor.workout.Exercises) > 0 {
		value = exerciseValues(editor.workout.Exercises[editor.table.Cursor()])[editor.column]
	}
	field := mutedStyle.Render("Enter to edit: ") + value
	if editor.editing {
		field = editor.input.View()
	}
	body := titleStyle.Render(title) + "\n\n" + editor.table.View() + "\n" +
		sectionStyle.Render(fmt.Sprintf("Row %d/%d · %s", editor.table.Cursor()+1, len(editor.workout.Exercises), exerciseColumnLabels[editor.column])) + "\n" +
		lipgloss.NewStyle().Width(contentWidth(width)).MaxHeight(2).Render(field)
	if editor.err != "" {
		body += "\n" + errorStyle.Width(contentWidth(width)).Render(editor.err)
	}
	return body
}

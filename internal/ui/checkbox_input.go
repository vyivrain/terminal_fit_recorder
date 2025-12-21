package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// CheckboxModel for selecting one option with visual checkboxes
type CheckboxModel struct {
	prompt    string
	choices   []string
	cursor    int
	selected  string
	cancelled bool
}

func NewCheckboxModel(prompt string, choices []string) CheckboxModel {
	return CheckboxModel{
		prompt:  prompt,
		choices: choices,
		cursor:  0,
	}
}

func (m CheckboxModel) Init() tea.Cmd {
	return nil
}

func (m CheckboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.cancelled = true
			return m, tea.Quit

		case tea.KeyUp, tea.KeyShiftTab:
			if m.cursor > 0 {
				m.cursor--
			}

		case tea.KeyDown, tea.KeyTab:
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case tea.KeyEnter, tea.KeySpace:
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m CheckboxModel) View() string {
	s := m.prompt + "\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if m.cursor == i {
			checked = "x"
		}

		s += cursor + " [" + checked + "] " + choice + "\n"
	}

	s += "\n(↑/↓ to move, enter/space to select, ctrl+c/ctrl+d to cancel)\n"

	return s
}

func GetCheckboxSelection(prompt string, choices []string) (string, bool) {
	m := NewCheckboxModel(prompt, choices)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", true
	}

	result := finalModel.(CheckboxModel)
	if result.cancelled {
		return "", true
	}

	return result.selected, false
}

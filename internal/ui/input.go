package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type InputType int

const (
	InputTypeText InputType = iota
	InputTypeList
	InputTypeCheckbox
	InputTypeAutocomplete
)

func GetInput(prompt string, suggestions []string) (string, bool) {
	return GetInputWithType(prompt, suggestions, InputTypeText)
}

func GetInputWithType(prompt string, suggestions []string, inputType InputType) (string, bool) {
	switch inputType {
	case InputTypeCheckbox:
		return GetCheckboxSelection(prompt, suggestions)
	case InputTypeList:
		if len(suggestions) > 0 {
			return GetListSelection(prompt, suggestions)
		}
		fallthrough
	case InputTypeAutocomplete:
		// TextText input with autocomplete dropdown
		m := NewInput(prompt, suggestions)
		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return "", true
		}

		result := finalModel.(InputModel)
		if result.cancelled {
			return "", true
		}

		return strings.TrimSpace(result.value), false
	default:
		// Regular text input (no suggestions)
		m := NewInput(prompt, nil)
		p := tea.NewProgram(m)

		finalModel, err := p.Run()
		if err != nil {
			return "", true
		}

		result := finalModel.(InputModel)
		if result.cancelled {
			return "", true
		}

		return strings.TrimSpace(result.value), false
	}
}

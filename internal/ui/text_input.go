package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type InputModel struct {
	textInput     textinput.Model
	suggestions   []string
	filteredSuggs []string
	selectedIdx   int
	showDropdown  bool
	value         string
	cancelled     bool
	submitted     bool
}

func NewInput(prompt string, suggestions []string) InputModel {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = prompt

	return InputModel{
		textInput:   ti,
		suggestions: suggestions,
	}
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyCtrlD:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEnter:
			// If dropdown is showing and item is selected, use that
			if m.showDropdown && len(m.filteredSuggs) > 0 && m.selectedIdx < len(m.filteredSuggs) {
				m.value = m.filteredSuggs[m.selectedIdx]
			} else {
				m.value = m.textInput.Value()
			}
			m.submitted = true
			return m, tea.Quit
		case tea.KeyDown:
			if m.showDropdown && len(m.filteredSuggs) > 0 {
				m.selectedIdx++
				if m.selectedIdx >= len(m.filteredSuggs) {
					m.selectedIdx = 0
				}
				return m, nil
			}
		case tea.KeyUp:
			if m.showDropdown && len(m.filteredSuggs) > 0 {
				m.selectedIdx--
				if m.selectedIdx < 0 {
					m.selectedIdx = len(m.filteredSuggs) - 1
				}
				return m, nil
			}
		case tea.KeyTab:
			// Autocomplete on Tab with first suggestion
			if len(m.filteredSuggs) > 0 {
				m.textInput.SetValue(m.filteredSuggs[m.selectedIdx])
				m.textInput.CursorEnd()
				m.showDropdown = false
				m.selectedIdx = 0
				return m, nil
			}
		case tea.KeyEsc:
			m.showDropdown = false
			m.selectedIdx = 0
			return m, nil
		}
	}

	// Update text input
	oldValue := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	newValue := m.textInput.Value()

	// Update filtered suggestions when text changes
	if oldValue != newValue {
		m.updateFilteredSuggestions()
	}

	return m, cmd
}

func (m *InputModel) updateFilteredSuggestions() {
	currentValue := strings.ToLower(m.textInput.Value())

	if currentValue == "" || len(m.suggestions) == 0 {
		m.showDropdown = false
		m.filteredSuggs = nil
		m.selectedIdx = 0
		return
	}

	// Store old filtered suggestions to detect changes
	oldFiltered := m.filteredSuggs
	m.filteredSuggs = nil

	for _, suggestion := range m.suggestions {
		if strings.Contains(strings.ToLower(suggestion), currentValue) {
			m.filteredSuggs = append(m.filteredSuggs, suggestion)
		}
	}

	m.showDropdown = len(m.filteredSuggs) > 0

	// Only reset selected index if the filtered list actually changed
	if len(oldFiltered) != len(m.filteredSuggs) {
		m.selectedIdx = 0
	} else if m.selectedIdx >= len(m.filteredSuggs) {
		// Ensure selectedIdx is within bounds
		m.selectedIdx = 0
	}
}

func (m InputModel) View() string {
	s := m.textInput.View()

	// Show dropdown if active
	if m.showDropdown && len(m.filteredSuggs) > 0 {
		s += "\n"
		maxItems := 10
		if len(m.filteredSuggs) < maxItems {
			maxItems = len(m.filteredSuggs)
		}

		for i := 0; i < maxItems; i++ {
			cursor := "  "
			if i == m.selectedIdx {
				cursor = "> "
			}
			s += "\n" + cursor + m.filteredSuggs[i]
		}

		if len(m.filteredSuggs) > maxItems {
			s += "\n  ..."
		}

		s += "\n\n(↑/↓ to navigate, Tab to complete, Enter to select, Esc to close)"
	}

	return s + "\n"
}

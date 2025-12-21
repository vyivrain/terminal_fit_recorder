package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// List item for the selector
type item string

func (i item) FilterValue() string { return string(i) }
func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }

// ListModel for selecting from suggestions
type ListModel struct {
	list      list.Model
	choice    string
	cancelled bool
	quitting  bool
}

func NewListModel(prompt string, suggestions []string) ListModel {
	items := make([]list.Item, len(suggestions))
	for i, s := range suggestions {
		items[i] = item(s)
	}

	// Add option for custom input
	items = append(items, item("(Type custom value)"))

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = prompt
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)

	return ListModel{
		list: l,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := msg.Width, msg.Height
		m.list.SetSize(h, v-2)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)

				// If custom value selected, prompt for text input
				if m.choice == "(Type custom value)" {
					return m, tea.Quit
				}
			}
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ListModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

func GetListSelection(prompt string, suggestions []string) (string, bool) {
	m := NewListModel(prompt, suggestions)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return "", true
	}

	result := finalModel.(ListModel)
	if result.cancelled {
		return "", true
	}

	// If custom value selected, prompt for text input
	if result.choice == "(Type custom value)" {
		return GetInput(prompt, nil)
	}

	return result.choice, false
}

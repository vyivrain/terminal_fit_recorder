package tui

import "github.com/charmbracelet/lipgloss"

const (
	minTerminalWidth  = 60
	minTerminalHeight = 18
)

var (
	accentColor = lipgloss.Color("39")
	greenColor  = lipgloss.Color("42")
	mutedColor  = lipgloss.Color("245")
	errorColor  = lipgloss.Color("196")

	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	headerMetaStyle = lipgloss.NewStyle().Foreground(mutedColor)
	sectionStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	mutedStyle      = lipgloss.NewStyle().Foreground(mutedColor)
	errorStyle      = lipgloss.NewStyle().Foreground(errorColor)
	successStyle    = lipgloss.NewStyle().Foreground(greenColor)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(1, 2)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(accentColor).
			Padding(1, 2)

	choiceStyle         = lipgloss.NewStyle().Foreground(mutedColor).Padding(0, 1)
	selectedChoiceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Padding(0, 1)
	focusedChoiceStyle  = lipgloss.NewStyle().Bold(true).Foreground(accentColor).Background(lipgloss.Color("236")).Padding(0, 1)

	fieldLabelStyle   = lipgloss.NewStyle().Foreground(mutedColor).Width(16)
	focusedLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(accentColor).Width(16)

	buttonStyle        = lipgloss.NewStyle().Foreground(mutedColor).Border(lipgloss.RoundedBorder()).Padding(0, 2)
	focusedButtonStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(accentColor).Padding(1, 3)
)

func contentWidth(width int) int {
	width -= 8
	if width < 1 {
		return 1
	}
	if width > 92 {
		return 92
	}
	return width
}

func contentHeight(height int) int {
	height -= 8
	if height < 1 {
		return 1
	}
	return height
}

func terminalTooSmall(width, height int) bool {
	return width < minTerminalWidth || height < minTerminalHeight
}

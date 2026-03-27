package state

import (
	"github.com/charmbracelet/lipgloss"
)

// statusBar renders the bottom status information bar.
type statusBar struct {
	message string
	width   int
}

func newStatusBar() statusBar {
	return statusBar{
		message: "Ready. Press Tab to switch panes, / to enter a command.",
	}
}

func (s *statusBar) SetSize(w int) {
	s.width = w
}

func (s *statusBar) SetMessage(msg string) {
	s.message = msg
}

func (s *statusBar) View() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1).
		Width(s.width)

	return style.Render(s.message)
}

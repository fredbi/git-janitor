package statusbar

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// TODO: colorize status message when errors (e.g. red) or warning (e.g. orange)

// TickMsg is sent to animate the progress bar.
type TickMsg struct{}

// StatusBar renders either a status message or an animated progress bar.
type StatusBar struct {
	Message  string
	Width    int
	progress progress.Model
	active   bool    // true when showing the progress bar
	percent  float64 // current progress (0.0 – 1.0)
	label    string  // label shown next to the progress bar
}

// New creates a StatusBar with a default ready message.
func New() StatusBar {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	return StatusBar{
		Message:  "Ready. Press Tab to switch panes, / to enter a command.",
		progress: p,
	}
}

// SetSize updates the width of the status bar.
func (s *StatusBar) SetSize(w int) {
	const barPadding = 4

	s.Width = w
	s.progress.Width = w - barPadding // padding
}

// SetMessage updates the displayed message and hides the progress bar.
func (s *StatusBar) SetMessage(msg string) {
	s.Message = msg
	s.active = false
	s.percent = 0
}

func (s *StatusBar) SetMessagef(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	s.SetMessage(formatted)
}

// StartProgress shows the progress bar with the given label.
// Returns a tea.Cmd that kicks off the animation ticks.
func (s *StatusBar) StartProgress(label string) tea.Cmd {
	s.active = true
	s.percent = 0
	s.label = label

	return s.tick()
}

// Update handles progress animation messages.
// Returns true if the message was consumed.
func (s *StatusBar) Update(msg tea.Msg) (tea.Cmd, bool) {
	const (
		maxAsymptotic = 0.95
		progressRate  = 0.08
	)
	if !s.active {
		return nil, false
	}

	switch msg.(type) {
	case TickMsg:
		// Advance the progress bar. It never reaches 1.0 —
		// it slows down asymptotically until StopProgress is called.
		s.percent += (1.0 - s.percent) * progressRate
		s.percent = min(maxAsymptotic, s.percent)

		return s.tick(), true

	case progress.FrameMsg:
		var cmd tea.Cmd
		m, c := s.progress.Update(msg)
		var ok bool
		s.progress, ok = m.(progress.Model)
		if !ok {
			return nil, false
		}

		cmd = c

		return cmd, true
	}

	return nil, false
}

// View renders the status bar or progress bar.
func (s *StatusBar) View() string {
	t := uxtypes.CurrentTheme
	if t == nil {
		return ""
	}

	if s.active {
		return s.viewProgress(t)
	}

	style := lipgloss.NewStyle().
		Foreground(t.StatusFg).
		Background(t.StatusBg).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1).
		Width(s.Width)

	return style.Render(s.Message)
}

func (s *StatusBar) viewProgress(t *uxtypes.Theme) string {
	// Style the progress bar colors to match the theme.
	s.progress.FullColor = string(t.Accent)
	s.progress.EmptyColor = string(t.Dim)

	bar := s.progress.ViewAs(s.percent)

	label := lipgloss.NewStyle().
		Foreground(t.StatusFg).
		Background(t.StatusBg).
		Bold(true).
		PaddingLeft(1).
		Render(s.label)

	row := lipgloss.NewStyle().
		Background(t.StatusBg).
		Width(s.Width).
		Render(fmt.Sprintf("%s %s", label, bar))

	return row
}

func (s *StatusBar) tick() tea.Cmd {
	const eightyMs = 80_000_000
	return tea.Tick(
		eightyMs,
		func(_ time.Time) tea.Msg { return TickMsg{} },
	)
}

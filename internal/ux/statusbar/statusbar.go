package statusbar

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

const ellipsis = " \u20DB" // combining three dots above — used as "more" indicator

// TickMsg is sent to animate the progress bar.
type TickMsg struct{}

// StatusBar renders a status message and an optional progress bar.
// Always occupies exactly 2 lines to prevent layout shift.
type StatusBar struct {
	Theme    *uxtypes.Theme
	Message  string // full message (may be multi-line)
	Width    int
	progress progress.Model
	active   bool    // true when showing the progress bar
	percent  float64 // current progress (0.0 – 1.0)
	label    string  // label shown next to the progress bar

	truncated bool // true when Message was too long to display in full
}

// New creates a StatusBar with a default ready message.
func New(theme *uxtypes.Theme) StatusBar {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	return StatusBar{
		Theme:    theme,
		Message:  "Ready. Press Tab to switch panes, / to enter a command.",
		progress: p,
	}
}

// SetSize updates the width of the status bar.
func (s *StatusBar) SetSize(w int) {
	s.Width = w
}

// SetMessage updates the displayed message and hides the progress bar.
func (s *StatusBar) SetMessage(msg string) {
	s.Message = msg
	s.active = false
	s.percent = 0
}

// SetMessagef is a formatted version of SetMessage.
func (s *StatusBar) SetMessagef(msg string, args ...any) {
	s.SetMessage(fmt.Sprintf(msg, args...))
}

// IsTruncated reports whether the current message is too long for the status bar.
func (s *StatusBar) IsTruncated() bool {
	return s.truncated
}

// FullMessage returns the complete untruncated message.
func (s *StatusBar) FullMessage() string {
	return s.Message
}

// StartProgress shows the progress bar with the given label.
func (s *StatusBar) StartProgress(label string) tea.Cmd {
	s.active = true
	s.percent = 0
	s.label = label

	return s.tick()
}

// Update handles progress animation messages.
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
		s.percent += (1.0 - s.percent) * progressRate
		s.percent = min(maxAsymptotic, s.percent)

		return s.tick(), true

	case progress.FrameMsg:
		m, c := s.progress.Update(msg)

		var ok bool
		if s.progress, ok = m.(progress.Model); !ok {
			return nil, false
		}

		return c, true
	}

	return nil, false
}

// View renders the status bar as exactly 2 lines.
func (s *StatusBar) View() string {
	t := s.Theme

	msgStyle := lipgloss.NewStyle().
		Foreground(t.StatusFg).
		Background(t.StatusBg).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1).
		Width(s.Width)

	emptyLine := lipgloss.NewStyle().
		Background(t.StatusBg).
		Width(s.Width).
		Render("")

	// Elide the message to fit 2 lines (when progress is inactive)
	// or 1 line (when progress is active and takes the second line).
	maxMsgLines := 2 //nolint:mnd // status bar height
	if s.active {
		maxMsgLines = 1
	}

	displayed, truncated := elideMessage(s.Message, s.Width-2, maxMsgLines) //nolint:mnd // padding
	s.truncated = truncated

	// Line 1 (and possibly line 2): message.
	var msgLines []string

	for _, line := range strings.Split(displayed, "\n") {
		msgLines = append(msgLines, msgStyle.Render(line))
	}

	// If truncated, append the ellipsis indicator to the last message line.
	if truncated {
		last := len(msgLines) - 1
		indicator := lipgloss.NewStyle().
			Foreground(t.Warning).
			Background(t.StatusBg).
			Bold(true).
			Render(ellipsis + " (Ctrl+D)")

		msgLines[last] = msgStyle.Render(displayed) + indicator
		// Re-render as single line with indicator appended
		msgLines = []string{msgStyle.Render(strings.Split(displayed, "\n")[0]) + indicator}

		if maxMsgLines > 1 && len(strings.Split(displayed, "\n")) > 1 {
			msgLines = append(msgLines, msgStyle.Render(strings.Split(displayed, "\n")[1]))
		}
	}

	// Pad to exactly maxMsgLines.
	for len(msgLines) < maxMsgLines {
		msgLines = append(msgLines, emptyLine)
	}

	// Line 2 (or 3 if msg took 2 lines): progress bar or empty.
	if s.active {
		msgLines = append(msgLines, s.viewProgress(t))
	} else if len(msgLines) < 2 { //nolint:mnd // status bar height
		msgLines = append(msgLines, emptyLine)
	}

	// Ensure exactly 2 lines.
	if len(msgLines) > 2 { //nolint:mnd // status bar height
		msgLines = msgLines[:2] //nolint:mnd // status bar height
	}

	return strings.Join(msgLines, "\n")
}

func (s *StatusBar) viewProgress(t *uxtypes.Theme) string {
	s.progress.FullColor = string(t.Accent)
	s.progress.EmptyColor = string(t.Dim)

	label := lipgloss.NewStyle().
		Foreground(t.StatusFg).
		Background(t.StatusBg).
		Bold(true).
		PaddingLeft(1).
		Render(s.label)

	labelW := lipgloss.Width(label) + 1
	s.progress.Width = max(s.Width-labelW, 10) //nolint:mnd // minimum bar width

	bar := s.progress.ViewAs(s.percent)

	return lipgloss.NewStyle().
		Background(t.StatusBg).
		Width(s.Width).
		Render(label + " " + bar)
}

// elideMessage truncates a multi-line message to fit within maxLines lines
// of the given width. Returns the elided message and whether truncation occurred.
const minElideWidth = 10

func elideMessage(msg string, maxWidth, maxLines int) (string, bool) {
	if maxWidth < minElideWidth {
		maxWidth = minElideWidth
	}

	if maxLines < 1 {
		maxLines = 1
	}

	lines := strings.Split(msg, "\n")

	if len(lines) <= maxLines {
		allFit := true

		for _, line := range lines {
			if len(line) > maxWidth {
				allFit = false

				break
			}
		}

		if allFit {
			return msg, false
		}
	}

	var result []string
	truncated := len(lines) > maxLines

	for i := range min(len(lines), maxLines) {
		line := lines[i]
		if len(line) > maxWidth {
			line = line[:maxWidth-3] + "..." //nolint:mnd // ellipsis
			truncated = true
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n"), truncated
}

func (s *StatusBar) tick() tea.Cmd {
	const eightyMs = 80_000_000

	return tea.Tick(
		eightyMs,
		func(_ time.Time) tea.Msg { return TickMsg{} },
	)
}

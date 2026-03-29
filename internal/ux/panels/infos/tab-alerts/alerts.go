package alerts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// severityBullet returns a colored emoji for the given engine.Severity.
func severityBullet(s engine.Severity) string {
	switch s {
	case engine.SeverityCritical:
		return "💀"
	case engine.SeverityHigh:
		return "🔴"
	case engine.SeverityMedium:
		return "🟠"
	case engine.SeverityLow:
		return "🟡"
	case engine.SeverityInfo:
		return "🔵"
	default:
		return "⚪"
	}
}

// alertCardLines is the number of display lines per alert card.
const alertCardLines = 3

// AlertsPanel displays alerts as multi-line cards.
// Each card shows:
//   - Line 1: severity bullet + summary + fix indicator
//   - Line 2: detail text (dimmed)
//   - Line 3: separator
type AlertsPanel struct {
	Alerts []engine.Alert // real alerts from the engine
	Cursor int
	Offset int // scroll offset in cards (not lines)
	Width  int
	Height int
}

// New creates a new AlertsPanel with no entries.
func New() AlertsPanel {
	return AlertsPanel{}
}

// SetAlerts replaces the displayed alerts with new ones.
// Alerts with SeverityNone are filtered out (check passed, nothing wrong).
func (p *AlertsPanel) SetAlerts(alerts []engine.Alert) {
	p.Alerts = p.Alerts[:0]

	for _, a := range alerts {
		if a.Severity == engine.SeverityNone {
			continue
		}

		p.Alerts = append(p.Alerts, a)
	}

	p.Cursor = 0
	p.Offset = 0
}

// SelectedAlert returns the currently selected alert, if any.
func (p *AlertsPanel) SelectedAlert() (engine.Alert, bool) {
	if len(p.Alerts) == 0 || p.Cursor < 0 || p.Cursor >= len(p.Alerts) {
		return engine.Alert{}, false
	}

	return p.Alerts[p.Cursor], true
}

func (p *AlertsPanel) SetSize(w, h int) {
	p.Width = w
	p.Height = h - 1 // reserve 1 line for header

	if p.Height < alertCardLines {
		p.Height = alertCardLines
	}

	p.clampScroll()
}

func (p *AlertsPanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch km.String() {
	case "up", "k":
		if p.Cursor > 0 {
			p.Cursor--
			p.clampScroll()
		}
	case "down", "j":
		if p.Cursor < len(p.Alerts)-1 {
			p.Cursor++
			p.clampScroll()
		}
	case "home", "g":
		p.Cursor = 0
		p.clampScroll()
	case "end", "G":
		if len(p.Alerts) > 0 {
			p.Cursor = len(p.Alerts) - 1
		}

		p.clampScroll()
	case "enter":
		// Show suggested actions for the selected alert.
		if len(p.Alerts) > 0 {
			return func() tea.Msg {
				return types.ShowSuggestionsMsg{AlertIndex: p.Cursor}
			}
		}
	case "c":
		// Copy URL from the selected alert's suggestion to clipboard.
		if url := p.selectedAlertURL(); url != "" {
			return func() tea.Msg {
				return types.CopyToClipboardMsg{Text: url}
			}
		}
	}

	return nil
}

// selectedAlertURL returns the first URL-like subject from the
// selected alert's suggestions (used by open-in-browser actions).
func (p *AlertsPanel) selectedAlertURL() string {
	if p.Cursor < 0 || p.Cursor >= len(p.Alerts) {
		return ""
	}

	for _, s := range p.Alerts[p.Cursor].Suggestions {
		if s.ActionName == "open-in-browser" && len(s.Subjects) > 0 {
			return s.Subjects[0]
		}
	}

	return ""
}

func (p *AlertsPanel) clampScroll() {
	visibleCards := p.Height / alertCardLines
	if visibleCards < 1 {
		visibleCards = 1
	}

	if p.Cursor < p.Offset {
		p.Offset = p.Cursor
	}

	if p.Cursor >= p.Offset+visibleCards {
		p.Offset = p.Cursor - visibleCards + 1
	}
}

func (p *AlertsPanel) View() string {
	t := types.CurrentTheme
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText)
	detailStyle := lipgloss.NewStyle().Foreground(t.Dim)
	selectedBg := lipgloss.NewStyle().Background(t.SelectedBg)
	sepStyle := lipgloss.NewStyle().Foreground(t.Dim)

	header := headerStyle.Render(fmt.Sprintf("  Alerts (%d)", len(p.Alerts)))
	if len(p.Alerts) > 0 {
		hints := "Enter: show actions"
		if p.selectedAlertURL() != "" {
			hints += "  c: copy URL"
		}

		header += "  " + detailStyle.Render(hints)
	}

	if len(p.Alerts) == 0 {
		empty := detailStyle.Render("  No alerts for this repository")

		return header + "\n" + empty
	}

	visibleCards := p.Height / alertCardLines
	if visibleCards < 1 {
		visibleCards = 1
	}

	contentWidth := p.Width - 4 // indent
	sep := sepStyle.Render(strings.Repeat("─", contentWidth))

	var rows []string

	end := p.Offset + visibleCards
	if end > len(p.Alerts) {
		end = len(p.Alerts)
	}

	for i := p.Offset; i < end; i++ {
		a := p.Alerts[i]
		selected := i == p.Cursor

		// Line 1: severity + summary + fix indicator
		fixIcon := ""
		if len(a.Suggestions) > 0 {
			fixIcon = fmt.Sprintf("  ⚡%d fix", len(a.Suggestions))
		}

		line1 := fmt.Sprintf("  %s %s%s", severityBullet(a.Severity), a.Summary, fixIcon)

		// Line 2: detail (truncated to fit)
		line2 := "  " + detailStyle.Render("(no detail)")

		if a.Detail != "" {
			detail := a.Detail
			if len(detail) > contentWidth {
				detail = detail[:contentWidth-3] + "..."
			}

			line2 = "  " + detailStyle.Render(detail)
		}

		card := line1 + "\n" + line2 + "\n" + sep

		if selected {
			card = selectedBg.Width(p.Width).Render(line1+"\n"+line2) + "\n" + sep
		}

		rows = append(rows, card)
	}

	// Pad remaining space.
	usedLines := len(rows) * alertCardLines

	for usedLines < p.Height {
		rows = append(rows, "")
		usedLines++
	}

	return header + "\n" + strings.Join(rows, "\n")
}

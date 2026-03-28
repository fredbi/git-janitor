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

// column widths.
const (
	colWidthSeverity = 3
	colWidthActions  = 5
	colWidthMinMsg   = 20
)

// AlertsPanel displays alerts as a hand-rendered borderless table
// with colored cells, using manual cursor management for scrolling.
type AlertsPanel struct {
	Alerts []engine.Alert // real alerts from the engine
	Cursor int
	Offset int // scroll offset (first visible row index)
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

	if p.Height < 1 {
		p.Height = 1
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
		p.Cursor = max(0, len(p.Alerts)-1)
		p.clampScroll()
	case "enter":
		// Show suggested actions for the selected alert.
		if len(p.Alerts) > 0 {
			return func() tea.Msg {
				return types.ShowSuggestionsMsg{AlertIndex: p.Cursor}
			}
		}
	}

	return nil
}

func (p *AlertsPanel) clampScroll() {
	if p.Cursor < p.Offset {
		p.Offset = p.Cursor
	}

	if p.Cursor >= p.Offset+p.Height {
		p.Offset = p.Cursor - p.Height + 1
	}
}

func (p *AlertsPanel) View() string {
	msgWidth := p.Width - colWidthSeverity - colWidthActions - 4
	if msgWidth < colWidthMinMsg {
		msgWidth = colWidthMinMsg
	}

	t := types.CurrentTheme
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.HeaderText)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Bright).
		Background(t.SelectedBg)

	detailStyle := lipgloss.NewStyle().
		Foreground(t.Dim)

	sevCol := lipgloss.NewStyle().Width(colWidthSeverity).Align(lipgloss.Center)
	msgCol := lipgloss.NewStyle().Width(msgWidth)
	actCol := lipgloss.NewStyle().Width(colWidthActions).Align(lipgloss.Center)

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		sevCol.Render(headerStyle.Render(" ")),
		" ",
		msgCol.Render(headerStyle.Render("Alert")),
		" ",
		actCol.Render(headerStyle.Render("Fix?")),
	)

	if len(p.Alerts) == 0 {
		empty := detailStyle.Render("  No alerts for this repository")

		return header + "\n" + empty
	}

	var rows []string

	end := p.Offset + p.Height
	if end > len(p.Alerts) {
		end = len(p.Alerts)
	}

	for i := p.Offset; i < end; i++ {
		a := p.Alerts[i]

		msgText := a.Summary
		if a.Detail != "" {
			msgText += "  " + detailStyle.Render(a.Detail)
		}

		fixIcon := " "
		if len(a.Suggestions) > 0 {
			fixIcon = fmt.Sprintf("⚡%d", len(a.Suggestions))
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			sevCol.Render(severityBullet(a.Severity)),
			" ",
			msgCol.Render(msgText),
			" ",
			actCol.Render(fixIcon),
		)

		if i == p.Cursor {
			row = selectedStyle.Render(row)
		}

		rows = append(rows, row)
	}

	for len(rows) < p.Height {
		rows = append(rows, "")
	}

	return header + "\n" + strings.Join(rows, "\n")
}

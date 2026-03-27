package state

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// severity represents the urgency level of an alert.
type severity int

const (
	severityHigh   severity = iota // red bullet
	severityMedium                 // orange bullet
	severityLow                    // yellow bullet
)

// severityBullet returns a colored emoji for the given severity.
func severityBullet(s severity) string {
	switch s {
	case severityHigh:
		return "🔴"
	case severityMedium:
		return "🟠"
	case severityLow:
		return "🟡"
	default:
		return "⚪"
	}
}

// scheduledCheckbox returns a checkbox emoji.
func scheduledCheckbox(scheduled bool) string {
	if scheduled {
		return " ✅"
	}

	return " ⬜"
}

// alertEntry holds the structured data for an alert row.
type alertEntry struct {
	severity  severity
	message   string
	detail    string
	scheduled bool
}

// column widths (severity bullet is narrow, scheduled is fixed).
const (
	colWidthSeverity  = 3
	colWidthScheduled = 5
	colWidthMinMsg    = 20
)

// alertsPanel displays alerts as a hand-rendered borderless table
// with colored cells, using manual cursor management for scrolling.
type alertsPanel struct {
	entries []alertEntry
	cursor  int
	offset  int // scroll offset (first visible row index)
	width   int
	height  int
}

func newAlertsPanel() alertsPanel {
	entries := []alertEntry{
		{severityHigh, "Stale branches", "3 branches older than 90 days", false},
		{severityMedium, "Merged branches", "2 branches already merged to main", true},
		{severityLow, "Large files", "1 file over 50MB detected", false},
		{severityHigh, "Unpushed commits", "5 commits ahead of origin/main", false},
	}

	return alertsPanel{entries: entries}
}

func (p *alertsPanel) SetSize(w, h int) {
	p.width = w
	// Reserve 1 line for the header.
	p.height = h - 1
	if p.height < 1 {
		p.height = 1
	}

	p.clampScroll()
}

func (p *alertsPanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	switch km.String() {
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
			p.clampScroll()
		}
	case "down", "j":
		if p.cursor < len(p.entries)-1 {
			p.cursor++
			p.clampScroll()
		}
	case "home", "g":
		p.cursor = 0
		p.clampScroll()
	case "end", "G":
		p.cursor = max(0, len(p.entries)-1)
		p.clampScroll()
	}

	return nil
}

func (p *alertsPanel) clampScroll() {
	// Ensure cursor is visible within the scroll window.
	if p.cursor < p.offset {
		p.offset = p.cursor
	}

	if p.cursor >= p.offset+p.height {
		p.offset = p.cursor - p.height + 1
	}
}

func (p *alertsPanel) View() string {
	msgWidth := p.width - colWidthSeverity - colWidthScheduled - 4 // gaps between columns
	if msgWidth < colWidthMinMsg {
		msgWidth = colWidthMinMsg
	}

	// Styles.
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("245"))

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("63"))

	detailStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	sevCol := lipgloss.NewStyle().Width(colWidthSeverity).Align(lipgloss.Center)
	msgCol := lipgloss.NewStyle().Width(msgWidth)
	schedCol := lipgloss.NewStyle().Width(colWidthScheduled).Align(lipgloss.Center)

	// Header row.
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		sevCol.Render(headerStyle.Render(" ")),
		" ",
		msgCol.Render(headerStyle.Render("Message")),
		" ",
		schedCol.Render(headerStyle.Render("Sched")),
	)

	// Data rows.
	var rows []string
	end := p.offset + p.height
	if end > len(p.entries) {
		end = len(p.entries)
	}

	for i := p.offset; i < end; i++ {
		e := p.entries[i]

		msgText := e.message + "  " + detailStyle.Render(e.detail)

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			sevCol.Render(severityBullet(e.severity)),
			" ",
			msgCol.Render(msgText),
			" ",
			schedCol.Render(scheduledCheckbox(e.scheduled)),
		)

		if i == p.cursor {
			row = selectedStyle.Render(row)
		}

		rows = append(rows, row)
	}

	// Pad remaining lines if fewer rows than height.
	for len(rows) < p.height {
		rows = append(rows, "")
	}

	return header + "\n" + strings.Join(rows, "\n")
}

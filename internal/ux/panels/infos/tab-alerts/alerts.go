package alerts

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// Severity represents the urgency level of an alert.
type Severity int

const (
	SeverityHigh   Severity = iota // red bullet
	SeverityMedium                 // orange bullet
	SeverityLow                    // yellow bullet
)

// severityBullet returns a colored emoji for the given Severity.
func severityBullet(s Severity) string {
	switch s {
	case SeverityHigh:
		return "🔴"
	case SeverityMedium:
		return "🟠"
	case SeverityLow:
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

// AlertEntry holds the structured data for an alert row.
type AlertEntry struct {
	Severity  Severity
	Message   string
	Detail    string
	Scheduled bool
}

// column widths (Severity bullet is narrow, scheduled is fixed).
const (
	colWidthSeverity  = 3
	colWidthScheduled = 5
	colWidthMinMsg    = 20
)

// AlertsPanel displays alerts as a hand-rendered borderless table
// with colored cells, using manual cursor management for scrolling.
type AlertsPanel struct {
	Entries []AlertEntry
	Cursor  int
	Offset  int // scroll offset (first visible row index)
	Width   int
	Height  int
}

// New creates a new AlertsPanel with sample entries.
func New() AlertsPanel {
	entries := []AlertEntry{
		{SeverityHigh, "Stale branches", "3 branches older than 90 days", false},
		{SeverityMedium, "Merged branches", "2 branches already merged to main", true},
		{SeverityLow, "Large files", "1 file over 50MB detected", false},
		{SeverityHigh, "Unpushed commits", "5 commits ahead of origin/main", false},
	}

	return AlertsPanel{Entries: entries}
}

func (p *AlertsPanel) SetSize(w, h int) {
	p.Width = w
	// Reserve 1 line for the header.
	p.Height = h - 1
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
		if p.Cursor < len(p.Entries)-1 {
			p.Cursor++
			p.clampScroll()
		}
	case "home", "g":
		p.Cursor = 0
		p.clampScroll()
	case "end", "G":
		p.Cursor = max(0, len(p.Entries)-1)
		p.clampScroll()
	}

	return nil
}

func (p *AlertsPanel) clampScroll() {
	// Ensure cursor is visible within the scroll window.
	if p.Cursor < p.Offset {
		p.Offset = p.Cursor
	}

	if p.Cursor >= p.Offset+p.Height {
		p.Offset = p.Cursor - p.Height + 1
	}
}

func (p *AlertsPanel) View() string {
	msgWidth := p.Width - colWidthSeverity - colWidthScheduled - 4 // gaps between columns
	if msgWidth < colWidthMinMsg {
		msgWidth = colWidthMinMsg
	}

	// Styles.
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
	end := p.Offset + p.Height
	if end > len(p.Entries) {
		end = len(p.Entries)
	}

	for i := p.Offset; i < end; i++ {
		e := p.Entries[i]

		msgText := e.Message + "  " + detailStyle.Render(e.Detail)

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			sevCol.Render(severityBullet(e.Severity)),
			" ",
			msgCol.Render(msgText),
			" ",
			schedCol.Render(scheduledCheckbox(e.Scheduled)),
		)

		if i == p.Cursor {
			row = selectedStyle.Render(row)
		}

		rows = append(rows, row)
	}

	// Pad remaining lines if fewer rows than height.
	for len(rows) < p.Height {
		rows = append(rows, "")
	}

	return header + "\n" + strings.Join(rows, "\n")
}

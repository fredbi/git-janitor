// SPDX-License-Identifier: Apache-2.0

package alerts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	"github.com/fredbi/git-janitor/internal/ux/key"
	"github.com/fredbi/git-janitor/internal/ux/panels"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// alertCardLines is the number of display lines per alert card.
const alertCardLines = 3

// Panel displays alerts as multi-line cards.
// Each card shows:
//   - Line 1: severity bullet + summary + fix indicator
//   - Line 2: detail text (dimmed)
//   - Line 3: separator
type Panel struct {
	panels.Base

	Alerts []models.Alert // real alerts from the engine
}

// New creates a new Panel with no entries.
func New(theme *types.Theme) Panel {
	return Panel{Base: panels.Base{Theme: theme}}
}

// SetAlerts replaces the displayed alerts with new ones.
// Alerts with SeverityNone are filtered out (check passed, nothing wrong).
func (p *Panel) SetAlerts(alerts []models.Alert) {
	p.Alerts = p.Alerts[:0]

	for _, a := range alerts {
		if a.Severity == models.SeverityNone {
			continue
		}

		p.Alerts = append(p.Alerts, a)
	}

	p.ResetScroll()
}

// SelectedAlert returns the currently selected alert, if any.
func (p *Panel) SelectedAlert() (models.Alert, bool) {
	if len(p.Alerts) == 0 || p.Cursor < 0 || p.Cursor >= len(p.Alerts) {
		return models.Alert{}, false
	}

	return p.Alerts[p.Cursor], true
}

func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 1, alertCardLines)
	p.ClampScroll(p.visibleCards())
}

func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if p.NavigateKey(km, len(p.Alerts)) {
		p.ClampScroll(p.visibleCards())

		return nil
	}

	switch key.MsgBinding(km) {
	case key.Enter:
		// Show suggested actions for the selected alert.
		if len(p.Alerts) > 0 {
			return func() tea.Msg {
				return types.ShowSuggestionsMsg{AlertIndex: p.Cursor}
			}
		}
	case key.C:
		// Copy URL from the selected alert's suggestion to clipboard.
		if url := p.selectedAlertURL(); url != "" {
			return func() tea.Msg {
				return types.CopyToClipboardMsg{Text: url}
			}
		}
	}

	return nil
}

func (p *Panel) View() string {
	t := p.Theme
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

	visible := p.visibleCards()

	const indent = 4
	contentWidth := p.Width - indent
	sep := sepStyle.Render(strings.Repeat("─", contentWidth))

	var rows []string

	start, end := p.VisibleRange(len(p.Alerts), visible)

	for i := start; i < end; i++ {
		a := p.Alerts[i]
		selected := i == p.Cursor

		// Line 1: severity + summary + fix indicator
		fixIcon := ""
		if len(a.Suggestions) > 0 {
			fixIcon = fmt.Sprintf("  ⚡%d fix", len(a.Suggestions))
		}

		line1 := fmt.Sprintf("  %s %s%s", gadgets.SeverityBullet(a.Severity), a.Summary, fixIcon)

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
	rows = panels.PadRows(rows, usedLines+(p.Height-usedLines))

	return header + "\n" + strings.Join(rows, "\n")
}

// selectedAlertURL returns the first URL-like subject from the
// selected alert's suggestions (used by open-in-browser actions).
func (p *Panel) selectedAlertURL() string {
	if p.Cursor < 0 || p.Cursor >= len(p.Alerts) {
		return ""
	}

	for _, s := range p.Alerts[p.Cursor].Suggestions {
		if s.ActionName == "open-in-browser" && len(s.Subjects) > 0 {
			return s.Subjects[0].Subject
		}
	}

	return ""
}

func (p *Panel) visibleCards() int {
	return max(p.Height/alertCardLines, 1)
}

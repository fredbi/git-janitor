// SPDX-License-Identifier: Apache-2.0

package recent

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	"github.com/fredbi/git-janitor/internal/ux/panels"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Panel displays recent action history for the selected repo.
type Panel struct {
	panels.Base

	entries []models.HistoryEntry
}

// New creates a new Panel with no entries.
func New(theme *uxtypes.Theme) Panel {
	return Panel{Base: panels.Base{Theme: theme}}
}

// SetHistory replaces the displayed entries.
func (p *Panel) SetHistory(entries []models.HistoryEntry) {
	p.entries = entries
	p.ResetScroll()
}

// SetSize updates the panel dimensions.
func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 1, 1) // 1 reserved line for header
}

// Update handles key messages for cursor navigation.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	if km, ok := msg.(tea.KeyMsg); ok {
		p.NavigateKey(km, len(p.entries))
		p.ClampScroll(p.Height)
	}

	return nil
}

// View renders the recent history list.
func (p *Panel) View() string {
	t := p.Theme

	headerStyle := lipgloss.NewStyle().
		Foreground(t.HeaderText).
		Bold(true)
	header := headerStyle.Render(fmt.Sprintf(" Recent actions (%d)", len(p.entries)))

	if len(p.entries) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(t.Dim).PaddingLeft(1)

		return header + "\n" + emptyStyle.Render("No recent actions for this repo.")
	}

	p.ClampScroll(p.Height)

	start, end := p.VisibleRange(len(p.entries), p.Height)
	rows := make([]string, 0, end-start)

	for i := start; i < end; i++ {
		rows = append(rows, p.renderEntry(i))
	}

	rows = panels.PadRows(rows, p.Height)

	return header + "\n" + strings.Join(rows, "\n")
}

// SetTheme updates the theme for rendering. This is called from the parent panel.
func (p *Panel) SetTheme(theme *uxtypes.Theme) {
	p.Theme = theme
}

func (p *Panel) renderEntry(idx int) string {
	e := p.entries[idx]
	t := p.Theme
	selected := idx == p.Cursor

	// Icon: ✓ or ✗.
	var icon string
	if e.Result.OK {
		icon = lipgloss.NewStyle().Foreground(t.Success).Render("✓")
	} else {
		icon = lipgloss.NewStyle().Foreground(t.Error).Render("✗")
	}

	// Action name.
	nameStyle := lipgloss.NewStyle().Foreground(t.Text)
	if selected {
		nameStyle = nameStyle.Foreground(t.Bright).Bold(true)
	}

	name := nameStyle.Render(e.ActionName)

	// Subjects (truncated).
	subjectStr := strings.Join(e.Subjects, ", ")
	subjectStr = gadgets.ElideLongLabel(subjectStr)

	subjectStyle := lipgloss.NewStyle().Foreground(t.Dim)
	subjects := subjectStyle.Render(subjectStr)

	// Relative time.
	timeStyle := lipgloss.NewStyle().Foreground(t.Dim)
	timeStr := timeStyle.Render(gadgets.TimeAgo(e.Timestamp))

	// Build the line.
	line := fmt.Sprintf(" %s %s  %s  · %s", icon, name, subjects, timeStr)

	if selected {
		line = lipgloss.NewStyle().
			Background(t.SelectedBg).
			Width(p.Width).
			Render(line)
	}

	return line
}

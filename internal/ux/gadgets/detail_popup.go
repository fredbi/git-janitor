// SPDX-License-Identifier: Apache-2.0

package gadgets

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// DetailPopup is a scrollable overlay that displays detail text
// for a selected item (branch, stash, status message, etc.).
type DetailPopup struct {
	Theme    *uxtypes.Theme
	Viewport viewport.Model
	Visible  bool
	Title    string
	Content  string                  // raw content for clipboard copy
	Scope    models.ActionSuggestion // subject scope for actions (delete, etc.)
	Width    int
	Height   int
}

// NewDetailPopup creates a new DetailPopup.
func NewDetailPopup(theme *uxtypes.Theme) DetailPopup {
	return DetailPopup{
		Theme:    theme,
		Viewport: viewport.New(0, 0),
	}
}

// Show makes the popup visible with the given title and content.
// scope identifies the subject being viewed (for actions like delete).
func (d *DetailPopup) Show(title, content string, scope models.ActionSuggestion) {
	d.Title = title
	d.Content = content
	d.Scope = scope
	d.Viewport.SetContent(content)
	d.Visible = true
	d.Viewport.GotoTop()
}

// Hide hides the popup.
func (d *DetailPopup) Hide() {
	d.Visible = false
}

// CanDelete reports whether the current scope supports a delete action.
func (d *DetailPopup) CanDelete() bool {
	return d.Scope.SubjectKind == models.SubjectBranch || d.Scope.SubjectKind == models.SubjectStash
}

// SetSize recalculates the popup dimensions (centered, ~60% of terminal).
func (d *DetailPopup) SetSize(termWidth, termHeight int) {
	d.Width = termWidth * 3 / 5  //nolint:mnd // 60% width
	if d.Width < 40 {            //nolint:mnd // minimum width
		d.Width = min(40, termWidth) //nolint:mnd // minimum width
	}

	d.Height = termHeight * 3 / 5 //nolint:mnd // 60% height
	if d.Height < 10 {             //nolint:mnd // minimum height
		d.Height = min(10, termHeight) //nolint:mnd // minimum height
	}

	// Account for border (2) + title (1) + padding (1) + hint (1).
	d.Viewport.Width = d.Width - 4   //nolint:mnd // border
	d.Viewport.Height = d.Height - 7 //nolint:mnd // border + title + padding + hint
}

// Update handles messages for the popup viewport (scroll keys).
func (d *DetailPopup) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	d.Viewport, cmd = d.Viewport.Update(msg)

	return cmd
}

// View renders the popup overlay centered on the screen.
func (d *DetailPopup) View(termWidth, termHeight int) string {
	if !d.Visible {
		return ""
	}

	t := d.Theme
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Bright).
		PaddingBottom(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		PaddingTop(1)

	hint := "Esc: close  C: copy to clipboard"
	if d.CanDelete() {
		hint += "  D: delete"
	}

	content := titleStyle.Render(d.Title) + "\n" +
		d.Viewport.View() + "\n" +
		hintStyle.Render(hint)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Tertiary).
		Width(d.Width - 2).  //nolint:mnd // border
		Height(d.Height - 2). //nolint:mnd // border
		Padding(0, 1)

	popup := border.Render(content)

	return lipgloss.Place(
		termWidth, termHeight,
		lipgloss.Center, lipgloss.Center,
		popup,
	)
}

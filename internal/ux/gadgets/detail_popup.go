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
	IsRemote bool                    // true when showing a remote branch

	// WorktreeLocked / WorktreePrunable / WorktreeMain capture the
	// boolean state of a worktree at the moment the popup was opened.
	// Used to gate the R / D / U / L key hints. The main worktree
	// cannot be removed, repaired, locked or unlocked via git —
	// those actions are hidden.
	WorktreeLocked   bool
	WorktreePrunable bool
	WorktreeMain     bool

	// URL, when non-empty, enables the "open in browser" action. The model
	// wires the O key to dispatch open-in-browser with this URL.
	URL string
	// Footer, when non-empty, is displayed on a reserved line between the
	// scrollable body and the key-hint line (e.g. "N comments" for issues).
	Footer string
	Width  int
	Height int
}

// NewDetailPopup creates a new DetailPopup.
func NewDetailPopup(theme *uxtypes.Theme) DetailPopup {
	return DetailPopup{
		Theme:    theme,
		Viewport: viewport.New(0, 0),
	}
}

// Show makes the popup visible with the given title and content. The URL
// and Footer are reset — callers that want the "open in browser" shortcut
// or a footer line must call [DetailPopup.SetURL] / [DetailPopup.SetFooter]
// after Show. scope identifies the subject being viewed.
func (d *DetailPopup) Show(title, content string, scope models.ActionSuggestion) {
	d.Title = title
	d.Content = content
	d.Scope = scope
	d.URL = ""
	d.Footer = ""
	d.WorktreeLocked = false
	d.WorktreePrunable = false
	d.WorktreeMain = false
	d.applyContent()
	d.Visible = true
	d.Viewport.GotoTop()
}

// SetURL attaches a URL to the currently displayed popup. When set, the
// popup offers an "open in browser" shortcut (O key).
func (d *DetailPopup) SetURL(url string) {
	d.URL = url
}

// SetFooter attaches a single-line footer displayed between the body and
// the key-hint line. Pass "" to clear it. Footer height is reclaimed from
// the scrollable viewport.
func (d *DetailPopup) SetFooter(footer string) {
	d.Footer = footer
	d.applyContent()
}

// CanOpenInBrowser reports whether the popup has a URL to launch.
func (d *DetailPopup) CanOpenInBrowser() bool {
	return d.URL != ""
}

// CanSelfAssign reports whether the current scope supports the
// self-assign-issue action (i.e. it is a single GitHub issue).
func (d *DetailPopup) CanSelfAssign() bool {
	return d.Scope.SubjectKind == models.SubjectIssueDetail && len(d.Scope.Subjects) > 0
}

// CanCloseIssue reports whether the current scope supports the close-issue
// action (i.e. it is a single GitHub issue).
func (d *DetailPopup) CanCloseIssue() bool {
	return d.Scope.SubjectKind == models.SubjectIssueDetail && len(d.Scope.Subjects) > 0
}

// Hide hides the popup.
func (d *DetailPopup) Hide() {
	d.Visible = false
}

// CanDelete reports whether the current scope supports a delete action.
// A locked or main worktree cannot be removed via git.
func (d *DetailPopup) CanDelete() bool {
	switch d.Scope.SubjectKind {
	case models.SubjectBranch, models.SubjectStash:
		return true
	case models.SubjectWorktree:
		return !d.WorktreeLocked && !d.WorktreeMain
	default:
		return false
	}
}

// CanRebase reports whether the current scope supports a rebase action.
// Only local branches can be rebased (remote branches cannot).
func (d *DetailPopup) CanRebase() bool {
	return d.Scope.SubjectKind == models.SubjectBranch && !d.IsRemote
}

// CanRepair reports whether the current scope supports a repair action.
// Meaningful only on a linked worktree whose working directory has
// gone missing (Prunable=true).
func (d *DetailPopup) CanRepair() bool {
	return d.Scope.SubjectKind == models.SubjectWorktree && d.WorktreePrunable && !d.WorktreeMain
}

// CanUnlock reports whether the current scope supports an unlock
// action — a locked linked worktree.
func (d *DetailPopup) CanUnlock() bool {
	return d.Scope.SubjectKind == models.SubjectWorktree && d.WorktreeLocked && !d.WorktreeMain
}

// CanLock reports whether the current scope supports a lock action —
// an unlocked linked (non-main) worktree.
func (d *DetailPopup) CanLock() bool {
	return d.Scope.SubjectKind == models.SubjectWorktree && !d.WorktreeLocked && !d.WorktreeMain
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

	// Account for border (2) + title (1) + padding (1) + hint (1) + optional footer (1).
	const chromeLines = 7

	d.Viewport.Width = d.Width - 4 //nolint:mnd // border + inner padding

	vpHeight := d.Height - chromeLines
	if d.Footer != "" {
		vpHeight--
	}

	d.Viewport.Height = vpHeight

	// Re-wrap content to the new width.
	d.applyContent()
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

	hint := "Esc: close  C: copy to clipboard  Ctrl+K: quick actions"
	if d.CanRebase() {
		hint += "  R: rebase"
	}
	if d.CanRepair() {
		hint += "  R: repair"
	}
	if d.CanUnlock() {
		hint += "  U: unlock"
	}
	if d.CanLock() {
		hint += "  L: lock"
	}
	if d.CanDelete() {
		hint += "  D: delete"
	}
	if d.CanOpenInBrowser() {
		hint += "  O: open in browser"
	}
	if d.CanSelfAssign() {
		hint += "  S: self-assign"
	}
	if d.CanCloseIssue() {
		hint += "  X: close issue"
	}

	body := titleStyle.Render(d.Title) + "\n" +
		d.Viewport.View()

	if d.Footer != "" {
		footerStyle := lipgloss.NewStyle().Foreground(t.Dim)
		body += "\n" + footerStyle.Render(d.Footer)
	}

	body += "\n" + hintStyle.Render(hint)
	content := body

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

// applyContent wraps the raw content to the current viewport width and
// pushes it into the viewport. Long lines (e.g. paragraph text from an
// issue body) get soft-wrapped so the user can scroll through them.
func (d *DetailPopup) applyContent() {
	width := d.Viewport.Width
	if width <= 0 {
		d.Viewport.SetContent(d.Content)

		return
	}

	wrapped := lipgloss.NewStyle().Width(width).Render(d.Content)
	d.Viewport.SetContent(wrapped)
}

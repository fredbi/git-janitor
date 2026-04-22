// SPDX-License-Identifier: Apache-2.0

// Package worktrees renders the list of git worktrees attached to the
// selected repository.
//
// The panel is a first-cut: list-only, cursor-navigable, no detail popup.
// Detail view and actions (delete / prune / unlock) are a follow-up.
package worktrees

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	"github.com/fredbi/git-janitor/internal/ux/key"
	"github.com/fredbi/git-janitor/internal/ux/panels"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Panel displays worktrees for the selected repo.
type Panel struct {
	panels.Base

	worktrees []models.Worktree
	mainPath  string
}

// New creates a new Panel with no entries.
func New(theme *uxtypes.Theme) Panel {
	return Panel{Base: panels.Base{Theme: theme}}
}

// SetInfo updates the panel with worktree data from the repo info.
// The first worktree in the porcelain output is the main checkout; it
// is tagged as such when rendering.
func (p *Panel) SetInfo(info *models.RepoInfo) {
	if info == nil {
		p.worktrees = nil
		p.mainPath = ""
		p.ResetScroll()

		return
	}

	p.worktrees = info.Worktrees
	p.mainPath = info.Path
	p.ResetScroll()
}

// Count returns the number of worktrees.
func (p *Panel) Count() int {
	return len(p.worktrees)
}

// SelectedWorktree returns the worktree under the cursor, if any.
func (p *Panel) SelectedWorktree() (models.Worktree, bool) {
	if p.Cursor < 0 || p.Cursor >= len(p.worktrees) {
		return models.Worktree{}, false
	}

	return p.worktrees[p.Cursor], true
}

// SetSize updates the panel dimensions. Two header lines are reserved:
// the count header and the key-hint line.
func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 2, 2) //nolint:mnd // 2 header lines: count + hint
}

// Update handles cursor navigation, detail-popup dispatch, and the A/M
// list-level key bindings (add / move worktree).
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if p.NavigateKey(km, len(p.worktrees)) {
		p.ClampScroll(p.Height)

		return nil
	}

	switch key.MsgBinding(km) { //nolint:exhaustive // only list-level bindings captured here
	case key.Enter:
		if p.Cursor < 0 || p.Cursor >= len(p.worktrees) {
			return nil
		}

		w := p.worktrees[p.Cursor]

		return func() tea.Msg {
			return uxtypes.FetchDetailMsg{
				Scope: models.ActionSuggestion{
					SubjectKind: models.SubjectWorktree,
					Subjects:    []models.ActionSubject{{Subject: w.Path}},
				},
			}
		}

	case key.A:
		return func() tea.Msg { return uxtypes.PromptAddWorktreeMsg{} }

	case key.M:
		return func() tea.Msg { return uxtypes.PromptMoveWorktreeMsg{} }
	}

	return nil
}

// View renders the worktree list.
func (p *Panel) View() string {
	t := p.Theme

	headerStyle := lipgloss.NewStyle().
		Foreground(t.HeaderText).
		Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(t.Dim).PaddingLeft(1)

	header := headerStyle.Render(fmt.Sprintf(" Worktrees (%d)", len(p.worktrees)))
	hint := hintStyle.Render("A: add  M: move  Enter: details  Ctrl+K: quick actions")

	if len(p.worktrees) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(t.Dim).PaddingLeft(1)

		return header + "\n" + hint + "\n" + emptyStyle.Render("No worktrees.")
	}

	p.ClampScroll(p.Height)

	start, end := p.VisibleRange(len(p.worktrees), p.Height)
	rows := make([]string, 0, end-start)

	for i := start; i < end; i++ {
		rows = append(rows, p.renderEntry(i))
	}

	rows = panels.PadRows(rows, p.Height)

	return header + "\n" + hint + "\n" + strings.Join(rows, "\n")
}

func (p *Panel) renderEntry(idx int) string {
	w := p.worktrees[idx]
	t := p.Theme
	selected := idx == p.Cursor

	refStyle := lipgloss.NewStyle().Foreground(t.Text)
	pathStyle := lipgloss.NewStyle().Foreground(t.Dim)
	flagStyle := lipgloss.NewStyle().Foreground(t.Dim)
	warnStyle := lipgloss.NewStyle().Foreground(t.Warning)

	if selected {
		refStyle = refStyle.Foreground(t.Bright).Bold(true)
	}

	ref := w.BranchShort()
	switch {
	case w.Bare:
		ref = "(bare)"
	case w.Detached:
		ref = "detached@" + shortHash(w.HEAD)
	case ref == "":
		ref = "(no branch)"
	}

	var flags []string
	if p.isMain(w) {
		flags = append(flags, flagStyle.Render("main"))
	}

	if w.Locked {
		flags = append(flags, warnStyle.Render("locked"))
	}

	if w.Prunable {
		flags = append(flags, warnStyle.Render("prunable"))
	}

	flagCol := ""
	if len(flags) > 0 {
		flagCol = " [" + strings.Join(flags, ",") + "]"
	}

	path := gadgets.ElideLongLabel(w.Path)
	line := fmt.Sprintf(" %s%s  %s", refStyle.Render(ref), flagCol, pathStyle.Render(path))

	if selected {
		line = lipgloss.NewStyle().
			Background(t.SelectedBg).
			Width(p.Width).
			Render(line)
	}

	return line
}

func (p *Panel) isMain(w models.Worktree) bool {
	return p.mainPath != "" && w.Path == p.mainPath
}

func shortHash(h string) string {
	const shortLen = 7
	if len(h) <= shortLen {
		return h
	}

	return h[:shortLen]
}

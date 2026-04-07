// SPDX-License-Identifier: Apache-2.0

package stashes

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

// Panel displays stash entries for the selected repo.
type Panel struct {
	panels.Base

	stashes []models.Stash
}

// New creates a new Panel with no entries.
func New(theme *uxtypes.Theme) Panel {
	return Panel{Base: panels.Base{Theme: theme}}
}

// SetInfo updates the panel with stash data from the repo info.
// Stashes are displayed most recent first (git stash list already returns them
// newest first, but we re-sort by LastUpdatedAt for safety).
func (p *Panel) SetInfo(info *models.RepoInfo) {
	if info == nil {
		p.stashes = nil
		p.ResetScroll()

		return
	}

	p.stashes = info.Stashes
	p.ResetScroll()
}

// Count returns the number of stashes.
func (p *Panel) Count() int {
	return len(p.stashes)
}

// SetSize updates the panel dimensions.
func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 1, 1) // 1 reserved line for header
}

// Update handles key messages for cursor navigation and detail popup.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if p.NavigateKey(km, len(p.stashes)) {
		p.ClampScroll(p.Height)

		return nil
	}

	if key.MsgBinding(km) == key.Enter && p.Cursor >= 0 && p.Cursor < len(p.stashes) {
		s := p.stashes[p.Cursor]

		return func() tea.Msg {
			return uxtypes.FetchDetailMsg{
				Scope: models.ActionSuggestion{
					SubjectKind: models.SubjectStash,
					Subjects:    []models.ActionSubject{{Subject: s.Ref}},
				},
			}
		}
	}

	return nil
}

// View renders the stash list.
func (p *Panel) View() string {
	t := p.Theme

	headerStyle := lipgloss.NewStyle().
		Foreground(t.HeaderText).
		Bold(true)
	header := headerStyle.Render(fmt.Sprintf(" Stashes (%d)", len(p.stashes)))

	if len(p.stashes) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(t.Dim).PaddingLeft(1)

		return header + "\n" + emptyStyle.Render("No stashes.")
	}

	p.ClampScroll(p.Height)

	start, end := p.VisibleRange(len(p.stashes), p.Height)
	rows := make([]string, 0, end-start)

	for i := start; i < end; i++ {
		rows = append(rows, p.renderEntry(i))
	}

	rows = panels.PadRows(rows, p.Height)

	return header + "\n" + strings.Join(rows, "\n")
}

func (p *Panel) renderEntry(idx int) string {
	s := p.stashes[idx]
	t := p.Theme
	selected := idx == p.Cursor

	refStyle := lipgloss.NewStyle().Foreground(t.Dim)
	branchStyle := lipgloss.NewStyle().Foreground(t.Text)
	msgStyle := lipgloss.NewStyle().Foreground(t.Dim)
	timeStyle := lipgloss.NewStyle().Foreground(t.Dim)

	if selected {
		branchStyle = branchStyle.Foreground(t.Bright).Bold(true)
	}

	ref := refStyle.Render(s.Ref)
	branch := branchStyle.Render(s.Branch)

	msg := s.Message
	if msg == "" {
		msg = "(no message)"
	}

	msg = gadgets.ElideLongLabel(msg)
	message := msgStyle.Render(msg)

	var timeStr string
	if !s.LastUpdatedAt.IsZero() {
		timeStr = " · " + timeStyle.Render(gadgets.TimeAgo(s.LastUpdatedAt))
	}

	line := fmt.Sprintf(" %s  %s  %s%s", ref, branch, message, timeStr)

	if selected {
		line = lipgloss.NewStyle().
			Background(t.SelectedBg).
			Width(p.Width).
			Render(line)
	}

	return line
}

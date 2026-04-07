// SPDX-License-Identifier: Apache-2.0

package branches

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

// Column widths.
const (
	dateW   = 12 // "12 days ago" or "2025-03-23"
	hashW   = 8
	gap     = 2
	gaps    = 4 * gap // 4 gaps between 5 columns (marker+name, hash, date, upstream)
	minName = 10
)

// Panel displays the branches for the selected repository.
type Panel struct {
	panels.Base

	branches []models.Branch
}

// New creates a new Panel.
func New(theme *uxtypes.Theme) Panel {
	return Panel{Base: panels.Base{Theme: theme}}
}

// SetInfo updates the panel with sorted branch data.
func (p *Panel) SetInfo(info *models.RepoInfo) {
	if info == nil || info.IsEmpty() {
		p.branches = nil
		p.ResetScroll()

		return
	}

	// Copy and sort for display.
	sorted := make([]models.Branch, len(info.Branches))
	copy(sorted, info.Branches)
	models.SortBranches(sorted, info.DefaultBranch)

	p.branches = sorted
	p.ResetScroll()
}

// SetSize updates the panel dimensions.
func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 1, 1) // 1 reserved for header
}

// Update handles key messages for cursor navigation.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if len(p.branches) == 0 {
		return nil
	}

	if p.NavigateKey(km, len(p.branches)) {
		p.ClampScroll(p.Height)
	}

	return nil
}

// View renders the branches list.
func (p *Panel) View() string {
	if len(p.branches) == 0 {
		return lipgloss.NewStyle().Foreground(p.Theme.Dim).
			Render("  Select a repository to view its branches.")
	}

	t := p.Theme
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Bright).Background(t.SelectedBg)

	// Compute name column width.
	nameW := p.computeNameWidth()
	nameCol := lipgloss.NewStyle().Width(nameW)
	hashCol := lipgloss.NewStyle().Width(hashW)
	dateCol := lipgloss.NewStyle().Width(dateW)

	header := fmt.Sprintf("  %s  %s  %s  %s",
		nameCol.Render(headerStyle.Render("Branch")),
		hashCol.Render(headerStyle.Render("Hash")),
		dateCol.Render(headerStyle.Render("Updated")),
		headerStyle.Render("Upstream"),
	)

	start, end := p.VisibleRange(len(p.branches), p.Height)

	var rows []string

	for i := start; i < end; i++ {
		b := p.branches[i]
		row := p.renderBranch(b, nameW, nameCol, hashCol, dateCol)

		if i == p.Cursor {
			row = selectedStyle.Render(row)
		}

		rows = append(rows, row)
	}

	rows = panels.PadRows(rows, p.Height)

	return header + "\n" + strings.Join(rows, "\n")
}

func (p *Panel) computeNameWidth() int {
	nameW := minName

	for _, b := range p.branches {
		w := len(b.Name) + gap
		if w > nameW {
			nameW = w
		}
	}

	maxNameW := max(p.Width-hashW-dateW-gaps-8, minName) //nolint:mnd // 8 for marker prefix + upstream breathing room
	if nameW > maxNameW {
		nameW = maxNameW
	}

	return nameW
}

func (p *Panel) renderBranch(
	b models.Branch,
	nameW int,
	nameCol, hashCol, dateCol lipgloss.Style,
) string {
	t := p.Theme
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)

	// Marker + name.
	marker := "  "
	var nameStyle lipgloss.Style

	switch {
	case b.IsCurrent:
		marker = "* "
		nameStyle = lipgloss.NewStyle().Foreground(t.Success)
	case b.IsRemote:
		nameStyle = lipgloss.NewStyle().Foreground(t.Dim)
	default:
		nameStyle = lipgloss.NewStyle().Foreground(t.Text)
	}

	name := marker + b.Name
	if len(name) > nameW {
		name = name[:nameW-1] + "\u2026" // ellipsis
	}

	// Hash.
	hash := b.Hash
	if len(hash) > hashW {
		hash = hash[:hashW]
	}

	// Date.
	var dateStr string
	if b.LastCommit.IsZero() {
		dateStr = dimStyle.Render("-")
	} else {
		dateStr = gadgets.TimeAgo(b.LastCommit)
	}

	// Upstream.
	upstream := b.Upstream
	if upstream == "" {
		upstream = dimStyle.Render("-")
	} else {
		maxUpstream := p.Width - nameW - hashW - dateW - gaps
		if maxUpstream > 0 && len(upstream) > maxUpstream {
			upstream = upstream[:maxUpstream-1] + "\u2026"
		}
	}

	return fmt.Sprintf("%s  %s  %s  %s",
		nameCol.Render(nameStyle.Render(name)),
		hashCol.Render(dimStyle.Render(hash)),
		dateCol.Render(dimStyle.Render(dateStr)),
		upstream,
	)
}

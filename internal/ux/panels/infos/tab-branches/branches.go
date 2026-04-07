// SPDX-License-Identifier: Apache-2.0

package branches

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/panels"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Panel displays the branches for the selected repository.
type Panel struct {
	panels.Base

	Info *models.RepoInfo
}

// New creates a new Panel.
func New(theme *uxtypes.Theme) Panel {
	return Panel{Base: panels.Base{Theme: theme}}
}

func (p *Panel) SetInfo(info *models.RepoInfo) {
	p.Info = info
	p.ResetScroll()
}

func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 1, 1)
}

func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if p.Info.IsEmpty() {
		return nil
	}

	if p.NavigateKey(km, len(p.Info.Branches)) {
		p.ClampScroll(p.Height)
	}

	return nil
}

const (
	hashW   = 10
	gap     = 2
	gaps    = 3 * gap // 2+2+2 gaps
	minName = 12
	reserve = 10
)

func (p *Panel) View() string {
	if p.Info.IsEmpty() {
		return lipgloss.NewStyle().Foreground(p.Theme.Dim).
			Render("  Select a repository to view its branches.")
	}

	if err := p.Info.RepoErr(); err != nil {
		warnStyle := lipgloss.NewStyle().Foreground(p.Theme.Warning)

		return warnStyle.Render(fmt.Sprintf("  Error: %v", err))
	}

	branches := p.Info.Branches
	if len(branches) == 0 {
		return lipgloss.NewStyle().Foreground(p.Theme.Dim).
			Render("  No branches found.")
	}

	// Styles.
	t := p.Theme
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Bright).Background(t.SelectedBg)
	currentStyle := lipgloss.NewStyle().Foreground(t.Success)
	localStyle := lipgloss.NewStyle().Foreground(t.Text)
	remoteStyle := lipgloss.NewStyle().Foreground(t.Dim)
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)

	// Compute column widths.
	nameW := minName
	for _, b := range branches {
		if len(b.Name)+gap > nameW {
			nameW = len(b.Name) + gap
		}
	}

	maxNameW := max(p.Width-hashW-gaps-reserve, minName)

	if nameW > maxNameW {
		nameW = maxNameW
	}

	// Header.
	nameCol := lipgloss.NewStyle().Width(nameW)
	hashCol := lipgloss.NewStyle().Width(hashW)

	header := fmt.Sprintf("  %s  %s  %s",
		nameCol.Render(headerStyle.Render("Branch")),
		hashCol.Render(headerStyle.Render("Hash")),
		headerStyle.Render("Upstream"),
	)

	// Rows.
	start, end := p.VisibleRange(len(branches), p.Height)

	var rows []string

	for i := start; i < end; i++ {
		b := branches[i]
		row := p.renderBranch(b, nameW, nameCol, hashCol, currentStyle, localStyle, remoteStyle, dimStyle)

		if i == p.Cursor {
			row = selectedStyle.Render(row)
		}

		rows = append(rows, row)
	}

	rows = panels.PadRows(rows, p.Height)

	return header + "\n" + strings.Join(rows, "\n")
}

func (p *Panel) renderBranch(
	b models.Branch,
	nameW int,
	nameCol, hashCol lipgloss.Style,
	currentStyle, localStyle, remoteStyle, dimStyle lipgloss.Style,
) string {
	marker := "  "
	nameStyle := localStyle

	if b.IsCurrent {
		marker = "* "
		nameStyle = currentStyle
	} else if b.IsRemote {
		nameStyle = remoteStyle
	}

	name := marker + b.Name
	if len(name) > nameW {
		name = name[:nameW-1] + "\u2026"
	}

	hash := b.Hash
	if len(hash) > hashW {
		hash = hash[:hashW]
	}

	upstream := b.Upstream
	if upstream == "" {
		upstream = dimStyle.Render("-")
	} else {
		// Truncate upstream to remaining width.
		maxUpstream := p.Width - nameW - hashW - gaps
		if maxUpstream > 0 && len(upstream) > maxUpstream {
			upstream = upstream[:maxUpstream-1] + "\u2026"
		}
	}

	return fmt.Sprintf("%s  %s  %s",
		nameCol.Render(nameStyle.Render(name)),
		hashCol.Render(dimStyle.Render(hash)),
		upstream,
	)
}

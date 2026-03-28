package branches

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/git"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// BranchesPanel displays the branches for the selected repository.
type BranchesPanel struct {
	Info   *git.RepoInfo
	Cursor int
	Offset int // scroll offset
	Width  int
	Height int
}

// New creates a new BranchesPanel.
func New() BranchesPanel {
	return BranchesPanel{}
}

func (p *BranchesPanel) SetInfo(info *git.RepoInfo) {
	p.Info = info
	p.Cursor = 0
	p.Offset = 0
}

func (p *BranchesPanel) SetSize(w, h int) {
	p.Width = w
	p.Height = h - 1 // reserve 1 line for header
	if p.Height < 1 {
		p.Height = 1
	}
}

func (p *BranchesPanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if p.Info == nil {
		return nil
	}

	n := len(p.Info.Branches)

	switch km.String() {
	case "up", "k":
		if p.Cursor > 0 {
			p.Cursor--
			p.clampScroll()
		}
	case "down", "j":
		if p.Cursor < n-1 {
			p.Cursor++
			p.clampScroll()
		}
	case "home", "g":
		p.Cursor = 0
		p.clampScroll()
	case "end", "G":
		p.Cursor = max(0, n-1)
		p.clampScroll()
	}

	return nil
}

func (p *BranchesPanel) clampScroll() {
	if p.Cursor < p.Offset {
		p.Offset = p.Cursor
	}

	if p.Cursor >= p.Offset+p.Height {
		p.Offset = p.Cursor - p.Height + 1
	}
}

func (p *BranchesPanel) View() string {
	if p.Info == nil {
		return lipgloss.NewStyle().Foreground(types.CurrentTheme.Dim).
			Render("  Select a repository to view its branches.")
	}

	if p.Info.Err != nil {
		warnStyle := lipgloss.NewStyle().Foreground(types.CurrentTheme.Warning)

		return warnStyle.Render(fmt.Sprintf("  Error: %v", p.Info.Err))
	}

	branches := p.Info.Branches
	if len(branches) == 0 {
		return lipgloss.NewStyle().Foreground(types.CurrentTheme.Dim).
			Render("  No branches found.")
	}

	// Styles.
	t := types.CurrentTheme
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Bright).Background(t.SelectedBg)
	currentStyle := lipgloss.NewStyle().Foreground(t.Success)
	localStyle := lipgloss.NewStyle().Foreground(t.Text)
	remoteStyle := lipgloss.NewStyle().Foreground(t.Dim)
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)

	// Compute column widths.
	// Layout: 2 (marker) + nameW + 2 (gap) + 10 (hash) + 2 (gap) + upstream
	// Reserve at least 10 chars for upstream display.
	const hashW = 10
	const gaps = 6  // 2+2+2 gaps
	const minName = 12

	nameW := minName
	for _, b := range branches {
		if len(b.Name)+2 > nameW {
			nameW = len(b.Name) + 2
		}
	}

	maxNameW := p.Width - hashW - gaps - 10
	if maxNameW < minName {
		maxNameW = minName
	}

	if nameW > maxNameW {
		nameW = maxNameW
	}

	// Header.
	nameCol := lipgloss.NewStyle().Width(nameW)
	hashCol := lipgloss.NewStyle().Width(10)

	header := fmt.Sprintf("  %s  %s  %s",
		nameCol.Render(headerStyle.Render("Branch")),
		hashCol.Render(headerStyle.Render("Hash")),
		headerStyle.Render("Upstream"),
	)

	// Rows.
	end := p.Offset + p.Height
	if end > len(branches) {
		end = len(branches)
	}

	var rows []string

	for i := p.Offset; i < end; i++ {
		b := branches[i]
		row := p.renderBranch(b, nameW, nameCol, hashCol, currentStyle, localStyle, remoteStyle, dimStyle)

		if i == p.Cursor {
			row = selectedStyle.Render(row)
		}

		rows = append(rows, row)
	}

	// Pad.
	for len(rows) < p.Height {
		rows = append(rows, "")
	}

	return header + "\n" + strings.Join(rows, "\n")
}

func (p *BranchesPanel) renderBranch(
	b git.Branch,
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
	if len(hash) > 8 {
		hash = hash[:8]
	}

	upstream := b.Upstream
	if upstream == "" {
		upstream = dimStyle.Render("-")
	} else {
		// Truncate upstream to remaining width.
		maxUpstream := p.Width - nameW - 16 // hash(10) + gaps(6)
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

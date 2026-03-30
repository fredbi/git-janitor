package repos

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// RepoDelegate is a custom list delegate that highlights non-git items
// using the current theme's NotGit color.
type RepoDelegate struct {
	Base list.DefaultDelegate
}

// NewRepoDelegate creates a RepoDelegate styled with the current theme.
func NewRepoDelegate() RepoDelegate {
	t := uxtypes.CurrentTheme
	base := list.NewDefaultDelegate()
	base.Styles.SelectedTitle = base.Styles.SelectedTitle.
		Foreground(t.Accent).
		BorderLeftForeground(t.Accent)
	base.Styles.SelectedDesc = base.Styles.SelectedDesc.
		Foreground(t.Dim).
		BorderLeftForeground(t.Accent)

	return RepoDelegate{Base: base}
}

func (d RepoDelegate) Height() int                               { return d.Base.Height() }
func (d RepoDelegate) Spacing() int                              { return d.Base.Spacing() }
func (d RepoDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return d.Base.Update(msg, m) }

func (d RepoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	repo, ok := item.(models.RepoItem)
	if !ok || repo.IsGit {
		d.Base.Render(w, m, index, item)

		return
	}

	t := uxtypes.CurrentTheme
	isSelected := index == m.Index()

	titleStyle := lipgloss.NewStyle().
		Foreground(t.NotGit).
		Padding(0, 0, 0, 2)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		Padding(0, 0, 0, 2)

	if isSelected {
		titleStyle = titleStyle.
			Bold(true).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(t.NotGit).
			Padding(0, 0, 0, 1)

		descStyle = descStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(t.NotGit).
			Padding(0, 0, 0, 1)
	}

	title := titleStyle.Render(repo.Title())
	desc := descStyle.Render(repo.Description())

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

package repos

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// indentStep is the number of spaces a single level of nesting adds to
// the rendered row, both for namespace headers and for repo entries
// living below them.
const indentStep = 2

// RepoDelegate is a custom list delegate that highlights non-git items
// using the current theme's NotGit color and renders namespace headers
// and indented repo rows for nested layouts.
type RepoDelegate struct {
	Theme *uxtypes.Theme
	Base  list.DefaultDelegate
}

// NewRepoDelegate creates a RepoDelegate styled with the current theme.
func NewRepoDelegate(theme *uxtypes.Theme) RepoDelegate {
	t := theme
	base := list.NewDefaultDelegate()
	base.Styles.SelectedTitle = base.Styles.SelectedTitle.
		Foreground(t.Accent).
		BorderLeftForeground(t.Accent)
	base.Styles.SelectedDesc = base.Styles.SelectedDesc.
		Foreground(t.Dim).
		BorderLeftForeground(t.Accent)

	return RepoDelegate{Theme: theme, Base: base}
}

func (d RepoDelegate) Height() int                               { return d.Base.Height() }
func (d RepoDelegate) Spacing() int                              { return d.Base.Spacing() }
func (d RepoDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return d.Base.Update(msg, m) }

func (d RepoDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	switch v := item.(type) {
	case groupHeaderItem:
		d.renderHeader(w, v)

		return

	case models.RepoItem:
		if v.Namespace == "" && v.IsGit {
			// Top-level git repo: keep the bubbles default rendering so
			// pre-nesting users see the same panel they're used to.
			d.Base.Render(w, m, index, item)

			return
		}

		d.renderRepo(w, m, index, v)

		return
	}
}

// renderHeader paints a non-selectable namespace header row in the dim
// theme color, indented by the header's depth.
func (d RepoDelegate) renderHeader(w io.Writer, h groupHeaderItem) {
	indent := strings.Repeat(" ", h.Depth*indentStep)
	style := lipgloss.NewStyle().
		Foreground(d.Theme.Dim).
		Padding(0, 0, 0, indentStep)

	fmt.Fprintf(w, "%s\n", style.Render(indent+h.Name+"/"))
}

// renderRepo paints a repository row, indenting it by its [models.RepoItem.Depth]
// so it visually nests under its namespace headers.
func (d RepoDelegate) renderRepo(w io.Writer, m list.Model, index int, repo models.RepoItem) {
	t := d.Theme
	isSelected := index == m.Index()

	titleColor := t.Text
	if !repo.IsGit {
		titleColor = t.NotGit
	}

	leftPad := indentStep + repo.Depth()*indentStep

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Padding(0, 0, 0, leftPad)

	descStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		Padding(0, 0, 0, leftPad)

	if isSelected {
		borderColor := t.Accent
		if !repo.IsGit {
			borderColor = t.NotGit
		}

		titleStyle = titleStyle.
			Bold(true).
			Foreground(borderColor).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(borderColor).
			Padding(0, 0, 0, leftPad-1)

		descStyle = descStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(borderColor).
			Padding(0, 0, 0, leftPad-1)
	}

	title := titleStyle.Render(repo.Title())
	desc := descStyle.Render(repo.Description())

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

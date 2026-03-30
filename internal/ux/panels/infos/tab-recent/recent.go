package recent

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// item represents a recent activity entry in the list.
type item struct {
	title string
	desc  string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Panel wraps a bubbles/list for the recent-activity tab.
type Panel struct {
	List list.Model
}

// New creates a new Panel with sample entries.
func New() Panel {
	items := []list.Item{
		item{title: "Deleted feature/old-login", desc: "2 minutes ago — stale branch removed"},
		item{title: "Ran git gc", desc: "10 minutes ago — freed 24 MB"},
		item{title: "Pruned origin/stale-ref", desc: "1 hour ago — remote ref cleaned"},
		item{title: "Deleted fix/typo-readme", desc: "1 hour ago — merged branch removed"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("141")).
		BorderLeftForeground(lipgloss.Color("141"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("241")).
		BorderLeftForeground(lipgloss.Color("141"))

	l := list.New(items, delegate, 0, 0)
	l.Title = ""
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)

	return Panel{List: l}
}

func (p *Panel) SetSize(w, h int) {
	p.List.SetSize(w, h)
}

func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.List, cmd = p.List.Update(msg)

	return cmd
}

func (p *Panel) View() string {
	return p.List.View()
}

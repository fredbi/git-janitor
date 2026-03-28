package recent

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RecentItem represents a recent activity entry in the list.
type RecentItem struct {
	title string
	desc  string
}

func (i RecentItem) Title() string       { return i.title }
func (i RecentItem) Description() string { return i.desc }
func (i RecentItem) FilterValue() string { return i.title }

// RecentPanel wraps a bubbles/list for the recent-activity tab.
type RecentPanel struct {
	List list.Model
}

// New creates a new RecentPanel with sample entries.
func New() RecentPanel {
	items := []list.Item{
		RecentItem{title: "Deleted feature/old-login", desc: "2 minutes ago — stale branch removed"},
		RecentItem{title: "Ran git gc", desc: "10 minutes ago — freed 24 MB"},
		RecentItem{title: "Pruned origin/stale-ref", desc: "1 hour ago — remote ref cleaned"},
		RecentItem{title: "Deleted fix/typo-readme", desc: "1 hour ago — merged branch removed"},
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

	return RecentPanel{List: l}
}

func (p *RecentPanel) SetSize(w, h int) {
	p.List.SetSize(w, h)
}

func (p *RecentPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.List, cmd = p.List.Update(msg)

	return cmd
}

func (p *RecentPanel) View() string {
	return p.List.View()
}

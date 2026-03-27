package state

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// recentItem represents a recent activity entry in the list.
type recentItem struct {
	title string
	desc  string
}

func (i recentItem) Title() string       { return i.title }
func (i recentItem) Description() string { return i.desc }
func (i recentItem) FilterValue() string { return i.title }

// recentPanel wraps a bubbles/list for the recent-activity tab.
type recentPanel struct {
	list list.Model
}

func newRecentPanel() recentPanel {
	items := []list.Item{
		recentItem{title: "Deleted feature/old-login", desc: "2 minutes ago — stale branch removed"},
		recentItem{title: "Ran git gc", desc: "10 minutes ago — freed 24 MB"},
		recentItem{title: "Pruned origin/stale-ref", desc: "1 hour ago — remote ref cleaned"},
		recentItem{title: "Deleted fix/typo-readme", desc: "1 hour ago — merged branch removed"},
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

	return recentPanel{list: l}
}

func (p *recentPanel) SetSize(w, h int) {
	p.list.SetSize(w, h)
}

func (p *recentPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)

	return cmd
}

func (p *recentPanel) View() string {
	return p.list.View()
}

package state

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// actionItem represents an action entry in the list.
type actionItem struct {
	title string
	desc  string
}

func (i actionItem) Title() string       { return i.title }
func (i actionItem) Description() string { return i.desc }
func (i actionItem) FilterValue() string { return i.title }

// actionsListPanel wraps a bubbles/list for the actions tab.
type actionsListPanel struct {
	list list.Model
}

func newActionsListPanel() actionsListPanel {
	items := []list.Item{
		actionItem{title: "Clean stale branches", desc: "Delete branches older than 90 days"},
		actionItem{title: "Remove merged branches", desc: "Delete branches merged into main"},
		actionItem{title: "Run git gc", desc: "Optimize repository storage"},
		actionItem{title: "Prune remotes", desc: "Remove stale remote-tracking refs"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("36")).
		BorderLeftForeground(lipgloss.Color("36"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("241")).
		BorderLeftForeground(lipgloss.Color("36"))

	l := list.New(items, delegate, 0, 0)
	l.Title = ""
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)

	return actionsListPanel{list: l}
}

func (p *actionsListPanel) SetSize(w, h int) {
	p.list.SetSize(w, h)
}

func (p *actionsListPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)

	return cmd
}

func (p *actionsListPanel) View() string {
	return p.list.View()
}

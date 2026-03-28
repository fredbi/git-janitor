package actions

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ActionItem represents an action entry in the list.
type ActionItem struct {
	title string
	desc  string
}

func (i ActionItem) Title() string       { return i.title }
func (i ActionItem) Description() string { return i.desc }
func (i ActionItem) FilterValue() string { return i.title }

// ActionsListPanel wraps a bubbles/list for the actions tab.
type ActionsListPanel struct {
	List list.Model
}

// New creates a new ActionsListPanel with sample entries.
func New() ActionsListPanel {
	items := []list.Item{
		ActionItem{title: "Clean stale branches", desc: "Delete branches older than 90 days"},
		ActionItem{title: "Remove merged branches", desc: "Delete branches merged into main"},
		ActionItem{title: "Run git gc", desc: "Optimize repository storage"},
		ActionItem{title: "Prune remotes", desc: "Remove stale remote-tracking refs"},
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

	return ActionsListPanel{List: l}
}

func (p *ActionsListPanel) SetSize(w, h int) {
	p.List.SetSize(w, h)
}

func (p *ActionsListPanel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.List, cmd = p.List.Update(msg)

	return cmd
}

func (p *ActionsListPanel) View() string {
	return p.List.View()
}

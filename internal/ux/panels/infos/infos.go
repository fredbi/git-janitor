package infos

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/git"
	actions "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-actions"
	alerts "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-alerts"
	branches "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-branches"
	facts "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-facts"
	recent "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-recent"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// RightTab identifies which tab is active in the right Pane.
type RightTab int

const (
	TabFacts    RightTab = iota // facts tab (repo properties)
	TabBranches                 // branches tab
	TabAlerts                   // alerts tab
	TabActions                  // actions tab
	TabRecent                   // recent activity tab
)

// RightTabCount is the number of tabs in the right panel.
const RightTabCount = 5

// Panel is a tab container for the right Pane.
type Panel struct {
	Facts    facts.FactsPanel
	Branches branches.BranchesPanel
	Alerts   alerts.AlertsPanel
	Actions  actions.ActionsListPanel
	Recent   recent.RecentPanel
	Active   RightTab
	Width    int
	Height   int
}

// New creates a new Panel with default tab sub-panels.
func New() Panel {
	return Panel{
		Facts:    facts.New(),
		Branches: branches.New(),
		Alerts:   alerts.New(),
		Actions:  actions.New(),
		Recent:   recent.New(),
		Active:   TabFacts,
	}
}

// CycleTab advances the active tab forward by one.
func (p *Panel) CycleTab() {
	p.Active = RightTab((int(p.Active) + 1) % RightTabCount)
}

// CycleTabBack moves the active tab backward by one.
func (p *Panel) CycleTabBack() {
	p.Active = RightTab((int(p.Active) - 1 + RightTabCount) % RightTabCount)
}

// SetTab sets the active tab directly.
func (p *Panel) SetTab(t RightTab) {
	if t >= 0 && t < RightTabCount {
		p.Active = t
	}
}

// RightTabDefs defines the ordered tab labels and IDs.
var RightTabDefs = []struct { //nolint:gochecknoglobals // tab definition table
	Label string
	ID    RightTab
}{
	{"Facts", TabFacts},
	{"Branches", TabBranches},
	{"Alerts", TabAlerts},
	{"Actions", TabActions},
	{"Recent", TabRecent},
}

// SetRepoInfo updates the facts and branches panels with new repo data.
func (p *Panel) SetRepoInfo(info *git.RepoInfo) {
	p.Facts.SetInfo(info)
	p.Branches.SetInfo(info)
}

// TabAtX returns the tab index for a click at the given x offset
// relative to the right panel's inner content area (inside the border).
// Returns -1 if the click doesn't land on a tab label.
func (p *Panel) TabAtX(x int) RightTab {
	cursor := 1 // 1 char left padding from tabBarStyle.PaddingLeft(1)
	for _, t := range RightTabDefs {
		w := len(t.Label) + 2 // +2 for Padding(0,1) on each side
		if x >= cursor && x < cursor+w {
			return t.ID
		}

		cursor += w
	}

	return -1
}

// ActiveTabName returns the label of the currently active tab.
func (p *Panel) ActiveTabName() string {
	for _, t := range RightTabDefs {
		if t.ID == p.Active {
			return t.Label
		}
	}

	return ""
}

// SetSize updates the panel dimensions and propagates to sub-panels.
func (p *Panel) SetSize(w, h int) {
	p.Width = w
	p.Height = h

	// Reserve 2 lines for the tab bar + 2 for border.
	contentW := w - 2
	contentH := h - 4

	if contentH < 1 {
		contentH = 1
	}

	p.Facts.SetSize(contentW, contentH)
	p.Branches.SetSize(contentW, contentH)
	p.Alerts.SetSize(contentW, contentH)
	p.Actions.SetSize(contentW, contentH)
	p.Recent.SetSize(contentW, contentH)
}

// Update forwards the message to the currently active tab sub-panel.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	switch p.Active {
	case TabFacts:
		return p.Facts.Update(msg)
	case TabBranches:
		return p.Branches.Update(msg)
	case TabAlerts:
		return p.Alerts.Update(msg)
	case TabActions:
		return p.Actions.Update(msg)
	case TabRecent:
		return p.Recent.Update(msg)
	}

	return nil
}

// View renders the right panel with its tab bar and active content.
func (p *Panel) View(focused bool) string {
	// --- Tab bar ---
	t := uxtypes.CurrentTheme
	tabBarStyle := lipgloss.NewStyle().PaddingLeft(1)

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Bright).
		Background(t.Secondary).
		Padding(0, 1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		Padding(0, 1)

	var renderedTabs []string
	for _, t := range RightTabDefs {
		if t.ID == p.Active {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(t.Label))
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(t.Label))
		}
	}

	tabBar := tabBarStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...),
	)

	// --- Hint ---
	hintStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		PaddingLeft(1)
	hint := hintStyle.Render(fmt.Sprintf("Ctrl+A: switch tab (%d/%d)", int(p.Active)+1, RightTabCount))

	header := lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, "  ", hint)

	// --- Content ---
	var content string
	switch p.Active {
	case TabFacts:
		content = p.Facts.View()
	case TabBranches:
		content = p.Branches.View()
	case TabAlerts:
		content = p.Alerts.View()
	case TabActions:
		content = p.Actions.View()
	case TabRecent:
		content = p.Recent.View()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// --- Border ---
	borderColor := t.Dim
	if focused {
		borderColor = t.Secondary
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(p.Width - 2).
		Height(p.Height - 2)

	return border.Render(inner)
}

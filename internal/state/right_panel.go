package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// rightTab identifies which tab is active in the right pane.
type rightTab int

const (
	tabAlerts  rightTab = iota // alerts tab (default, on top)
	tabActions                 // actions tab
	tabRecent                  // recent activity tab
)

const rightTabCount = 3

// rightPanel is a tab container for the right pane, holding Alerts, Actions, and Recent.
type rightPanel struct {
	alerts  alertsPanel
	actions actionsListPanel
	recent  recentPanel
	active  rightTab
	width   int
	height  int
}

func newRightPanel() rightPanel {
	return rightPanel{
		alerts:  newAlertsPanel(),
		actions: newActionsListPanel(),
		recent:  newRecentPanel(),
		active:  tabAlerts,
	}
}

// CycleTab advances the active tab forward by one.
func (p *rightPanel) CycleTab() {
	p.active = rightTab((int(p.active) + 1) % rightTabCount)
}

// CycleTabBack moves the active tab backward by one.
func (p *rightPanel) CycleTabBack() {
	p.active = rightTab((int(p.active) - 1 + rightTabCount) % rightTabCount)
}

// SetTab sets the active tab directly.
func (p *rightPanel) SetTab(t rightTab) {
	if t >= 0 && t < rightTabCount {
		p.active = t
	}
}

// TabAtX returns the tab index for a click at the given x offset
// relative to the right panel's inner content area (inside the border).
// Returns -1 if the click doesn't land on a tab label.
func (p *rightPanel) TabAtX(x int) rightTab {
	// Tab bar layout: 1 char left padding, then each tab label
	// with 1 char padding on each side (Padding(0,1)).
	// Labels: "Alerts" (6), "Actions" (7), "Recent" (6)
	// Each rendered tab: padding(1) + label + padding(1).
	tabs := []struct {
		label string
		id    rightTab
	}{
		{"Alerts", tabAlerts},
		{"Actions", tabActions},
		{"Recent", tabRecent},
	}

	cursor := 1 // 1 char left padding from tabBarStyle.PaddingLeft(1)
	for _, t := range tabs {
		w := len(t.label) + 2 // +2 for Padding(0,1) on each side
		if x >= cursor && x < cursor+w {
			return t.id
		}

		cursor += w
	}

	return -1
}
func (p *rightPanel) ActiveTabName() string {
	switch p.active {
	case tabAlerts:
		return "Alerts"
	case tabActions:
		return "Actions"
	case tabRecent:
		return "Recent"
	default:
		return ""
	}
}

func (p *rightPanel) SetSize(w, h int) {
	p.width = w
	p.height = h

	// Reserve 2 lines for the tab bar + 2 for border.
	contentW := w - 2
	contentH := h - 4

	if contentH < 1 {
		contentH = 1
	}

	p.alerts.SetSize(contentW, contentH)
	p.actions.SetSize(contentW, contentH)
	p.recent.SetSize(contentW, contentH)
}

func (p *rightPanel) Update(msg tea.Msg) tea.Cmd {
	switch p.active {
	case tabAlerts:
		return p.alerts.Update(msg)
	case tabActions:
		return p.actions.Update(msg)
	case tabRecent:
		return p.recent.Update(msg)
	}

	return nil
}

func (p *rightPanel) View(focused bool) string {
	// --- Tab bar ---
	tabBarStyle := lipgloss.NewStyle().PaddingLeft(1)

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	tabs := []struct {
		label string
		id    rightTab
	}{
		{"Alerts", tabAlerts},
		{"Actions", tabActions},
		{"Recent", tabRecent},
	}

	var renderedTabs []string
	for _, t := range tabs {
		if t.id == p.active {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(t.label))
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(t.label))
		}
	}

	tabBar := tabBarStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...),
	)

	// --- Hint ---
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingLeft(1)
	hint := hintStyle.Render(fmt.Sprintf("Ctrl+A: switch tab (%d/%d)", int(p.active)+1, rightTabCount))

	header := lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, "  ", hint)

	// --- Content ---
	var content string
	switch p.active {
	case tabAlerts:
		content = p.alerts.View()
	case tabActions:
		content = p.actions.View()
	case tabRecent:
		content = p.recent.View()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// --- Border ---
	borderColor := lipgloss.Color("241")
	if focused {
		borderColor = lipgloss.Color("63")
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(p.width - 2).
		Height(p.height - 2)

	return border.Render(inner)
}

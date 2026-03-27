package state

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
)

// repoItem represents a repository entry in the list.
type repoItem struct {
	path string
	name string
}

func (i repoItem) Title() string       { return i.name }
func (i repoItem) Description() string { return i.path }
func (i repoItem) FilterValue() string { return i.name }

// rootTab holds one tab's state: the root's display name and its repo list.
type rootTab struct {
	name string
	list list.Model
}

// reposPanel is a tabbed left panel where each tab represents a configured root.
type reposPanel struct {
	tabs   []rootTab
	active int // index of the active tab
	width  int
	height int

	// visibleStart tracks the first visible tab index for elision.
	visibleStart int
}

func newReposPanel(cfg *config.Config) reposPanel {
	p := reposPanel{}
	p.rebuildTabs(cfg)

	return p
}

// rebuildTabs recreates tabs from the current config.
func (p *reposPanel) rebuildTabs(cfg *config.Config) {
	if cfg == nil || len(cfg.Roots) == 0 {
		p.tabs = []rootTab{p.makeTab("(no roots)")}
		p.active = 0
		p.visibleStart = 0

		return
	}

	tabs := make([]rootTab, len(cfg.Roots))
	for i := range cfg.Roots {
		tabs[i] = p.makeTab(cfg.RootDisplayName(i))
	}

	p.tabs = tabs

	// Clamp active tab.
	if p.active >= len(p.tabs) {
		p.active = len(p.tabs) - 1
	}

	if p.active < 0 {
		p.active = 0
	}

	p.clampVisibleStart()
}

func (p *reposPanel) makeTab(name string) rootTab {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("170")).
		BorderLeftForeground(lipgloss.Color("170"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("241")).
		BorderLeftForeground(lipgloss.Color("170"))

	l := list.New(nil, delegate, 0, 0)
	l.Title = name
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		PaddingLeft(1)

	if p.width > 0 && p.height > 0 {
		l.SetSize(p.width-2, p.height-4) // border + tab bar
	}

	return rootTab{name: name, list: l}
}

// TabCount returns the number of tabs.
func (p *reposPanel) TabCount() int { return len(p.tabs) }

// CycleTab advances the active tab forward.
func (p *reposPanel) CycleTab() {
	if len(p.tabs) == 0 {
		return
	}

	p.active = (p.active + 1) % len(p.tabs)
	p.clampVisibleStart()
}

// CycleTabBack moves the active tab backward.
func (p *reposPanel) CycleTabBack() {
	if len(p.tabs) == 0 {
		return
	}

	p.active = (p.active - 1 + len(p.tabs)) % len(p.tabs)
	p.clampVisibleStart()
}

// SetTab sets the active tab directly.
func (p *reposPanel) SetTab(t int) {
	if t >= 0 && t < len(p.tabs) {
		p.active = t
		p.clampVisibleStart()
	}
}

// ActiveTabName returns the name of the active tab.
func (p *reposPanel) ActiveTabName() string {
	if p.active < 0 || p.active >= len(p.tabs) {
		return ""
	}

	return p.tabs[p.active].name
}

// SetSize adjusts dimensions for the panel and all tab lists.
func (p *reposPanel) SetSize(w, h int) {
	p.width = w
	p.height = h

	listW := w - 2     // border
	listH := h - 2 - 2 // border + tab bar (2 lines)

	if listH < 1 {
		listH = 1
	}

	for i := range p.tabs {
		p.tabs[i].list.SetSize(listW, listH)
	}
}

// Update forwards messages to the active tab's list.
func (p *reposPanel) Update(msg tea.Msg) tea.Cmd {
	if p.active < 0 || p.active >= len(p.tabs) {
		return nil
	}

	var cmd tea.Cmd
	p.tabs[p.active].list, cmd = p.tabs[p.active].list.Update(msg)

	return cmd
}

// SetRootItems replaces the repo list for a specific root tab.
func (p *reposPanel) SetRootItems(rootIndex int, items []list.Item) tea.Cmd {
	if rootIndex < 0 || rootIndex >= len(p.tabs) {
		return nil
	}

	return p.tabs[rootIndex].list.SetItems(items)
}

// SelectedRepo returns the currently selected repository in the active tab.
func (p *reposPanel) SelectedRepo() (repoItem, bool) {
	if p.active < 0 || p.active >= len(p.tabs) {
		return repoItem{}, false
	}

	item := p.tabs[p.active].list.SelectedItem()
	if item == nil {
		return repoItem{}, false
	}

	repo, ok := item.(repoItem)

	return repo, ok
}

// maxVisibleTabs calculates how many tabs can be shown in the available width.
// Each tab takes: padding(1) + len(name) + padding(1).
// "..." takes 5 chars (with padding).
func (p *reposPanel) maxVisibleTabs() int {
	avail := p.width - 4 // border (2) + left padding (1) + margin (1)
	if avail <= 0 {
		return 1
	}

	count := 0
	used := 0
	ellipsisW := 5 // " ... "

	for i := range p.tabs {
		tw := len(p.tabs[i].name) + 2 // padding(0,1) on each side
		needed := used + tw

		// If this isn't the last tab, reserve space for "..." in case we truncate.
		remaining := len(p.tabs) - (i + 1)
		if remaining > 0 && needed+ellipsisW > avail {
			break
		}

		if needed > avail {
			break
		}

		used = needed
		count++
	}

	if count == 0 {
		count = 1
	}

	return count
}

// clampVisibleStart ensures the active tab is visible.
func (p *reposPanel) clampVisibleStart() {
	maxVis := p.maxVisibleTabs()

	// Ensure active is within [visibleStart, visibleStart+maxVis).
	if p.active < p.visibleStart {
		p.visibleStart = p.active
	}

	if p.active >= p.visibleStart+maxVis {
		p.visibleStart = p.active - maxVis + 1
	}

	if p.visibleStart < 0 {
		p.visibleStart = 0
	}
}

// TabAtX returns the tab index for a click at the given x offset
// relative to the panel's inner content area (inside the border).
// Returns -1 if the click doesn't land on a tab label.
func (p *reposPanel) TabAtX(x int) int {
	maxVis := p.maxVisibleTabs()
	visEnd := p.visibleStart + maxVis
	if visEnd > len(p.tabs) {
		visEnd = len(p.tabs)
	}

	cursor := 1 // 1 char left padding

	// Leading "..." if visibleStart > 0.
	if p.visibleStart > 0 {
		w := 5 // " ... "
		if x >= cursor && x < cursor+w {
			// Click on leading ellipsis — navigate backward.
			return p.visibleStart - 1
		}

		cursor += w
	}

	for i := p.visibleStart; i < visEnd; i++ {
		w := len(p.tabs[i].name) + 2 // Padding(0,1)
		if x >= cursor && x < cursor+w {
			return i
		}

		cursor += w
	}

	// Trailing "..." if there are hidden tabs after.
	if visEnd < len(p.tabs) {
		w := 5
		if x >= cursor && x < cursor+w {
			return visEnd
		}
	}

	return -1
}

// View renders the tabbed repos panel.
func (p *reposPanel) View(focused bool) string {
	// --- Tab bar ---
	tabBar := p.renderTabBar()

	// --- Hint ---
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingLeft(1)
	hint := hintStyle.Render(fmt.Sprintf("Ctrl+A (%d/%d)", p.active+1, len(p.tabs)))

	header := lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, "  ", hint)

	// --- Content ---
	var content string
	if p.active >= 0 && p.active < len(p.tabs) {
		content = p.tabs[p.active].list.View()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// --- Border ---
	borderColor := lipgloss.Color("241")
	if focused {
		borderColor = lipgloss.Color("170")
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(p.width - 2).
		Height(p.height - 2)

	return border.Render(inner)
}

// renderTabBar builds the tab bar string with elision.
func (p *reposPanel) renderTabBar() string {
	tabBarStyle := lipgloss.NewStyle().PaddingLeft(1)

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("170")).
		Padding(0, 1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	ellipsisStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	maxVis := p.maxVisibleTabs()
	visEnd := p.visibleStart + maxVis
	if visEnd > len(p.tabs) {
		visEnd = len(p.tabs)
	}

	var parts []string

	// Leading "..." if there are hidden tabs before.
	if p.visibleStart > 0 {
		parts = append(parts, ellipsisStyle.Render("..."))
	}

	for i := p.visibleStart; i < visEnd; i++ {
		if i == p.active {
			parts = append(parts, activeTabStyle.Render(p.tabs[i].name))
		} else {
			parts = append(parts, inactiveTabStyle.Render(p.tabs[i].name))
		}
	}

	// Trailing "..." if there are hidden tabs after.
	if visEnd < len(p.tabs) {
		parts = append(parts, ellipsisStyle.Render("..."))
	}

	return tabBarStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Bottom, parts...),
	)
}

// repoItemsToListItems converts a slice of repoItem to a slice of list.Item.
func repoItemsToListItems(repos []repoItem) []list.Item {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = r
	}

	return items
}

// Package repos implements the tabbed left panel where each tab represents a configured root directory.
package repos

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// RootTab holds one tab's state: the root's display name and its repo list.
type RootTab struct {
	Name     string
	List     list.Model
	AllItems []list.Item // unfiltered items, kept for re-applying the filter
}

// Panel is a tabbed left panel where each tab represents a configured root.
type Panel struct {
	Tabs   []RootTab
	Active int // index of the active tab
	Width  int
	Height int

	// VisibleStart tracks the first visible tab index for elision.
	VisibleStart int

	// Filter is a text input for filtering repos by RE2 regexp.
	Filter    textinput.Model
	FilterRe  *regexp.Regexp // compiled filter (nil = show all)
	FilterErr bool           // true when the current input is not valid regexp
}

// New creates a new Panel from the given configuration.
func New(cfg *config.Config) Panel {
	fi := textinput.New()
	fi.Placeholder = "filter (regexp)"
	fi.Prompt = " 🔍 "
	fi.CharLimit = 128
	fi.Width = 40

	p := Panel{Filter: fi}
	p.RebuildTabs(cfg)

	return p
}

// RebuildTabs recreates tabs from the current config.
func (p *Panel) RebuildTabs(cfg *config.Config) {
	if cfg == nil || len(cfg.Roots) == 0 {
		p.Tabs = []RootTab{p.makeTab("(no roots)")}
		p.Active = 0
		p.VisibleStart = 0

		return
	}

	tabs := make([]RootTab, len(cfg.Roots))
	for i := range cfg.Roots {
		tabs[i] = p.makeTab(cfg.RootDisplayName(i))
	}

	p.Tabs = tabs

	// Clamp active tab.
	if p.Active >= len(p.Tabs) {
		p.Active = len(p.Tabs) - 1
	}

	if p.Active < 0 {
		p.Active = 0
	}

	p.clampVisibleStart()
}

func (p *Panel) makeTab(name string) RootTab {
	delegate := NewRepoDelegate()

	l := list.New(nil, delegate, 0, 0)
	l.Title = name
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		PaddingLeft(1)

	if p.Width > 0 && p.Height > 0 {
		l.SetSize(p.Width-2, p.Height-4) // border + tab bar
	}

	return RootTab{Name: name, List: l}
}

// RebuildDelegate recreates the list delegate with current theme colors.
func (p *Panel) RebuildDelegate() {
	d := NewRepoDelegate()
	for i := range p.Tabs {
		p.Tabs[i].List.SetDelegate(d)
	}
}

// TabCount returns the number of tabs.
func (p *Panel) TabCount() int { return len(p.Tabs) }

// CycleTab advances the active tab forward.
func (p *Panel) CycleTab() {
	if len(p.Tabs) == 0 {
		return
	}

	p.Active = (p.Active + 1) % len(p.Tabs)
	p.clampVisibleStart()
	p.refilterActive()
}

// CycleTabBack moves the active tab backward.
func (p *Panel) CycleTabBack() {
	if len(p.Tabs) == 0 {
		return
	}

	p.Active = (p.Active - 1 + len(p.Tabs)) % len(p.Tabs)
	p.clampVisibleStart()
	p.refilterActive()
}

// SetTab sets the active tab directly.
func (p *Panel) SetTab(t int) {
	if t >= 0 && t < len(p.Tabs) {
		p.Active = t
		p.clampVisibleStart()
		p.refilterActive()
	}
}

// ActiveTabName returns the name of the active tab.
func (p *Panel) ActiveTabName() string {
	if p.Active < 0 || p.Active >= len(p.Tabs) {
		return ""
	}

	return p.Tabs[p.Active].Name
}

// SetSize adjusts dimensions for the panel and all tab lists.
func (p *Panel) SetSize(w, h int) {
	p.Width = w
	p.Height = h

	listW := w - 2     // border
	listH := h - 2 - 3 // border + tab bar (2 lines) + filter row (1 line)

	if listH < 1 {
		listH = 1
	}

	p.Filter.Width = listW - 6 // account for prompt width

	for i := range p.Tabs {
		p.Tabs[i].List.SetSize(listW, listH)
	}
}

// Update forwards messages to the active tab's list or the filter input.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	if p.Active < 0 || p.Active >= len(p.Tabs) {
		return nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return p.handleKey(keyMsg)
	}

	// Non-key messages (e.g. blink) go to the filter input.
	var cmd tea.Cmd
	p.Filter, cmd = p.Filter.Update(msg)

	return cmd
}

func (p *Panel) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	switch key {
	case "up", "down", "k", "j", "pgup", "pgdown", "home", "end":
		// Navigation keys go to the list.
		var cmd tea.Cmd
		p.Tabs[p.Active].List, cmd = p.Tabs[p.Active].List.Update(msg)

		return cmd

	case "esc":
		// Clear the filter.
		if p.Filter.Value() != "" {
			p.Filter.SetValue("")
			p.recompileFilter()
			p.refilterActive()

			return nil
		}

		return nil

	default:
		// Everything else goes to the filter text input.
		prev := p.Filter.Value()

		var cmd tea.Cmd
		p.Filter, cmd = p.Filter.Update(msg)

		if p.Filter.Value() != prev {
			p.recompileFilter()
			p.refilterActive()
		}

		return cmd
	}
}

// Focus gives keyboard focus to the filter input.
func (p *Panel) Focus() tea.Cmd {
	return p.Filter.Focus()
}

// Blur removes keyboard focus from the filter input.
func (p *Panel) Blur() {
	p.Filter.Blur()
}

// recompileFilter parses the current filter text as a case-insensitive RE2 regexp.
func (p *Panel) recompileFilter() {
	raw := strings.TrimSpace(p.Filter.Value())
	if raw == "" {
		p.FilterRe = nil
		p.FilterErr = false

		return
	}

	re, err := regexp.Compile("(?i)" + raw)
	if err != nil {
		p.FilterRe = nil
		p.FilterErr = true

		return
	}

	p.FilterRe = re
	p.FilterErr = false
}

// applyFilter returns items matching the current filter regexp.
func (p *Panel) applyFilter(items []list.Item) []list.Item {
	if p.FilterRe == nil {
		return items
	}

	var filtered []list.Item

	for _, item := range items {
		repo, ok := item.(uxtypes.RepoItem)
		if !ok {
			continue
		}

		if p.FilterRe.MatchString(repo.Name) || p.FilterRe.MatchString(repo.Path) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// refilterActive re-applies the filter to the active tab.
func (p *Panel) refilterActive() {
	if p.Active < 0 || p.Active >= len(p.Tabs) {
		return
	}

	tab := &p.Tabs[p.Active]
	tab.List.SetItems(p.applyFilter(tab.AllItems))
}

// RefilterAll re-applies the filter to all tabs.
func (p *Panel) RefilterAll() {
	for i := range p.Tabs {
		p.Tabs[i].List.SetItems(p.applyFilter(p.Tabs[i].AllItems))
	}
}

// SetRootItems replaces the repo list for a specific root tab.
func (p *Panel) SetRootItems(rootIndex int, items []list.Item) tea.Cmd {
	if rootIndex < 0 || rootIndex >= len(p.Tabs) {
		return nil
	}

	p.Tabs[rootIndex].AllItems = items

	return p.Tabs[rootIndex].List.SetItems(p.applyFilter(items))
}

// SelectedRepo returns the currently selected repository in the active tab.
func (p *Panel) SelectedRepo() (uxtypes.RepoItem, bool) {
	if p.Active < 0 || p.Active >= len(p.Tabs) {
		return uxtypes.RepoItem{}, false
	}

	item := p.Tabs[p.Active].List.SelectedItem()
	if item == nil {
		return uxtypes.RepoItem{}, false
	}

	repo, ok := item.(uxtypes.RepoItem)

	return repo, ok
}

// maxVisibleTabs calculates how many tabs can be shown in the available width.
// Each tab takes: padding(1) + len(name) + padding(1).
// "..." takes 5 chars (with padding).
func (p *Panel) maxVisibleTabs() int {
	avail := p.Width - 4 // border (2) + left padding (1) + margin (1)
	if avail <= 0 {
		return 1
	}

	count := 0
	used := 0
	ellipsisW := 5 // " ... "

	for i := range p.Tabs {
		tw := len(p.Tabs[i].Name) + 2 // padding(0,1) on each side
		needed := used + tw

		// If this isn't the last tab, reserve space for "..." in case we truncate.
		remaining := len(p.Tabs) - (i + 1)
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
func (p *Panel) clampVisibleStart() {
	maxVis := p.maxVisibleTabs()

	// Ensure active is within [visibleStart, visibleStart+maxVis).
	if p.Active < p.VisibleStart {
		p.VisibleStart = p.Active
	}

	if p.Active >= p.VisibleStart+maxVis {
		p.VisibleStart = p.Active - maxVis + 1
	}

	if p.VisibleStart < 0 {
		p.VisibleStart = 0
	}
}

// TabAtX returns the tab index for a click at the given x offset
// relative to the panel's inner content area (inside the border).
// Returns -1 if the click doesn't land on a tab label.
func (p *Panel) TabAtX(x int) int {
	maxVis := p.maxVisibleTabs()
	visEnd := p.VisibleStart + maxVis
	if visEnd > len(p.Tabs) {
		visEnd = len(p.Tabs)
	}

	cursor := 1 // 1 char left padding

	// Leading "..." if visibleStart > 0.
	if p.VisibleStart > 0 {
		w := 5 // " ... "
		if x >= cursor && x < cursor+w {
			// Click on leading ellipsis — navigate backward.
			return p.VisibleStart - 1
		}

		cursor += w
	}

	for i := p.VisibleStart; i < visEnd; i++ {
		w := len(p.Tabs[i].Name) + 2 // Padding(0,1)
		if x >= cursor && x < cursor+w {
			return i
		}

		cursor += w
	}

	// Trailing "..." if there are hidden tabs after.
	if visEnd < len(p.Tabs) {
		w := 5
		if x >= cursor && x < cursor+w {
			return visEnd
		}
	}

	return -1
}

// View renders the tabbed repos panel.
func (p *Panel) View(focused bool) string {
	// --- Tab bar ---
	tabBar := p.renderTabBar()

	// --- Hint ---
	t := uxtypes.CurrentTheme
	hintStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		PaddingLeft(1)
	hint := hintStyle.Render(fmt.Sprintf("Ctrl+A (%d/%d)", p.Active+1, len(p.Tabs)))

	header := lipgloss.JoinHorizontal(lipgloss.Bottom, tabBar, "  ", hint)

	// --- Filter row ---
	filterRow := p.renderFilterRow()

	// --- Content ---
	var content string
	if p.Active >= 0 && p.Active < len(p.Tabs) {
		content = p.Tabs[p.Active].List.View()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, filterRow, content)

	// --- Border ---
	borderColor := t.Dim
	if focused {
		borderColor = t.Accent
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(p.Width - 2).
		Height(p.Height - 2)

	return border.Render(inner)
}

// renderFilterRow renders the filter input row.
func (p *Panel) renderFilterRow() string {
	row := p.Filter.View()

	if p.FilterErr {
		errHint := lipgloss.NewStyle().Foreground(uxtypes.CurrentTheme.Error).Render(" (invalid regexp)")
		row += errHint
	} else if p.FilterRe != nil && p.Active >= 0 && p.Active < len(p.Tabs) {
		total := len(p.Tabs[p.Active].AllItems)
		shown := len(p.Tabs[p.Active].List.Items())
		countHint := lipgloss.NewStyle().Foreground(uxtypes.CurrentTheme.Dim).
			Render(fmt.Sprintf(" %d/%d", shown, total))
		row += countHint
	}

	return row
}

// renderTabBar builds the tab bar string with elision.
func (p *Panel) renderTabBar() string {
	t := uxtypes.CurrentTheme
	tabBarStyle := lipgloss.NewStyle().PaddingLeft(1)

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Bright).
		Background(t.Accent).
		Padding(0, 1)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		Padding(0, 1)

	ellipsisStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		Padding(0, 1)

	maxVis := p.maxVisibleTabs()
	visEnd := p.VisibleStart + maxVis
	if visEnd > len(p.Tabs) {
		visEnd = len(p.Tabs)
	}

	var parts []string

	// Leading "..." if there are hidden tabs before.
	if p.VisibleStart > 0 {
		parts = append(parts, ellipsisStyle.Render("..."))
	}

	for i := p.VisibleStart; i < visEnd; i++ {
		if i == p.Active {
			parts = append(parts, activeTabStyle.Render(p.Tabs[i].Name))
		} else {
			parts = append(parts, inactiveTabStyle.Render(p.Tabs[i].Name))
		}
	}

	// Trailing "..." if there are hidden tabs after.
	if visEnd < len(p.Tabs) {
		parts = append(parts, ellipsisStyle.Render("..."))
	}

	return tabBarStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Bottom, parts...),
	)
}

// RepoItemsToListItems converts a slice of RepoItem to a slice of list.Item.
func RepoItemsToListItems(repos []uxtypes.RepoItem) []list.Item {
	items := make([]list.Item, len(repos))
	for i, r := range repos {
		items[i] = r
	}

	return items
}

package infos

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
	actions "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-actions"
	activity "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-activity"
	alerts "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-alerts"
	branches "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-branches"
	facts "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-facts"
	recent "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-recent"
	stashes "github.com/fredbi/git-janitor/internal/ux/panels/infos/tab-stashes"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

const recentHistoryWindow = 30 * 24 * time.Hour // 30 days

// RightTab identifies which tab is active in the right Pane.
type RightTab int

const (
	TabFacts    RightTab = iota // facts tab (repo properties)
	TabBranches                 // branches tab
	TabAlerts                   // alerts tab
	TabActions                  // actions tab
	TabActivity                 // activity tab (issues, PRs, workflow runs)
	TabStashes                  // stashes tab
	TabRecent                   // recent activity tab
)

// RightTabCount is the number of tabs in the right panel.
const RightTabCount = 7

// Panel is a tab container for the right Pane.
type Panel struct {
	Theme    *uxtypes.Theme
	Facts    facts.Panel
	Branches branches.Panel
	Stashes  stashes.Panel
	Alerts   alerts.Panel
	Actions  actions.Panel
	Activity activity.Panel
	Recent   recent.Panel
	Active   RightTab
	Width    int
	Height   int

	// Engine evaluates checks and produces alerts.
	Engine ifaces.Engineer

	// RepoPath is the path of the currently displayed repo (for action execution).
	RepoPath string

	// LastAlerts stores the most recent evaluation results for suggestion lookup.
	LastAlerts []models.Alert
}

// New creates a new Panel with default tab sub-panels.
func New(eng ifaces.Engineer, theme *uxtypes.Theme) Panel {
	return Panel{
		Theme:    theme,
		Facts:    facts.New(theme),
		Branches: branches.New(theme),
		Stashes:  stashes.New(theme),
		Alerts:   alerts.New(theme),
		Actions:  actions.New(eng, theme),
		Activity: activity.New(theme),
		Recent:   recent.New(theme),
		Active:   TabFacts,
		Engine:   eng,
	}
}

// SetTheme propagates a new theme to all sub-panels.
func (p *Panel) SetTheme(theme *uxtypes.Theme) {
	p.Theme = theme
	p.Facts.Theme = theme
	p.Branches.Theme = theme
	p.Stashes.Theme = theme
	p.Alerts.Theme = theme
	p.Actions.Theme = theme
	p.Activity.Theme = theme
	p.Recent.Theme = theme
}

// RefreshRecent updates the Recent tab with latest history for the current repo.
// Called after action execution to pick up the new entry.
func (p *Panel) RefreshRecent() {
	p.refreshRecent(p.RepoPath)
}

// SetActivityData updates the Activity panel with fetched data.
func (p *Panel) SetActivityData(info *models.RepoInfo) {
	p.Activity.SetData(info)
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
// TODO: unexport.
var RightTabDefs = []struct { //nolint:gochecknoglobals // tab definition table
	Label string
	ID    RightTab
}{
	{"Facts", TabFacts},
	{"Branches", TabBranches},
	{"Alerts", TabAlerts},
	{"Actions", TabActions},
	{"Activity", TabActivity},
	{"Stashes", TabStashes},
	{"Recent", TabRecent},
}

// SetRepoInfo updates all panels with new repo data.
// If an engine is configured, it evaluates checks and populates the alerts panel.
func (p *Panel) SetRepoInfo(info *models.RepoInfo) {
	p.RepoPath = info.Path

	if p.Engine != nil {
		p.Facts.GitHubEnabled = p.Engine.ProviderEnabled("github")
	}

	p.Facts.SetInfo(info)
	p.Branches.SetInfo(info)
	p.Stashes.SetInfo(info)
	p.Activity.Reset()
	p.Actions.Clear()
	p.LastAlerts = nil // clear previous alerts before re-evaluation

	if p.Engine == nil || info.IsEmpty() {
		p.Alerts.SetAlerts(nil)

		return
	}

	alertsIter, err := p.Engine.Evaluate(context.Background(), info)
	if err != nil {
		return // TODO: we should return an error
	}

	p.LastAlerts = slices.AppendSeq(p.LastAlerts, alertsIter)
	p.Alerts.SetAlerts(p.LastAlerts)

	// Populate Recent tab with history for this repo.
	p.refreshRecent(info.Path)
}

// SetGitHubData updates the panel with GitHub API data.
// It re-evaluates all checks (git + GitHub) so alerts reflect the
// complete picture, then updates the Facts panel with platform data.
func (p *Panel) SetGitHubData(info *models.RepoInfo) {
	p.Facts.SetGitHubData(info)

	if p.Engine == nil || info.RepoErr() != nil {
		return
	}

	// Re-evaluate all checks with the enriched info (now includes platform data).
	// The engine returns alerts sorted by severity.
	alertsIter, err := p.Engine.Evaluate(context.Background(), info)
	if err != nil {
		return // TODO: we should return an error
	}

	p.LastAlerts = slices.Collect(alertsIter)
	p.Alerts.SetAlerts(p.LastAlerts)
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

// IsCapturingInput reports whether a sub-panel is capturing keys that
// would otherwise be handled by the parent (text input, sub-tab navigation).
func (p *Panel) IsCapturingInput() bool {
	if p.Active == TabActions && p.Actions.IsCapturingInput() {
		return true
	}

	if p.Active == TabActivity && p.Activity.IsCapturingInput() {
		return true
	}

	return false
}

// SetSize updates the panel dimensions and propagates to sub-panels.
func (p *Panel) SetSize(w, h int) {
	p.Width = w
	p.Height = h

	// Reserve 2 lines for the tab bar + 2 for border.
	contentW := w - 2
	contentH := max(h-4, 1)

	p.Facts.SetSize(contentW, contentH)
	p.Branches.SetSize(contentW, contentH)
	p.Stashes.SetSize(contentW, contentH)
	p.Alerts.SetSize(contentW, contentH)
	p.Actions.SetSize(contentW, contentH)
	p.Activity.SetSize(contentW, contentH)
	p.Recent.SetSize(contentW, contentH)
}

// Update forwards the message to the currently active tab sub-panel.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	// Handle ShowSuggestionsMsg: switch to Actions tab and populate it.
	if sm, ok := msg.(uxtypes.ShowSuggestionsMsg); ok {
		if sm.AlertIndex >= 0 && sm.AlertIndex < len(p.Alerts.Alerts) {
			alert := p.Alerts.Alerts[sm.AlertIndex]
			p.Actions.SetAlert(p.RepoPath, &alert)
			p.Active = TabActions
		}

		return nil
	}

	switch p.Active {
	case TabFacts:
		return p.Facts.Update(msg)
	case TabBranches:
		return p.Branches.Update(msg)
	case TabStashes:
		return p.Stashes.Update(msg)
	case TabAlerts:
		return p.Alerts.Update(msg)
	case TabActions:
		return p.Actions.Update(msg)
	case TabActivity:
		return p.Activity.Update(msg)
	case TabRecent:
		return p.Recent.Update(msg)
	default:
		panic(errors.New("invalid tab"))
	}
}

// View renders the right panel with its tab bar and active content.
func (p *Panel) View(focused bool) string {
	// --- Tab bar ---
	t := p.Theme
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
	case TabStashes:
		content = p.Stashes.View()
	case TabAlerts:
		content = p.Alerts.View()
	case TabActions:
		content = p.Actions.View()
	case TabActivity:
		content = p.Activity.View()
	case TabRecent:
		content = p.Recent.View()
	default:
		panic(errors.New("invalid tab"))
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, content)

	// Truncate inner content to fit the border box.
	// Border uses 2 lines (top + bottom), header uses 1 line.
	maxInnerLines := p.Height - 2
	if maxInnerLines > 0 {
		inner = truncateLines(inner, maxInnerLines)
	}

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

func (p *Panel) refreshRecent(repoPath string) {
	if p.Engine == nil || repoPath == "" {
		p.Recent.SetHistory(nil)

		return
	}

	since := time.Now().Add(-recentHistoryWindow)
	entries := p.Engine.RecentHistory(repoPath, since)
	p.Recent.SetHistory(entries)
}

// truncateLines keeps at most maxLines lines from s.
func truncateLines(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}

	return strings.Join(lines[:maxLines], "\n")
}

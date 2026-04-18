// SPDX-License-Identifier: Apache-2.0

package activity

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	"github.com/fredbi/git-janitor/internal/ux/key"
	"github.com/fredbi/git-janitor/internal/ux/panels"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// SubTab identifies which sub-tab is active.
type SubTab int

const (
	SubTabIssues SubTab = iota
	SubTabPRs
	SubTabWorkflows
	subTabCount
)

var subTabLabels = [subTabCount]string{"Issues", "Pull Requests", "Workflows"} //nolint:gochecknoglobals // tab label table

// FocusLevel tracks whether the user is navigating sub-tabs or list items.
type FocusLevel int

const (
	FocusSubTab FocusLevel = iota // navigating the sub-tab bar
	FocusList                     // navigating the list content
)

// Panel displays GitHub activity data: issues, PRs, workflow runs.
type Panel struct {
	panels.Base

	ActiveSub SubTab
	Focus     FocusLevel
	Page      int // current page for the active sub-tab

	issues   []models.Issue
	prs      []models.PullRequest
	runs     []models.WorkflowRun
	fetched  [subTabCount]bool // track whether each sub-tab has been fetched
	hasMore  [subTabCount]bool // true until a fetch adds 0 new items
	prevLen  [subTabCount]int  // item count before last fetch
	fetching bool              // true while a fetch is in flight (prevent double-fetch)
}

// New creates a new Activity panel.
func New(theme *uxtypes.Theme) Panel {
	return Panel{
		Base: panels.Base{Theme: theme},
		Page: 1,
	}
}

// SetData updates the panel with activity data from PlatformInfo.
// Derives "has more" by comparing item counts before and after the update.
func (p *Panel) SetData(info *models.RepoInfo) {
	p.fetching = false

	if info == nil || info.Platform == nil {
		p.issues = nil
		p.prs = nil
		p.runs = nil

		return
	}

	platform := info.Platform
	p.issues = platform.Issues
	p.prs = platform.PullRequests
	p.runs = platform.WorkflowRuns

	// Derive hasMore: if the new count is greater than what we had before
	// the fetch, there might be more. If the count didn't change (or the
	// list is empty), we've reached the end.
	newLens := [subTabCount]int{len(p.issues), len(p.prs), len(p.runs)}
	for i := range subTabCount {
		if p.fetched[i] {
			p.hasMore[i] = newLens[i] > p.prevLen[i]
		}

		p.prevLen[i] = newLens[i]
	}
}

// Reset clears all data and focus when switching repos.
func (p *Panel) Reset() {
	p.issues = nil
	p.prs = nil
	p.runs = nil
	p.fetched = [subTabCount]bool{}
	p.hasMore = [subTabCount]bool{}
	p.prevLen = [subTabCount]int{}
	p.fetching = false
	p.Focus = FocusSubTab
	p.ActiveSub = SubTabIssues
	p.Page = 1
	p.ResetScroll()
}

// IsCapturingInput reports whether the panel is in list navigation mode
// and should capture arrow keys that would otherwise cycle top-level tabs.
func (p *Panel) IsCapturingInput() bool {
	return p.Focus == FocusList
}

// SetSize updates the panel dimensions.
func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 3, 1) // 3 reserved: sub-tab bar + hint + blank
}

// Update handles key messages for two-level navigation.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	if p.Focus == FocusSubTab {
		return p.updateSubTabNav(km)
	}

	return p.updateListNav(km)
}

func (p *Panel) updateSubTabNav(km tea.KeyMsg) tea.Cmd {
	switch key.MsgBinding(km) {
	case key.LeftArrow, key.H:
		p.ActiveSub = SubTab((int(p.ActiveSub) - 1 + int(subTabCount)) % int(subTabCount))
		p.ResetScroll()
	case key.RightArrow, key.L:
		p.ActiveSub = SubTab((int(p.ActiveSub) + 1) % int(subTabCount))
		p.ResetScroll()
	case key.Enter, key.Down, key.J:
		// Enter list navigation — trigger fetch if not yet done.
		p.Focus = FocusList
		p.ResetScroll()

		if !p.fetched[p.ActiveSub] {
			p.fetched[p.ActiveSub] = true
			p.hasMore[p.ActiveSub] = true // optimistic until proven otherwise
			p.Page = 1

			return p.fetchCmd()
		}
	}

	return nil
}

func (p *Panel) updateListNav(km tea.KeyMsg) tea.Cmd {
	switch key.MsgBinding(km) {
	case key.Esc:
		// Esc always returns to sub-tab navigation.
		p.Focus = FocusSubTab

		return nil

	case key.LeftArrow, key.H:
		// Cycle sub-tabs while staying in list mode.
		p.ActiveSub = SubTab((int(p.ActiveSub) - 1 + int(subTabCount)) % int(subTabCount))
		p.ResetScroll()

		if !p.fetched[p.ActiveSub] {
			p.fetched[p.ActiveSub] = true
			p.hasMore[p.ActiveSub] = true
			p.Page = 1

			return p.fetchCmd()
		}

		return nil

	case key.RightArrow, key.L:
		// Cycle sub-tabs while staying in list mode.
		p.ActiveSub = SubTab((int(p.ActiveSub) + 1) % int(subTabCount))
		p.ResetScroll()

		if !p.fetched[p.ActiveSub] {
			p.fetched[p.ActiveSub] = true
			p.hasMore[p.ActiveSub] = true
			p.Page = 1

			return p.fetchCmd()
		}

		return nil

	case key.Up, key.K:
		if p.Cursor > 0 {
			p.Cursor--
			p.ClampScroll(p.visibleCards())
		}

		return nil

	case key.Down, key.J:
		itemCount := p.activeItemCount()
		if p.Cursor < itemCount-1 {
			p.Cursor++
			p.ClampScroll(p.visibleCards())
		} else if p.hasMore[p.ActiveSub] && !p.fetching {
			// At the bottom with more pages — fetch next page.
			p.fetching = true
			p.Page++

			return p.fetchCmd()
		}

		return nil

	case key.PageUp:
		p.Cursor = max(0, p.Cursor-p.visibleCards())
		p.ClampScroll(p.visibleCards())

		return nil

	case key.PageDown:
		itemCount := p.activeItemCount()
		newCursor := min(itemCount-1, p.Cursor+p.visibleCards())
		p.Cursor = newCursor
		p.ClampScroll(p.visibleCards())

		// If we landed at the bottom and there are more pages, fetch.
		if p.Cursor >= itemCount-1 && p.hasMore[p.ActiveSub] && !p.fetching {
			p.fetching = true
			p.Page++

			return p.fetchCmd()
		}

		return nil

	case key.Home, key.G:
		p.Cursor = 0
		p.ClampScroll(p.visibleCards())

		return nil

	case key.End, key.GG:
		p.Cursor = max(0, p.activeItemCount()-1)
		p.ClampScroll(p.visibleCards())

		return nil

	case key.Enter:
		// Show detail popup for the selected item.
		return p.detailCmd()
	}

	return nil
}

// View renders the activity panel.
func (p *Panel) View() string {
	t := p.Theme
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)

	// Sub-tab bar.
	subTabBar := p.renderSubTabBar()

	// Content.
	var content string

	if p.Focus == FocusSubTab {
		content = dimStyle.Render("  Press Enter or ↓ to load data")
	} else {
		content = p.renderList()
	}

	// Hint.
	var hint string
	if p.Focus == FocusSubTab {
		hint = dimStyle.Render("  ←/→: switch  Enter/↓: load  ")
	} else {
		hint = dimStyle.Render("  ↑↓: navigate  Enter: details  Esc: back  ")
	}

	return subTabBar + "\n" + content + "\n" + hint
}

func (p *Panel) renderSubTabBar() string {
	t := p.Theme

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Bright).
		Background(t.Secondary).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(t.Dim).
		Padding(0, 1)

	focusedStyle := activeStyle
	if p.Focus != FocusSubTab {
		focusedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(t.Bright).
			Padding(0, 1)
	}

	var tabs []string

	for i, label := range subTabLabels {
		if SubTab(i) == p.ActiveSub {
			tabs = append(tabs, focusedStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveStyle.Render(label))
		}
	}

	return " " + lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
}

// cardLines returns the number of display lines per card for the active sub-tab.
func (p *Panel) cardLines() int {
	switch p.ActiveSub {
	case SubTabWorkflows:
		return 3 //nolint:mnd // icon+name, branch+event+age, status
	default:
		return 2 //nolint:mnd // title line + detail line
	}
}

func (p *Panel) visibleCards() int {
	return max(p.Height/p.cardLines(), 1)
}

func (p *Panel) renderList() string {
	t := p.Theme
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)

	itemCount := p.activeItemCount()
	if itemCount == 0 {
		return dimStyle.Render("  (no data)")
	}

	visible := p.visibleCards()
	p.ClampScroll(visible)

	start, end := p.VisibleRange(itemCount, visible)
	rows := make([]string, 0, end-start)

	for i := start; i < end; i++ {
		rows = append(rows, p.renderRow(i))
	}

	// Show "more" indicator if we likely have more pages.
	if p.hasMore[p.ActiveSub] && end >= itemCount {
		rows = append(rows, dimStyle.Render("  ··· more ···"))
	}

	usedLines := len(rows) * p.cardLines()
	rows = panels.PadRows(rows, usedLines+(p.Height-usedLines))

	return strings.Join(rows, "\n")
}

const defaultPerPage = 20

func (p *Panel) renderRow(idx int) string {
	switch p.ActiveSub {
	case SubTabIssues:
		return p.renderIssueRow(idx)
	case SubTabPRs:
		return p.renderPRRow(idx)
	case SubTabWorkflows:
		return p.renderWorkflowRow(idx)
	default:
		return ""
	}
}

func (p *Panel) renderIssueRow(idx int) string {
	if idx >= len(p.issues) {
		return ""
	}

	issue := p.issues[idx]
	t := p.Theme
	selected := idx == p.Cursor
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)
	titleStyle := lipgloss.NewStyle().Foreground(t.Text)

	if selected {
		titleStyle = titleStyle.Foreground(t.Bright).Bold(true)
	}

	// Line 1: icon #number title
	title := elideToWidth(issue.Title, p.Width-10) //nolint:mnd // icon+#num+gaps
	line1 := fmt.Sprintf(" %s %s %s",
		issueStateIcon(issue.State),
		dimStyle.Render(fmt.Sprintf("#%d", issue.Number)),
		titleStyle.Render(title),
	)

	// Line 2: author · labels · age
	labels := ""
	if len(issue.Labels) > 0 {
		labels = "·" + dimStyle.Render(elideToWidth(strings.Join(issue.Labels, ","), 20)) //nolint:mnd // label budget
	}

	line2 := fmt.Sprintf("   %s %s ·%s",
		dimStyle.Render(elideToWidth(issue.Author, 15)), //nolint:mnd // author budget
		labels,
		dimStyle.Render(gadgets.TimeAgo(issue.UpdatedAt)),
	)

	card := line1 + "\n" + line2

	if selected {
		card = lipgloss.NewStyle().
			Background(t.SelectedBg).
			Width(p.Width).
			Render(card)
	}

	return card
}

func (p *Panel) renderPRRow(idx int) string {
	if idx >= len(p.prs) {
		return ""
	}

	pr := p.prs[idx]
	t := p.Theme
	selected := idx == p.Cursor
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)
	titleStyle := lipgloss.NewStyle().Foreground(t.Text)

	if selected {
		titleStyle = titleStyle.Foreground(t.Bright).Bold(true)
	}

	// Line 1: icon #number title
	title := elideToWidth(pr.Title, p.Width-10) //nolint:mnd // icon+#num+gaps
	line1 := fmt.Sprintf(" %s %s %s",
		prStateIcon(pr.State, pr.Draft),
		dimStyle.Render(fmt.Sprintf("#%d", pr.Number)),
		titleStyle.Render(title),
	)

	// Line 2: author · branch→base · age
	branchInfo := elideToWidth(pr.Branch+"→"+pr.Base, 25) //nolint:mnd // branch budget
	line2 := fmt.Sprintf("   %s ·%s ·%s",
		dimStyle.Render(elideToWidth(pr.Author, 15)), //nolint:mnd // author budget
		dimStyle.Render(branchInfo),
		dimStyle.Render(gadgets.TimeAgo(pr.UpdatedAt)),
	)

	card := line1 + "\n" + line2

	if selected {
		card = lipgloss.NewStyle().
			Background(t.SelectedBg).
			Width(p.Width).
			Render(card)
	}

	return card
}

func (p *Panel) renderWorkflowRow(idx int) string {
	if idx >= len(p.runs) {
		return ""
	}

	run := p.runs[idx]
	t := p.Theme
	selected := idx == p.Cursor
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)
	titleStyle := lipgloss.NewStyle().Foreground(t.Text)

	if selected {
		titleStyle = titleStyle.Foreground(t.Bright).Bold(true)
	}

	// Line 1: icon name
	name := elideToWidth(run.Name, p.Width-4) //nolint:mnd // icon+gap
	line1 := fmt.Sprintf(" %s %s", workflowIcon(run.Status, run.Conclusion), titleStyle.Render(name))

	// Line 2: branch · event · age
	branch := elideToWidth(run.Branch, 20) //nolint:mnd // branch budget
	line2 := fmt.Sprintf("   %s ·%s ·%s",
		dimStyle.Render(branch),
		dimStyle.Render(run.Event),
		dimStyle.Render(gadgets.TimeAgo(run.CreatedAt)),
	)

	// Line 3: status: conclusion
	statusLine := run.Status
	if run.Conclusion != "" {
		statusLine += ": " + run.Conclusion
	}

	line3 := "   " + dimStyle.Render(statusLine)

	card := line1 + "\n" + line2 + "\n" + line3

	if selected {
		card = lipgloss.NewStyle().
			Background(t.SelectedBg).
			Width(p.Width).
			Render(card)
	}

	return card
}

func elideToWidth(s string, maxW int) string {
	if maxW < 4 { //nolint:mnd // minimum for ellipsis
		maxW = 4 //nolint:mnd // minimum
	}

	if len(s) > maxW {
		return s[:maxW-1] + "\u2026"
	}

	return s
}

func (p *Panel) activeItemCount() int {
	switch p.ActiveSub {
	case SubTabIssues:
		return len(p.issues)
	case SubTabPRs:
		return len(p.prs)
	case SubTabWorkflows:
		return len(p.runs)
	default:
		return 0
	}
}

func (p *Panel) fetchCmd() tea.Cmd {
	var subjectKind models.SubjectKind

	switch p.ActiveSub {
	case SubTabIssues:
		subjectKind = models.SubjectIssues
	case SubTabPRs:
		subjectKind = models.SubjectPullRequests
	case SubTabWorkflows:
		subjectKind = models.SubjectWorkflowRuns
	default:
		return nil
	}

	page := p.Page

	return func() tea.Msg {
		return uxtypes.FetchDetailMsg{
			Scope: models.ActionSuggestion{
				SubjectKind: subjectKind,
				Subjects: []models.ActionSubject{{
					Subject: "list",
					Params:  []string{fmt.Sprintf("page=%d", page), fmt.Sprintf("per_page=%d", defaultPerPage)},
				}},
			},
		}
	}
}

func (p *Panel) detailCmd() tea.Cmd {
	switch p.ActiveSub {
	case SubTabIssues:
		if p.Cursor >= len(p.issues) {
			return nil
		}

		issue := p.issues[p.Cursor]

		// Route through the engine so we pick up the issue body via the
		// dedicated GitHub API call. The popup content is built once the
		// detail arrives.
		return func() tea.Msg {
			return uxtypes.FetchDetailMsg{
				Scope: models.ActionSuggestion{
					SubjectKind: models.SubjectIssueDetail,
					Subjects: []models.ActionSubject{{
						Subject: strconv.Itoa(issue.Number),
					}},
				},
			}
		}

	case SubTabPRs:
		if p.Cursor >= len(p.prs) {
			return nil
		}

		pr := p.prs[p.Cursor]
		title := fmt.Sprintf("PR #%d: %s", pr.Number, pr.Title)

		return func() tea.Msg {
			return uxtypes.ShowDetailMsg{
				Title:   title,
				Content: pr.Title,
				OpenURL: pr.HTMLURL,
			}
		}

	case SubTabWorkflows:
		if p.Cursor >= len(p.runs) {
			return nil
		}

		run := p.runs[p.Cursor]
		title := fmt.Sprintf("Run %d: %s", run.ID, run.Name)

		return func() tea.Msg {
			return uxtypes.ShowDetailMsg{
				Title:   title,
				Content: run.Name,
				OpenURL: run.HTMLURL,
			}
		}

	default:
		return nil
	}
}

func issueStateIcon(state string) string {
	if state == "open" {
		return "🟢"
	}

	return "🟣"
}

func prStateIcon(state string, draft bool) string {
	if draft {
		return "⚪"
	}

	switch state {
	case "open":
		return "🟢"
	case "merged":
		return "🟣"
	default:
		return "🔴"
	}
}

func workflowIcon(status, conclusion string) string {
	if status != "completed" {
		return "🔵" // in progress
	}

	switch conclusion {
	case "success":
		return "✅"
	case "failure":
		return "❌"
	case "cancelled":
		return "⚪"
	default:
		return "🟡"
	}
}

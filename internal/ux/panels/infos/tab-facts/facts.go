package facts

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	"github.com/fredbi/git-janitor/internal/ux/key"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// Panel displays a quick recap of the selected repository's properties.
type Panel struct {
	Theme         *types.Theme
	Info          *models.RepoInfo
	GitHubEnabled bool // whether the GitHub provider is available
	Offset        int  // scroll offset
	Width         int
	Height        int
}

// New creates a new Panel.
func New(theme *types.Theme) Panel {
	return Panel{Theme: theme}
}

func (p *Panel) SetInfo(info *models.RepoInfo) {
	p.Info = info
	p.Info.Platform = nil // clear until GitHub data arrives
	p.Offset = 0
}

func (p *Panel) SetGitHubData(info *models.RepoInfo) {
	if p.Info == nil {
		p.Info = info

		return
	}

	p.Info.Platform = info.Platform
}

func (p *Panel) SetSize(w, h int) {
	p.Width = w
	p.Height = h
}

func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	lines := p.buildLines()
	maxOffset := max(len(lines)-p.Height, 0)

	switch key.MsgBinding(km) {
	case key.Up, key.K:
		if p.Offset > 0 {
			p.Offset--
		}
	case key.Down, key.J:
		if p.Offset < maxOffset {
			p.Offset++
		}
	case key.PageUp:
		p.Offset = max(0, p.Offset-p.Height)
	case key.PageDown:
		p.Offset = min(maxOffset, p.Offset+p.Height)
	case key.Home, key.G:
		p.Offset = 0
	case key.End, key.GG:
		p.Offset = maxOffset
	}

	return nil
}

func (p *Panel) View() string {
	if p.Info.IsEmpty() {
		return lipgloss.NewStyle().Foreground(p.Theme.Dim).
			Render("  Select a repository to view its properties.")
	}

	lines := p.buildLines()

	// Apply scroll.
	end := min(p.Offset+p.Height, len(lines))

	visible := lines[p.Offset:end]

	// Pad to exactly p.Height lines to prevent overflow.
	for len(visible) < p.Height {
		visible = append(visible, "")
	}

	return strings.Join(visible, "\n")
}

func (p *Panel) buildLines() []string {
	if p.Info.IsEmpty() {
		return nil
	}

	info := p.Info
	t := p.Theme
	// TODO: factorize
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Secondary)
	valStyle := lipgloss.NewStyle().Foreground(t.Text)
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)
	warnStyle := lipgloss.NewStyle().Foreground(t.Warning)

	if err := info.RepoErr(); err != nil {
		return []string{
			warnStyle.Render(fmt.Sprintf("  Error: %v", err)),
		}
	}

	var lines []string

	// maxVal is the maximum visible width for a value after the label.
	maxVal := max(p.Width-18, 10) //nolint:mnd // 2 indent + ~14 label + 2 gap

	elide := func(s string, maxW int) string {
		// Take first line only, then truncate.
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[:idx]
		}

		if len(s) > maxW && maxW > 3 { //nolint:mnd // room for ellipsis
			return s[:maxW-1] + "\u2026"
		}

		return s
	}

	line := func(label, value string) {
		lines = append(lines, fmt.Sprintf("  %s  %s", labelStyle.Render(label), valStyle.Render(elide(value, maxVal))))
	}

	// Path.
	line("Path:", info.Path)

	// Kind and SCM.
	line("Kind:", info.Kind.String())
	line("SCM:", info.SCM.String())

	// GitHub indicator (only for GitHub-hosted repos).
	if info.SCM == models.SCMGitHub {
		if p.GitHubEnabled {
			line("GitHub:", "☑ enabled")
		} else {
			line("GitHub:", dimStyle.Render("☐ disabled")+" "+dimStyle.Render("(set GH_TOKEN)"))
		}
	}

	// Remote status (shown when fetch failed).
	if info.FetchErr != nil {
		line("Remote:", warnStyle.Render(elide(fmt.Sprintf("unavailable — %v", info.FetchErr), maxVal)))
	}

	// Non-git directory: show minimal info.
	if !info.IsGit {
		return lines
	}

	// Last commit: date on the label line, message on its own line.
	if !info.LastCommit.IsZero() {
		line("Last commit:", info.LastCommit.Format("2006-01-02 15:04"))

		if info.LastCommitMessage != "" {
			msg := elide(info.LastCommitMessage, p.Width-6) //nolint:mnd // indent
			lines = append(lines, "    "+dimStyle.Render(msg))
		}
	}

	// Total commits and first-commit date (absent on shallow clones).
	if info.CommitCount > 0 {
		line("Commits:", strconv.Itoa(info.CommitCount))
	}

	if !info.FirstCommit.IsZero() {
		line("First commit:", gadgets.TimeAgo(info.FirstCommit)+" "+dimStyle.Render("("+info.FirstCommit.Format("2006-01-02")+")"))
	}

	if info.LastSemverTag != "" {
		suffix := "(" + gadgets.TimeAgo(info.LastSemverDate)
		if info.CommitsSinceLastTag > 0 {
			suffix += fmt.Sprintf(", +%d commit", info.CommitsSinceLastTag)
			if info.CommitsSinceLastTag > 1 {
				suffix += "s"
			}
		}

		suffix += ")"
		line("Latest tag:", info.LastSemverTag+" "+dimStyle.Render(suffix))
	} else {
		line("Latest tag:", dimStyle.Render("no tag"))
	}

	// Last local update (differs from last commit when worktree is dirty).
	if !info.LastLocalUpdate.IsZero() && info.LastLocalUpdate != info.LastCommit {
		line("Last update:", info.LastLocalUpdate.Format("2006-01-02 15:04")+" "+dimStyle.Render("(dirty files)"))
	}

	// Current branch.
	branch := info.Status.Branch
	if branch == "" {
		branch = "(detached HEAD)"
	}

	line("Branch:", elide(branch, maxVal))

	// Default branch.
	if info.DefaultBranch != "" {
		line("Default:", info.DefaultBranch)
	}

	// HEAD.
	oid := info.Status.OID
	if len(oid) > 10 { //nolint:mnd // short hash
		oid = oid[:10]
	}

	line("HEAD:", oid)

	// Upstream.
	if info.Status.Upstream != "" {
		upstream := info.Status.Upstream
		if info.Status.Ahead > 0 || info.Status.Behind > 0 {
			upstream += fmt.Sprintf("  (ahead %d, behind %d)", info.Status.Ahead, info.Status.Behind)
		}

		line("Upstream:", elide(upstream, maxVal))
	} else {
		line("Upstream:", dimStyle.Render("(none)"))
	}

	// Dirty status.
	if info.Status.IsDirty() {
		var counts []string

		staged, unstaged, untracked := classifyEntries(info.Status.Entries)
		if staged > 0 {
			counts = append(counts, fmt.Sprintf("%d staged", staged))
		}

		if unstaged > 0 {
			counts = append(counts, fmt.Sprintf("%d unstaged", unstaged))
		}

		if untracked > 0 {
			counts = append(counts, fmt.Sprintf("%d untracked", untracked))
		}

		line("Working tree:", warnStyle.Render("dirty")+" "+dimStyle.Render("("+strings.Join(counts, ", ")+")"))
	} else {
		line("Working tree:", "clean")
	}

	// Branches summary.
	local, remote := countBranches(info.Branches)
	line("Branches:", fmt.Sprintf("%d local, %d remote", local, remote))

	// Remotes.
	if len(info.Remotes) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+labelStyle.Render("Remotes:"))

		maxURL := max(p.Width-16, 10) //nolint:mnd // indent + remote name

		for _, rm := range info.Remotes {
			lines = append(lines, fmt.Sprintf("    %s  %s",
				valStyle.Render(rm.Name),
				dimStyle.Render(elide(rm.FetchURL, maxURL)),
			))
		}
	}

	// Stashes (count only — detail in the Stashes tab).
	if len(info.Stashes) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Stashes:"),
			valStyle.Render(strconv.Itoa(len(info.Stashes))),
		))
	}

	// GitHub section.
	lines = append(lines, p.buildGitHubLines(labelStyle, valStyle, dimStyle, warnStyle)...)

	return lines
}

func (p *Panel) buildGitHubLines(labelStyle, valStyle, dimStyle, warnStyle lipgloss.Style) []string {
	if p.Info.IsEmpty() {
		return nil
	}

	gh := p.Info.Platform
	if gh == nil {
		return nil
	}

	var lines []string

	sep := dimStyle.Render("  ── GitHub ──────────────")
	lines = append(lines, "", sep)

	line := func(label, value string) {
		lines = append(lines, fmt.Sprintf("  %s  %s", labelStyle.Render(label), valStyle.Render(value)))
	}

	if gh.Err != nil {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Status:"),
			warnStyle.Render(gh.Err.Error()),
		))

		return lines
	}

	// Visibility.
	vis := "public"
	if gh.IsPrivate {
		vis = "private"
	}

	line("Visibility:", vis)

	// Fork lineage.
	if gh.IsFork && gh.ParentFullName != "" {
		line("Fork of:", gh.ParentFullName)
	}

	// Description.
	if gh.Description != "" {
		desc := gh.Description
		maxLen := p.Width - 20
		if maxLen > 0 && len(desc) > maxLen {
			desc = desc[:maxLen-1] + "..."
		}

		line("Description:", desc)
	}

	// Counts on one line.
	counts := fmt.Sprintf("★ %d  Forks: %d  Issues: %d  PRs: %d",
		gh.StarCount, gh.ForkCount, gh.OpenIssues, gh.OpenPRs)
	lines = append(lines, "  "+valStyle.Render(counts))

	// License. GitHub returns "NOASSERTION" when its classifier cannot
	// identify the license (e.g. a custom preamble on an otherwise standard
	// text); render it as "unclassified" instead of the raw SPDX sentinel.
	if gh.License != "" {
		display := gh.License
		if display == "NOASSERTION" {
			display = "unclassified"
		}

		line("License:", display)
	}

	// Archived.
	if gh.IsArchived {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Archived:"),
			warnStyle.Render("yes"),
		))
	}

	// Security alerts — per-scanner status.
	lines = append(lines, p.buildSecurityLines(gh, labelStyle, valStyle, dimStyle, warnStyle)...)

	// Branch protection.
	line("Protected:", checkbox(gh.DefaultBranchProtected)+dimStyle.Render(" ("+gh.DefaultBranch+")"))

	// Topics.
	if len(gh.Topics) > 0 {
		line("Topics:", strings.Join(gh.Topics, ", "))
	}

	// Fork-specific section: show data from whichever side is the fork.
	// If origin is the fork, use origin's Platform.
	// If upstream is the fork, use UpstreamPlatform.
	fork := p.forkPlatform()
	if fork != nil {
		sep2 := dimStyle.Render("  ── GitHub (fork: " + fork.FullName + ") ──")
		lines = append(lines, "", sep2)

		line("Protected:", checkbox(fork.DefaultBranchProtected)+dimStyle.Render(" ("+fork.DefaultBranch+")"))
		line("CI enabled:", checkbox(fork.ActionsEnabled))
		line("Delete head:", checkbox(fork.DeleteBranchOnMerge))
	}

	return lines
}

// forkPlatform returns the PlatformInfo for the fork side of the relationship.
// If origin is the fork, returns Platform. If upstream is the fork, returns UpstreamPlatform.
// Returns nil if no fork relationship exists.
func (p *Panel) forkPlatform() *models.PlatformInfo {
	if p.Info == nil {
		return nil
	}

	// Origin is the fork.
	if p.Info.Platform != nil && p.Info.Platform.IsFork {
		return p.Info.Platform
	}

	// Upstream is the fork.
	if p.Info.UpstreamPlatform != nil && p.Info.UpstreamPlatform.IsFork {
		return p.Info.UpstreamPlatform
	}

	return nil
}

func checkbox(val int) string {
	switch {
	case val > 0:
		return "☑"
	case val == 0:
		return "☐"
	default:
		return "?"
	}
}

// classifyEntries counts staged, unstaged, and untracked entries.
func classifyEntries(entries []models.StatusEntry) (staged, unstaged, untracked int) {
	for _, e := range entries {
		if e.IsUntracked() {
			untracked++

			continue
		}

		if e.XY[0] != '.' {
			staged++
		}

		if e.XY[1] != '.' {
			unstaged++
		}
	}

	return
}

func (p *Panel) buildSecurityLines(gh *models.PlatformInfo, labelStyle, valStyle, dimStyle, warnStyle lipgloss.Style) []string {
	if gh.SecuritySkipped {
		return []string{fmt.Sprintf("  %s  %s",
			labelStyle.Render("Security:"),
			dimStyle.Render("not queried"),
		)}
	}

	dep := gh.DependabotAlerts
	code := gh.CodeScanningAlerts
	secret := gh.SecretScanningAlerts

	// All inaccessible: show a single dim line.
	if dep < 0 && code < 0 && secret < 0 {
		return []string{fmt.Sprintf("  %s  %s",
			labelStyle.Render("Security:"),
			dimStyle.Render("no access to security APIs"),
		)}
	}

	var lines []string

	scannerLine := func(name string, count int) {
		switch {
		case count < 0:
			lines = append(lines, fmt.Sprintf("    %s  %s",
				dimStyle.Render(name+":"),
				dimStyle.Render("no access"),
			))
		case count == 0:
			lines = append(lines, fmt.Sprintf("    %s  %s",
				dimStyle.Render(name+":"),
				valStyle.Render("clean"),
			))
		default:
			lines = append(lines, fmt.Sprintf("    %s  %s",
				dimStyle.Render(name+":"),
				warnStyle.Render(fmt.Sprintf("%d open", count)),
			))
		}
	}

	total := gh.SecurityAlerts()
	if total > 0 {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Security:"),
			warnStyle.Render(fmt.Sprintf("%d open alert(s)", total)),
		))
	} else {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Security:"),
			valStyle.Render("no open alerts"),
		))
	}

	scannerLine("Dependabot", dep)
	scannerLine("Code scanning", code)
	scannerLine("Secret scanning", secret)

	return lines
}

// countBranches counts local and remote branches.
func countBranches(branches []models.Branch) (local, remote int) {
	for _, b := range branches {
		if b.IsRemote {
			remote++
		} else {
			local++
		}
	}

	return
}

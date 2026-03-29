package facts

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/git"
	"github.com/fredbi/git-janitor/internal/github"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// FactsPanel displays a quick recap of the selected repository's properties.
type FactsPanel struct {
	Info       *git.RepoInfo
	GitHubData *github.RepoData
	Offset     int // scroll offset
	Width      int
	Height     int
}

// New creates a new FactsPanel.
func New() FactsPanel {
	return FactsPanel{}
}

func (p *FactsPanel) SetInfo(info *git.RepoInfo) {
	p.Info = info
	p.GitHubData = nil // clear until GitHub data arrives
	p.Offset = 0
}

func (p *FactsPanel) SetGitHubData(data *github.RepoData) {
	p.GitHubData = data
}

func (p *FactsPanel) SetSize(w, h int) {
	p.Width = w
	p.Height = h
}

func (p *FactsPanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	lines := p.buildLines()
	maxOffset := len(lines) - p.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	switch km.String() {
	case "up", "k":
		if p.Offset > 0 {
			p.Offset--
		}
	case "down", "j":
		if p.Offset < maxOffset {
			p.Offset++
		}
	case "home", "g":
		p.Offset = 0
	case "end", "G":
		p.Offset = maxOffset
	}

	return nil
}

func (p *FactsPanel) View() string {
	if p.Info == nil {
		return lipgloss.NewStyle().Foreground(types.CurrentTheme.Dim).
			Render("  Select a repository to view its properties.")
	}

	lines := p.buildLines()

	// Apply scroll.
	end := p.Offset + p.Height
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[p.Offset:end]

	return strings.Join(visible, "\n")
}

func (p *FactsPanel) buildLines() []string {
	if p.Info == nil {
		return nil
	}

	info := p.Info
	t := types.CurrentTheme
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Secondary)
	valStyle := lipgloss.NewStyle().Foreground(t.Text)
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)
	warnStyle := lipgloss.NewStyle().Foreground(t.Warning)

	if info.Err != nil {
		return []string{
			warnStyle.Render(fmt.Sprintf("  Error: %v", info.Err)),
		}
	}

	var lines []string

	line := func(label, value string) {
		lines = append(lines, fmt.Sprintf("  %s  %s", labelStyle.Render(label), valStyle.Render(value)))
	}

	// Path.
	line("Path:", info.Path)

	// Kind and SCM.
	line("Kind:", info.Kind)
	line("SCM:", info.SCM)

	// Remote status (shown when fetch failed).
	if info.FetchErr != nil {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Remote:"),
			warnStyle.Render("unavailable — "+info.FetchErr.Error())))
	}

	// Non-git directory: show minimal info.
	if !info.IsGit {
		return lines
	}

	// Last commit.
	if !info.LastCommit.IsZero() {
		line("Last commit:", info.LastCommit.Format("2006-01-02 15:04"))
	}

	// Current branch.
	branch := info.Status.Branch
	if branch == "" {
		branch = "(detached HEAD)"
	}

	line("Branch:", branch)

	// Default branch.
	if info.DefaultBranch != "" {
		line("Default:", info.DefaultBranch)
	}

	// HEAD.
	oid := info.Status.OID
	if len(oid) > 10 {
		oid = oid[:10]
	}

	line("HEAD:", oid)

	// Upstream.
	if info.Status.Upstream != "" {
		upstream := info.Status.Upstream
		if info.Status.Ahead > 0 || info.Status.Behind > 0 {
			upstream += fmt.Sprintf("  (ahead %d, behind %d)", info.Status.Ahead, info.Status.Behind)
		}

		line("Upstream:", upstream)
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

		line("Working tree:", warnStyle.Render("dirty")+dimStyle.Render("  ("+strings.Join(counts, ", ")+")"))
	} else {
		line("Working tree:", "clean")
	}

	// Branches summary.
	local, remote := countBranches(info.Branches)
	line("Branches:", fmt.Sprintf("%d local, %d remote", local, remote))

	// Remotes.
	if len(info.Remotes) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s", labelStyle.Render("Remotes:")))

		for _, rm := range info.Remotes {
			lines = append(lines, fmt.Sprintf("    %s  %s",
				valStyle.Render(rm.Name),
				dimStyle.Render(rm.FetchURL),
			))
		}
	}

	// Stashes.
	if len(info.Stashes) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Stashes:"),
			valStyle.Render(fmt.Sprintf("%d", len(info.Stashes))),
		))

		for _, st := range info.Stashes {
			msg := st.Message
			if msg == "" {
				msg = "(no message)"
			}

			lines = append(lines, fmt.Sprintf("    %s  %s  %s",
				dimStyle.Render(st.Ref),
				valStyle.Render(st.Branch),
				dimStyle.Render(msg),
			))
		}
	}

	// GitHub section.
	lines = append(lines, p.buildGitHubLines(labelStyle, valStyle, dimStyle, warnStyle)...)

	return lines
}

func (p *FactsPanel) buildGitHubLines(labelStyle, valStyle, dimStyle, warnStyle lipgloss.Style) []string {
	gh := p.GitHubData
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
			desc = desc[:maxLen-1] + "…"
		}

		line("Description:", desc)
	}

	// Counts on one line.
	counts := fmt.Sprintf("★ %d  Forks: %d  Issues: %d  PRs: %d",
		gh.StarCount, gh.ForkCount, gh.OpenIssues, gh.OpenPRs)
	lines = append(lines, fmt.Sprintf("  %s", valStyle.Render(counts)))

	// License.
	if gh.License != "" {
		line("License:", gh.License)
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

	// Topics.
	if len(gh.Topics) > 0 {
		line("Topics:", strings.Join(gh.Topics, ", "))
	}

	return lines
}

// classifyEntries counts staged, unstaged, and untracked entries.
func classifyEntries(entries []git.StatusEntry) (staged, unstaged, untracked int) {
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

func (p *FactsPanel) buildSecurityLines(gh *github.RepoData, labelStyle, valStyle, dimStyle, warnStyle lipgloss.Style) []string {
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
func countBranches(branches []git.Branch) (local, remote int) {
	for _, b := range branches {
		if b.IsRemote {
			remote++
		} else {
			local++
		}
	}

	return
}

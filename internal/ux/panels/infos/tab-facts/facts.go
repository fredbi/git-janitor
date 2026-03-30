package facts

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/engine"
	git "github.com/fredbi/git-janitor/internal/git/backend"
	github "github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// Panel displays a quick recap of the selected repository's properties.
type Panel struct {
	Info   *engine.RepoInfo
	Offset int // scroll offset
	Width  int
	Height int
}

// New creates a new Panel.
func New() Panel {
	return Panel{}
}

func (p *Panel) SetInfo(info *engine.RepoInfo) {
	p.Info = info
	p.Info.GitHubInfo = nil // clear until GitHub data arrives
	p.Offset = 0
}

func (p *Panel) SetGitHubData(info *engine.RepoInfo) {
	if p.Info == nil {
		p.Info = info

		return
	}

	p.Info.GitHubInfo = info.GitHubInfo
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

func (p *Panel) View() string {
	if p.Info.IsEmpty() {
		return lipgloss.NewStyle().Foreground(types.CurrentTheme.Dim).
			Render("  Select a repository to view its properties.")
	}

	lines := p.buildLines()

	// Apply scroll.
	end := min(p.Offset+p.Height, len(lines))

	visible := lines[p.Offset:end]

	return strings.Join(visible, "\n")
}

func (p *Panel) buildLines() []string {
	if p.Info.IsEmpty() {
		return nil
	}

	info := p.Info
	t := types.CurrentTheme
	// TODO: factorize
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Secondary)
	valStyle := lipgloss.NewStyle().Foreground(t.Text)
	dimStyle := lipgloss.NewStyle().Foreground(t.Dim)
	warnStyle := lipgloss.NewStyle().Foreground(t.Warning)

	if err := info.Err(); err != nil {
		return []string{
			warnStyle.Render(fmt.Sprintf("  Error: %v", err)),
		}
	}

	var lines []string

	line := func(label, value string) {
		lines = append(lines, fmt.Sprintf("  %s  %s", labelStyle.Render(label), valStyle.Render(value)))
	}

	// Path.
	line("Path:", info.GitInfo.Path)

	// Kind and SCM.
	line("Kind:", info.GitInfo.Kind.String())
	line("SCM:", info.GitInfo.SCM.String())

	// Remote status (shown when fetch failed).
	if info.GitInfo.FetchErr != nil {
		lines = append(
			lines,
			"  "+
				labelStyle.Render("Remote:")+
				" "+
				warnStyle.Render(fmt.Sprintf("unavailable — %v", info.GitInfo.FetchErr))+
				" ",
		)
	}

	// Non-git directory: show minimal info.
	if !info.GitInfo.IsGit {
		return lines
	}

	// Last commit.
	if !info.GitInfo.LastCommit.IsZero() {
		line("Last commit:", info.GitInfo.LastCommit.Format("2006-01-02 15:04"))
	}

	// Current branch.
	branch := info.GitInfo.Status.Branch
	if branch == "" {
		branch = "(detached HEAD)"
	}

	line("Branch:", branch)

	// Default branch.
	if info.GitInfo.DefaultBranch != "" {
		line("Default:", info.GitInfo.DefaultBranch)
	}

	// HEAD.
	oid := info.GitInfo.Status.OID
	if len(oid) > 10 {
		oid = oid[:10]
	}

	line("HEAD:", oid)

	// Upstream.
	if info.GitInfo.Status.Upstream != "" {
		upstream := info.GitInfo.Status.Upstream
		if info.GitInfo.Status.Ahead > 0 || info.GitInfo.Status.Behind > 0 {
			upstream += fmt.Sprintf("  (ahead %d, behind %d)", info.GitInfo.Status.Ahead, info.GitInfo.Status.Behind)
		}

		line("Upstream:", upstream)
	} else {
		line("Upstream:", dimStyle.Render("(none)"))
	}

	// Dirty status.
	if info.GitInfo.Status.IsDirty() {
		var counts []string

		staged, unstaged, untracked := classifyEntries(info.GitInfo.Status.Entries)
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
	local, remote := countBranches(info.GitInfo.Branches)
	line("Branches:", fmt.Sprintf("%d local, %d remote", local, remote))

	// Remotes.
	if len(info.GitInfo.Remotes) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+labelStyle.Render("Remotes:"))

		for _, rm := range info.GitInfo.Remotes {
			lines = append(lines, fmt.Sprintf("    %s  %s",
				valStyle.Render(rm.Name),
				dimStyle.Render(rm.FetchURL),
			))
		}
	}

	// Stashes.
	if len(info.GitInfo.Stashes) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Stashes:"),
			valStyle.Render(strconv.Itoa(len(info.GitInfo.Stashes))),
		))

		for _, st := range info.GitInfo.Stashes {
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

func (p *Panel) buildGitHubLines(labelStyle, valStyle, dimStyle, warnStyle lipgloss.Style) []string {
	if p.Info.IsEmpty() {
		return nil
	}

	gh := p.Info.GitHubInfo
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
	lines = append(lines, "  "+valStyle.Render(counts))

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

func (p *Panel) buildSecurityLines(gh *github.RepoInfo, labelStyle, valStyle, dimStyle, warnStyle lipgloss.Style) []string {
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

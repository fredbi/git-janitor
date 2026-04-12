package ux

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/models"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// fetchRepoInfo runs git commands in the background and returns a RepoInfoMsg.
func (m *Model) fetchRepoInfo(pth string, isGit bool) tea.Cmd {
	rootIndex := m.SelectedRoot

	return func() tea.Msg {
		if !isGit {
			return noGitCmd(pth)
		}

		// Use the fast path for navigation — skips expensive operations
		// (fsck, file stats, health, merge/rebase checks).
		// Full inspection runs on Ctrl+R (refreshRepo).
		info := m.Engine.Collect(context.Background(), models.NewRepoInfoForRoot(pth, rootIndex), models.CollectFast)

		return uxtypes.RepoInfoMsg{
			Info: info,
		}
	}
}

// refreshRepo runs git fetch --all --tags then re-inspects, returning a RepoRefreshMsg.
func (m *Model) refreshRepo(pth string) tea.Cmd {
	rootIndex := m.SelectedRoot

	return func() tea.Msg {
		info := m.Engine.Refresh(context.Background(), models.NewRepoInfoForRoot(pth, rootIndex))

		return uxtypes.RepoRefreshMsg{
			Info: info,
		}
	}
}

// fullRepoCheck performs a full (non-fast) re-collect of repo info.
// Used after action execution to ensure checks are re-evaluated
// against the updated repository state.
func (m *Model) fullRepoCheck(pth string) tea.Cmd {
	rootIndex := m.SelectedRoot

	return func() tea.Msg {
		info := m.Engine.Collect(context.Background(), models.NewRepoInfoForRoot(pth, rootIndex), models.CollectForceRefresh)

		return uxtypes.RepoInfoMsg{
			Info: info,
		}
	}
}

// triggerGitHubFetch fires an async platform API fetch if applicable.
// The engine checks internally whether the repo is hosted on a supported
// platform, whether a token is available, and whether the config enables it.
func (m *Model) triggerGitHubFetch(info *models.RepoInfo, forceRefresh bool) tea.Cmd {
	if info.IsEmpty() {
		return nil
	}

	opts := []models.CollectOption{models.CollectPlatform}
	if forceRefresh {
		opts = append(opts, models.CollectForceRefresh)
	}

	repoPath := info.Path

	return func() tea.Msg {
		// The engine checks config (GitHub enabled, security alerts) using info.RootIndex.
		result := m.Engine.Collect(context.Background(), info, opts...)

		return uxtypes.GitHubInfoMsg{
			RepoPath: repoPath,
			Data:     result,
		}
	}
}

// fetchDetail runs CollectDetails asynchronously and returns a ShowDetailMsg
// or an ActivityDataMsg for list-type subjects.
func (m *Model) fetchDetail(scope models.ActionSuggestion) tea.Cmd {
	info := m.LastRepoInfo
	if info == nil || info.IsEmpty() {
		return nil
	}

	// Activity list subjects return data to populate the panel, not a popup.
	if isActivityListSubject(scope.SubjectKind) {
		return func() tea.Msg {
			enriched := m.Engine.CollectDetails(context.Background(), info, scope)

			return uxtypes.ActivityDataMsg{Info: enriched}
		}
	}

	return func() tea.Msg {
		enriched := m.Engine.CollectDetails(context.Background(), info, scope)

		title, content := buildDetailContent(enriched, scope)

		return uxtypes.ShowDetailMsg{
			Title:   title,
			Content: content,
			Scope:   scope,
		}
	}
}

func isActivityListSubject(kind models.SubjectKind) bool {
	switch kind {
	case models.SubjectIssues, models.SubjectPullRequests, models.SubjectWorkflowRuns:
		return true
	default:
		return false
	}
}

// buildDetailContent extracts the detail text from the enriched RepoInfo.
func buildDetailContent(info *models.RepoInfo, scope models.ActionSuggestion) (string, string) {
	if len(scope.Subjects) == 0 {
		return "Details", "(no subject)"
	}

	name := scope.Subjects[0].Subject

	switch scope.SubjectKind {
	case models.SubjectBranch:
		return buildBranchDetail(info, name)
	case models.SubjectStash:
		return buildStashDetail(info, name)
	default:
		return "Details: " + name, "(no details available)"
	}
}

func buildBranchDetail(info *models.RepoInfo, name string) (string, string) {
	for _, b := range info.Branches {
		if b.Name != name {
			continue
		}

		if b.Detail == nil {
			return "Branch: " + name, "(details not available)"
		}

		var lines []string
		if b.Detail.LastCommitMessage != "" {
			lines = append(lines, "Last commit: "+b.Detail.LastCommitMessage)
		}

		if b.Detail.DiffStat != "" {
			lines = append(lines, "")
			lines = append(lines, "Diff vs "+info.DefaultBranch+":")
			lines = append(lines, b.Detail.DiffStat)
		}

		if len(lines) == 0 {
			return "Branch: " + name, "(no details)"
		}

		return "Branch: " + name, strings.Join(lines, "\n")
	}

	return "Branch: " + name, "(branch not found)"
}

func buildStashDetail(info *models.RepoInfo, ref string) (string, string) {
	for _, s := range info.Stashes {
		if s.Ref != ref {
			continue
		}

		title := "Stash: " + ref
		if s.Message != "" {
			title += " — " + s.Message
		}

		if s.Detail == nil {
			return title, "(details not available)"
		}

		if s.Detail.DiffStat != "" {
			return title, s.Detail.DiffStat
		}

		return title, "(empty stash)"
	}

	return "Stash: " + ref, "(stash not found)"
}

func noGitCmd(pth string) uxtypes.RepoInfoMsg {
	return uxtypes.RepoInfoMsg{
		Info: models.NoGit(pth),
	}
}

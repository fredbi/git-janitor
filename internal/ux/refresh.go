package ux

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/models"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

func (m *Model) useGitWithPath(pth string) context.Context {
	ctx := context.Background() // TODO: bubbletea context?

	return m.Engine.WithRunner(ctx, "git-runner", pth)
}

func (m *Model) useGithubWithURL(remoteURL string) context.Context {
	ctx := context.Background() // TODO: bubbletea context?

	return m.Engine.WithRunner(ctx, "github-runner", remoteURL)
}

// fetchRepoInfo runs git commands in the background and returns a RepoInfoMsg.
func (m *Model) fetchRepoInfo(pth string, isGit bool) tea.Cmd {
	return func() tea.Msg {
		if !isGit {
			return noGitCmd(pth)
		}

		// Use the fast path for navigation — skips expensive operations
		// (fsck, file stats, health, merge/rebase checks).
		// Full inspection runs on Ctrl+R (refreshRepo).
		data := m.Engine.Collect(m.useGitWithPath(pth), models.CollectFast)
		info, ok := data.(*engine.RepoInfo)
		if !ok {
			panic(fmt.Errorf("internal error: expected repoInfo to be *engine.RepoInfo but got: %T", data))
		}

		// TODO: should separate check evaluation

		return uxtypes.RepoInfoMsg{
			Info: info,
		}
	}
}

// refreshRepo runs git fetch --all --tags then re-inspects, returning a RepoRefreshMsg.
func (m *Model) refreshRepo(pth string) tea.Cmd {
	// TODO: should be a registered git action
	return func() tea.Msg {
		data := m.Engine.Refresh(m.useGitWithPath(pth))
		info, ok := data.(*engine.RepoInfo)
		if !ok {
			panic(fmt.Errorf("internal error: expected repoInfo to be *engine.RepoInfo but got: %T", data))
		}

		return uxtypes.RepoRefreshMsg{
			Info: info,
		}
	}
}

// triggerGitHubFetch fires an async GitHub API fetch if the repo is GitHub-hosted,
// a token is available, and GitHub is enabled for the current root.
func (m *Model) triggerGitHubFetch(info *engine.RepoInfo, forceRefresh bool) tea.Cmd {
	if info.IsEmpty() {
		return nil
	}

	if info.GitInfo.SCM != models.SCMGitHub {
		return nil
	}

	if !m.Engine.RunnerEnabledFor("github-runner", info.GitInfo.Path) {
		return nil
	}
	/*
		// TODO: delegate to engine (knows about config)

		if !m.Cfg.GitHubEnabled(m.SelectedRoot) { // TODO: delegate to engine (knows about config)
			return nil
		}

		if !m.GitHubClient.Available() { // TODO: delegate to engine
			return nil
		}
	*/

	opts := make([]models.CollectOption, 0, 2)
	originURL := engine.OriginFetchURL(info)
	if originURL == "" {
		return nil
	}

	if m.Cfg.GitHubSecurityAlerts(m.SelectedRoot) {
		opts = append(opts, models.CollectSecurityAlerts)
	}
	if forceRefresh {
		opts = append(opts, models.CollectForceRefresh)
	}
	repoPath := info.GitInfo.Path

	return func() tea.Msg {
		ctx := m.useGithubWithURL(originURL)
		data := m.Engine.Collect(ctx, opts...)
		info, ok := data.(*engine.RepoInfo)
		if !ok {
			panic(fmt.Errorf("internal error: expected repoInfo to be *engine.RepoInfo but got: %T", data))
		}

		return uxtypes.GitHubInfoMsg{
			RepoPath: repoPath,
			Data:     info,
		}
	}
}

func noGitCmd(pth string) uxtypes.RepoInfoMsg {
	return uxtypes.RepoInfoMsg{
		Info: engine.NoGit(pth),
	}
}

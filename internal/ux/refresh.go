package ux

import (
	"context"

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

func noGitCmd(pth string) uxtypes.RepoInfoMsg {
	return uxtypes.RepoInfoMsg{
		Info: models.NoGit(pth),
	}
}

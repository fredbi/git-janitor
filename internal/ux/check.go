package ux

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/models"
)

// forceRepoCheck unconditionally fetches info for the selected repo.
// Used when cycling root tabs — always refreshes even if the repo path
// matches (handles single-repo roots revisited after navigating away).
func (m *Model) forceRepoCheck() tea.Cmd {
	m.SelectedRoot = m.Repos.Active

	repo, ok := m.Repos.SelectedRepo()
	if !ok {
		m.SelectedRepo = ""
		m.LastRepoInfo = nil

		return nil
	}

	m.SelectedRepo = repo.Path
	m.Status.SetMessage("Loading " + repo.Name + "...")

	// For non-git repos, update panels immediately (no async fetch needed).
	if !repo.IsGit {
		info := models.NoGit(repo.Path)
		m.LastRepoInfo = info
		m.Right.SetRepoInfo(info)

		return nil
	}

	return m.fetchRepoInfo(repo.Path, true)
}

// checkSelectedRepo detects when the selected repo or root changes and triggers a fetch.
func (m *Model) checkSelectedRepo() tea.Cmd {
	currentRoot := m.Repos.Active

	repo, ok := m.Repos.SelectedRepo()
	if !ok {
		// Root changed but no repo selected (empty tab) — clear panels.
		if currentRoot != m.SelectedRoot {
			m.SelectedRoot = currentRoot
			m.SelectedRepo = ""
			m.LastRepoInfo = nil
		}

		return nil
	}

	repoChanged := repo.Path != m.SelectedRepo
	rootChanged := currentRoot != m.SelectedRoot

	if !repoChanged && !rootChanged {
		return nil
	}

	m.SelectedRepo = repo.Path
	m.SelectedRoot = currentRoot

	return m.fetchRepoInfo(repo.Path, repo.IsGit)
}

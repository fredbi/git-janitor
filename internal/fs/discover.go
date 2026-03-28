package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// DiscoverRepos walks dir looking for first-level subdirectories.
//
// Each subdirectory is listed as a RepoItem. Directories containing a .git
// subdirectory have IsGit set to true; others are listed with IsGit=false.
// Hidden directories and common noise directories are skipped.
func DiscoverRepos(dir string) ([]uxtypes.RepoItem, error) {
	dir, err := ExpandHome(dir)
	if err != nil {
		return nil, err
	}

	// If the root is a linked worktree, skip it entirely.
	if IsLinkedWorktree(dir) {
		return nil, nil
	}

	// If the root directory itself is a git repo, return it as a single entry
	// rather than diving into its subdirectories.
	if IsGitDir(dir) {
		return []uxtypes.RepoItem{{
			Name:  filepath.Base(dir),
			Path:  dir,
			IsGit: true,
		}}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var repos []uxtypes.RepoItem

	for _, d := range entries {
		if !d.IsDir() {
			continue
		}

		name := d.Name()

		// Skip hidden directories and common noise.
		if strings.HasPrefix(name, ".") {
			continue
		}

		if ShouldSkipDir(name) {
			continue
		}

		path := filepath.Join(dir, name)

		// Skip linked worktrees — they appear under their parent repo's
		// worktree list and should not be listed as independent repos.
		if IsLinkedWorktree(path) {
			continue
		}

		repos = append(repos, uxtypes.RepoItem{
			Name:  name,
			Path:  path,
			IsGit: IsGitDir(path),
		})
	}

	return repos, nil
}

// IsGitDir reports whether dir contains a .git subdirectory (a real repository).
// Linked worktrees have a .git file instead of a directory — this returns false for those.
func IsGitDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))

	return err == nil && info.IsDir()
}

// IsLinkedWorktree reports whether dir is a linked git worktree.
// A linked worktree has a .git file (not a directory) whose first line is
// "gitdir: /path/to/parent/.git/worktrees/<name>".
func IsLinkedWorktree(dir string) bool {
	gitPath := filepath.Join(dir, ".git")

	info, err := os.Stat(gitPath)
	if err != nil || info.IsDir() {
		return false
	}

	// It's a file — read the first line to confirm it's a gitdir pointer.
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return false
	}

	return strings.HasPrefix(string(data), "gitdir: ")
}

// ShouldSkipDir returns true for directory names that should never be traversed.
func ShouldSkipDir(name string) bool {
	switch name {
	case "vendor", "node_modules", "__pycache__", ".cache", "dist", "build":
		return true
	default:
		return false
	}
}

// ExpandHome expands a leading ~/ in a path to the user's home directory.
func ExpandHome(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding home directory: %w", err)
		}

		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

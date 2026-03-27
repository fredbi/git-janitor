package state

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/config"
)

// scanResultMsg is sent when a background scan completes.
type scanResultMsg struct {
	// reposByRoot maps root index → discovered repos for that root.
	reposByRoot map[int][]repoItem
	err         error
}

// scanRoots walks all configured roots and returns discovered git repositories,
// grouped by root index.
//
// This runs as a tea.Cmd so it doesn't block the UI.
func scanRoots(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if cfg == nil || len(cfg.Roots) == 0 {
			return scanResultMsg{err: fmt.Errorf("no roots configured — use /config to add one")}
		}

		byRoot := make(map[int][]repoItem, len(cfg.Roots))
		total := 0

		for i, root := range cfg.Roots {
			discovered, err := discoverRepos(root.Path)
			if err != nil {
				return scanResultMsg{err: fmt.Errorf("scanning %s: %w", root.Path, err)}
			}

			byRoot[i] = discovered
			total += len(discovered)
		}

		if total == 0 {
			return scanResultMsg{err: fmt.Errorf("no git repositories found under configured roots")}
		}

		return scanResultMsg{reposByRoot: byRoot}
	}
}

// discoverRepos walks dir looking for directories that contain a .git subdirectory.
//
// It does not recurse into .git directories themselves, and skips hidden
// directories (other than .git detection) and common non-project directories.
func discoverRepos(dir string) ([]repoItem, error) {
	dir, err := expandHome(dir)
	if err != nil {
		return nil, err
	}

	var repos []repoItem

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't read (permission denied, etc.).
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}

			return nil //nolint:nilerr // skip unreadable files silently
		}

		// Only look at directories.
		if !d.IsDir() {
			return nil
		}

		name := d.Name()

		// Skip hidden directories (except the root itself) and common noise.
		if name != "." && strings.HasPrefix(name, ".") {
			return fs.SkipDir
		}

		if shouldSkipDir(name) {
			return fs.SkipDir
		}

		// Check if this directory is a git repo.
		gitDir := filepath.Join(path, ".git")
		if info, statErr := os.Stat(gitDir); statErr == nil && info.IsDir() {
			repos = append(repos, repoItem{
				name: filepath.Base(path),
				path: path,
			})

			// Don't recurse into the repo's subdirectories.
			return fs.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return repos, nil
}

// shouldSkipDir returns true for directory names that should never be traversed.
func shouldSkipDir(name string) bool {
	switch name {
	case "vendor", "node_modules", "__pycache__", ".cache", "dist", "build":
		return true
	default:
		return false
	}
}

func expandHome(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding home directory: %w", err)
		}

		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// maxRecursionDepth is the absolute hard cap on directory recursion,
// independent of any user-supplied MaxDepth value. It guards against
// runaway scans of pathological directory layouts.
const maxRecursionDepth = 16

// DiscoverRepos walks dir looking for first-level subdirectories.
//
// It is a thin compatibility wrapper around [DiscoverReposDepth] with
// maxDepth=1 — preserving the legacy GitHub-style flat behavior. New
// callers should use [DiscoverReposDepth] directly.
func DiscoverRepos(dir string) ([]models.RepoItem, error) {
	return DiscoverReposDepth(dir, 1)
}

// DiscoverReposDepth walks dir looking for git repositories up to
// maxDepth levels deep.
//
// At depth=1 the behavior matches the legacy [DiscoverRepos]: every
// first-level subdirectory is listed (git or not), so the user can see
// non-git noise sitting next to their repos. For depth>1 the walker
// only emits leaves that are actual git repositories — non-git
// intermediate directories are traversed but not surfaced.
//
// A repository's Namespace is the slash-separated relative parent
// directory from dir ("" for top-level repos, "group/sub" when nested).
//
// maxDepth semantics:
//   - 1   → flat layout, current GitHub-style behavior
//   - >1  → recurse that many levels
//   - ≤0  → unlimited (still hard-capped at [maxRecursionDepth])
//
// Hidden directories, common noise directories ([ShouldSkipDir]),
// symlinked entries and linked worktrees are all skipped.
func DiscoverReposDepth(dir string, maxDepth int) ([]models.RepoItem, error) {
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
		return []models.RepoItem{{
			Name:  filepath.Base(dir),
			Path:  dir,
			IsGit: true,
		}}, nil
	}

	effectiveDepth := maxDepth
	if effectiveDepth <= 0 || effectiveDepth > maxRecursionDepth {
		effectiveDepth = maxRecursionDepth
	}

	// Flat mode preserves the legacy behavior of surfacing non-git
	// top-level subdirectories so the user notices stray directories
	// sitting next to their repos. In nested modes the namespace-aware
	// child entries already give that context, so we suppress them.
	emitNonGitTop := maxDepth == 1

	var repos []models.RepoItem
	if err := walkRepos(dir, dir, 1, effectiveDepth, emitNonGitTop, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// walkRepos descends current looking for git repositories. depth is the
// 1-based level of the current directory under root. The recursion stops
// once it has produced repos from level == maxDepth. When emitNonGitTop
// is true, top-level (depth == 1) non-git subdirectories are also
// surfaced as non-git RepoItems — used by flat mode for parity with the
// legacy [DiscoverRepos] behavior.
func walkRepos(root, current string, depth, maxDepth int, emitNonGitTop bool, out *[]models.RepoItem) error {
	entries, err := os.ReadDir(current)
	if err != nil {
		return err
	}

	for _, d := range entries {
		// Skip non-directories and any kind of symlink (loops, escapes).
		t := d.Type()
		if !t.IsDir() || t&os.ModeSymlink != 0 {
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

		path := filepath.Join(current, name)

		// Skip linked worktrees — they appear under their parent repo's
		// worktree list and should not be listed as independent repos.
		if IsLinkedWorktree(path) {
			continue
		}

		if IsGitDir(path) {
			// Real git repository: emit and stop descending into it.
			*out = append(*out, models.RepoItem{
				Name:      name,
				Path:      path,
				Namespace: namespaceFor(root, path),
				IsGit:     true,
			})

			continue
		}

		if emitNonGitTop && depth == 1 {
			// Flat-mode parity: surface top-level non-git directories.
			*out = append(*out, models.RepoItem{
				Name:  name,
				Path:  path,
				IsGit: false,
			})
		}

		// Recurse if there is room.
		if depth < maxDepth {
			if err := walkRepos(root, path, depth+1, maxDepth, emitNonGitTop, out); err != nil {
				return err
			}
		}
	}

	return nil
}

// namespaceFor returns the slash-separated relative parent path of repo
// under root, or "" when repo sits directly under root. The result always
// uses forward slashes regardless of OS so display + filtering stay
// consistent across platforms.
func namespaceFor(root, repo string) string {
	rel, err := filepath.Rel(root, filepath.Dir(repo))
	if err != nil || rel == "." {
		return ""
	}

	return filepath.ToSlash(rel)
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
//
// TODO: configurable skip list.
func ShouldSkipDir(name string) bool {
	switch name {
	case "vendor", "node_modules", "__pycache__", ".cache", "dist", "build":
		return true
	default:
		return false
	}
}

// ExpandHome expands a leading ~/ in a path to the user's home directory.
//
// TODO: move to fs.
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

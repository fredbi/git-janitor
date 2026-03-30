package backend

import (
	"bufio"
	"context"
	"strings"
)

// Worktree represents a git worktree linked to a repository.
type Worktree struct {
	// Path is the absolute filesystem path of the worktree.
	Path string

	// HEAD is the commit hash at the tip of the worktree.
	HEAD string

	// Branch is the checked-out branch (e.g. "refs/heads/main").
	// Empty if the worktree is in detached HEAD state.
	Branch string

	// Detached is true if the worktree is in detached HEAD state.
	Detached bool

	// Bare is true if this is the bare repository entry.
	Bare bool

	// Prunable is true if the worktree path is missing and can be pruned.
	Prunable bool
}

// BranchShort returns the short branch name (e.g. "main" from "refs/heads/main").
// Returns empty string if detached or bare.
func (w Worktree) BranchShort() string {
	return strings.TrimPrefix(w.Branch, "refs/heads/")
}

// IsMain reports whether this is the main worktree (the original checkout,
// not a linked worktree). The main worktree is always listed first by git.
// We detect it by checking that it's not bare and its path matches the
// repository's own directory.
//
// Note: callers typically just check the first entry from Worktrees().
func (w Worktree) IsMain() bool {
	return !w.Bare && w.Branch != "" && !w.Prunable
}

// Worktrees lists all worktrees for the repository.
// The first entry is always the main worktree.
func (r *Runner) Worktrees(ctx context.Context) ([]Worktree, error) {
	out, err := r.run(ctx, cmdWorktreeList()...)
	if err != nil {
		return nil, err
	}

	return parseWorktrees(out), nil
}

// parseWorktrees parses the porcelain output of git worktree list.
//
// Format: blocks separated by blank lines. Each block has lines:
//
//	worktree /path/to/worktree
//	HEAD <hash>
//	branch refs/heads/<name>   (or "detached" or "bare")
//	prunable gitdir file points to non-existent location
func parseWorktrees(output string) []Worktree {
	var worktrees []Worktree
	var current *Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}

			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}

		case strings.HasPrefix(line, "HEAD "):
			if current != nil {
				current.HEAD = strings.TrimPrefix(line, "HEAD ")
			}

		case strings.HasPrefix(line, "branch "):
			if current != nil {
				current.Branch = strings.TrimPrefix(line, "branch ")
			}

		case line == "detached":
			if current != nil {
				current.Detached = true
			}

		case line == "bare":
			if current != nil {
				current.Bare = true
			}

		case strings.HasPrefix(line, "prunable "):
			if current != nil {
				current.Prunable = true
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees
}

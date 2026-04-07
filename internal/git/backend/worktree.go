package backend

import (
	"bufio"
	"context"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// Worktrees lists all worktrees for the repository.
// The first entry is always the main worktree.
func (r *Runner) Worktrees(ctx context.Context) ([]models.Worktree, error) {
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
func parseWorktrees(output string) []models.Worktree {
	var worktrees []models.Worktree
	var current *models.Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}

			current = &models.Worktree{
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

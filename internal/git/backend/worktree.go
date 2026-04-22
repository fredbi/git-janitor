package backend

import (
	"bufio"
	"context"
	"os"
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
// Format: blocks separated by blank lines. Each block may contain:
//
//	worktree /path/to/worktree
//	HEAD <hash>
//	branch refs/heads/<name>      — or "detached" (stand-alone keyword)
//	bare                          — bare repo entry (no HEAD/branch)
//	locked [<reason>]             — worktree is locked; reason optional
//	prunable [<reason>]           — worktree can be pruned; reason optional
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

		case current == nil:
			// Stray line before a worktree header — ignore.

		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")

		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch ")

		case line == "detached":
			current.Detached = true

		case line == "bare":
			current.Bare = true

		case line == "locked" || strings.HasPrefix(line, "locked "):
			current.Locked = true
			current.LockReason = strings.TrimPrefix(strings.TrimPrefix(line, "locked"), " ")

		case line == "prunable" || strings.HasPrefix(line, "prunable "):
			current.Prunable = true
			current.PrunableReason = strings.TrimPrefix(strings.TrimPrefix(line, "prunable"), " ")
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees
}

// MarkWorktreeDetails enriches each non-bare, non-prunable worktree with
// Dirty, LastCommit, and LastCommitMessage. Each worktree is inspected in
// its own working directory via a scratch Runner — the main-worktree
// runner already has this data via RepoInfo.Status, but populating the
// fields uniformly simplifies the UX. Failures are swallowed: the fields
// stay at their zero values.
func (r *Runner) MarkWorktreeDetails(ctx context.Context, wts []models.Worktree) {
	for i := range wts {
		w := &wts[i]

		if w.Bare || w.Prunable {
			continue
		}

		// Skip paths that don't exist on disk — the worktree is effectively
		// prunable even if git hasn't flagged it yet.
		if _, err := os.Stat(w.Path); err != nil {
			continue
		}

		scratch := NewRunner(w.Path)

		if status, err := scratch.Status(ctx); err == nil {
			w.Dirty = status.IsDirty()
		}

		if t, err := scratch.LastCommitTime(ctx); err == nil {
			w.LastCommit = t
		}

		w.LastCommitMessage = scratch.LastCommitMessage(ctx)
	}
}

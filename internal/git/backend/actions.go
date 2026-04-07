package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// UpdateBranch fast-forwards a local branch from its upstream remote.
//
// If the branch is currently checked out, it uses git pull --ff-only.
// Otherwise, it uses git fetch <remote> <branch>:<branch> which updates
// the local ref without changing the working tree or current branch.
//
// The update is strictly fast-forward: if the branch has diverged,
// the operation fails safely.
func (r *Runner) UpdateBranch(ctx context.Context, branch models.Branch) models.ActionResult {
	if !branch.HasUpstream() {
		return models.ActionResult{Message: fmt.Sprintf("branch %s has no upstream configured", branch.Name)}
	}

	// Parse remote name from upstream (e.g. "origin/main" → "origin").
	remote, _, ok := strings.Cut(branch.Upstream, "/")
	if !ok {
		return models.ActionResult{Message: fmt.Sprintf("cannot parse remote from upstream %q", branch.Upstream)}
	}

	if branch.IsCurrent {
		// Guard: dirty worktree.
		if result := r.guardClean(ctx); result != nil {
			return *result
		}

		return r.pullFFOnly(ctx)
	}

	return r.fetchUpdateRef(ctx, remote, branch.Name)
}

// pullFFOnly runs git pull --ff-only on the current branch.
func (r *Runner) pullFFOnly(ctx context.Context) models.ActionResult {
	_, err := r.run(ctx, cmdPullFFOnly()...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("pull --ff-only failed: %v", err)}
	}

	return models.ActionResult{OK: true, Message: "fast-forward updated current branch"}
}

// fetchUpdateRef updates a non-checked-out local branch from a remote
// using git fetch <remote> <branch>:<branch>. This only succeeds for
// fast-forward updates.
func (r *Runner) fetchUpdateRef(ctx context.Context, remote, branch string) models.ActionResult {
	refspec := branch + ":" + branch
	_, err := r.run(ctx, cmdFetchRefspec(remote, refspec)...)
	if err != nil {
		return models.ActionResult{
			Message: fmt.Sprintf("fetch %s %s failed (not fast-forwardable?): %v", remote, refspec, err),
		}
	}

	return models.ActionResult{OK: true, Message: fmt.Sprintf("fast-forward updated %s from %s", branch, remote)}
}

// RebaseBranch rebases a branch onto target (typically the default branch).
//
// If the branch is currently checked out, it runs git rebase <target> directly
// after checking for a clean worktree.
//
// If the branch is NOT checked out, it uses a temporary git worktree to perform
// the rebase without disturbing the user's main checkout. The temp worktree is
// always cleaned up, even on failure.
//
// Prerequisites:
//   - For the current branch: the worktree must be clean.
//   - Use [Runner.CheckRebase] first to verify the rebase would succeed.
func (r *Runner) RebaseBranch(ctx context.Context, target string, branch models.Branch) models.ActionResult {
	if branch.IsCurrent {
		if result := r.guardClean(ctx); result != nil {
			return *result
		}

		return r.rebaseCurrent(ctx, target)
	}

	return r.rebaseInWorktree(ctx, target, branch.Name)
}

// rebaseCurrent rebases the current branch onto target.
func (r *Runner) rebaseCurrent(ctx context.Context, target string) models.ActionResult {
	_, err := r.run(ctx, cmdRebase(target)...)
	if err != nil {
		// Abort the rebase to leave the repo in a clean state.
		r.run(ctx, cmdRebaseAbort()...) //nolint:errcheck // best-effort cleanup

		return models.ActionResult{Message: fmt.Sprintf("rebase onto %s failed: %v", target, err)}
	}

	return models.ActionResult{OK: true, Message: "rebased onto " + target}
}

// rebaseInWorktree rebases a non-checked-out branch using a temporary worktree.
// The main worktree and current branch are never touched.
func (r *Runner) rebaseInWorktree(ctx context.Context, target, branch string) models.ActionResult {
	return r.inWorktree(ctx, branch, func(wt *Runner) models.ActionResult {
		_, err := wt.run(ctx, cmdRebase(target)...)
		if err != nil {
			wt.run(ctx, cmdRebaseAbort()...) //nolint:errcheck // best-effort

			return models.ActionResult{Message: fmt.Sprintf("rebase %s onto %s failed: %v", branch, target, err)}
		}

		return models.ActionResult{OK: true, Message: fmt.Sprintf("rebased %s onto %s", branch, target)}
	})
}

// MergeInto merges a source branch into the current branch.
//
// This is typically used to merge the default branch into the current
// working branch to bring it up to date.
//
// Prerequisites:
//   - The worktree must be clean (no uncommitted changes).
//   - Use [Runner.CanMerge] first to verify the merge would succeed.
func (r *Runner) MergeInto(ctx context.Context, source string) models.ActionResult {
	if result := r.guardClean(ctx); result != nil {
		return *result
	}

	_, err := r.run(ctx, cmdMerge(source)...)
	if err != nil {
		// Abort the merge to leave the repo clean.
		r.run(ctx, cmdMergeAbort()...) //nolint:errcheck // best-effort cleanup

		return models.ActionResult{Message: fmt.Sprintf("merge %s failed: %v", source, err)}
	}

	return models.ActionResult{OK: true, Message: fmt.Sprintf("merged %s into current branch", source)}
}

// RenameRemote renames a git remote.
func (r *Runner) RenameRemote(ctx context.Context, oldName, newName string) models.ActionResult {
	_, err := r.run(ctx, cmdRenameRemote(oldName, newName)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("rename remote %s→%s failed: %v", oldName, newName, err)}
	}

	return models.ActionResult{OK: true, Message: fmt.Sprintf("renamed remote %s to %s", oldName, newName)}
}

// DeleteBranch deletes a local branch using git branch -D (force delete).
//
// Force delete is used because squash-merged branches are not recognized
// by the safe -d flag. The caller should verify the branch is merged
// before calling this.
//
// Refuses to delete the current branch.
func (r *Runner) DeleteBranch(ctx context.Context, name string) models.ActionResult {
	// Guard: refuse to delete the current branch.
	current, err := r.run(ctx, cmdRevParseAbbrev("HEAD")...)
	if err == nil && strings.TrimSpace(current) == name {
		return models.ActionResult{Message: "cannot delete current branch " + name}
	}

	_, err = r.run(ctx, cmdDeleteBranch(name)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("delete branch %s failed: %v", name, err)}
	}

	return models.ActionResult{OK: true, Message: "deleted branch " + name}
}

// PushBranch pushes a local branch to the given remote and sets upstream tracking.
func (r *Runner) PushBranch(ctx context.Context, remote, name string) models.ActionResult {
	_, err := r.run(ctx, cmdPushBranchUpstream(remote, name)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("push branch %s to %s failed: %v", name, remote, err)}
	}

	return models.ActionResult{OK: true, Message: fmt.Sprintf("pushed %s to %s with upstream tracking", name, remote)}
}

// PushTag pushes a single tag to the origin remote.
func (r *Runner) PushTag(ctx context.Context, name string) models.ActionResult {
	_, err := r.run(ctx, cmdPushTag("origin", name)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("push tag %s failed: %v", name, err)}
	}

	return models.ActionResult{OK: true, Message: fmt.Sprintf("pushed tag %s to origin", name)}
}

// Compact runs git gc to reclaim space and optimize the repository.
//
// This is the standard garbage collection pass: repacks objects, prunes
// unreachable objects, expires old reflog entries, and updates the commit-graph.
//
// The timeout is extended to 5 minutes since gc can be slow on large repos.
//
// Use [Runner.Health] and [Runner.Size] to check whether gc is advisable
// before calling this.
func (r *Runner) Compact(ctx context.Context) models.ActionResult {
	return r.runGC(ctx, cmdGC()...)
}

// CompactAggressive runs git gc --aggressive for deeper optimization.
//
// This uses a more thorough (and slower) repack strategy with higher
// compression settings (--depth=50 --window=250). Use this when
// [RepoSize.RepackAdvised] is true or the repository has significant bloat.
//
// The timeout is extended to 10 minutes.
func (r *Runner) CompactAggressive(ctx context.Context) models.ActionResult {
	return r.runGC(ctx, cmdGCAggressive()...)
}

// runGC executes a git gc variant with an extended timeout.
func (r *Runner) runGC(ctx context.Context, args ...string) models.ActionResult {
	// Save and restore the timeout — gc can be slow.
	origTimeout := r.Timeout
	if len(args) > 1 && args[1] == "--aggressive" {
		r.Timeout = 10 * time.Minute
	} else {
		r.Timeout = 5 * time.Minute
	}

	defer func() { r.Timeout = origTimeout }()

	_, err := r.run(ctx, args...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("%s failed: %v", strings.Join(args, " "), err)}
	}

	return models.ActionResult{OK: true, Message: strings.Join(args, " ") + " completed"}
}

// RebaseBranchRemote rebases a remote branch onto target and pushes the result.
//
// The operation is performed entirely in a temporary worktree — the user's
// main checkout is never touched. On success, the rebased branch is pushed
// with --force-with-lease to update the remote safely (fails if someone else
// pushed in the meantime).
//
// Prerequisites:
//   - The branch must track an upstream remote.
//   - Use [Runner.CheckRebase] first to verify the rebase would succeed.
func (r *Runner) RebaseBranchRemote(ctx context.Context, target string, branch models.Branch) models.ActionResult {
	if !branch.HasUpstream() {
		return models.ActionResult{Message: fmt.Sprintf("branch %s has no upstream — cannot push", branch.Name)}
	}

	remote, _, ok := strings.Cut(branch.Upstream, "/")
	if !ok {
		return models.ActionResult{Message: fmt.Sprintf("cannot parse remote from upstream %q", branch.Upstream)}
	}

	return r.inWorktree(ctx, branch.Name, func(wt *Runner) models.ActionResult {
		_, err := wt.run(ctx, cmdRebase(target)...)
		if err != nil {
			wt.run(ctx, cmdRebaseAbort()...) //nolint:errcheck // best-effort

			return models.ActionResult{Message: fmt.Sprintf("rebase %s onto %s failed: %v", branch.Name, target, err)}
		}

		// Push with --force-with-lease: safe force push that fails if the
		// remote branch was updated by someone else since our last fetch.
		_, err = wt.run(ctx, cmdPushForceWithLease(remote, branch.Name)...)
		if err != nil {
			return models.ActionResult{
				Message: fmt.Sprintf("rebased %s onto %s locally, but push failed: %v", branch.Name, target, err),
			}
		}

		return models.ActionResult{OK: true, Message: fmt.Sprintf("rebased %s onto %s and pushed to %s", branch.Name, target, remote)}
	})
}

// MergeIntoRemote merges a source branch into a target branch and pushes the result.
//
// This is typically used to merge the default branch into a feature branch
// to bring it up to date, then push the updated feature branch.
//
// The operation is performed in a temporary worktree — the user's main
// checkout is never touched. On success, the target branch is pushed
// with --force-with-lease.
//
// Prerequisites:
//   - The target branch must track an upstream remote.
//   - Use [Runner.CanMerge] first to verify the merge would succeed.
func (r *Runner) MergeIntoRemote(ctx context.Context, source string, target models.Branch) models.ActionResult {
	if !target.HasUpstream() {
		return models.ActionResult{Message: fmt.Sprintf("branch %s has no upstream — cannot push", target.Name)}
	}

	remote, _, ok := strings.Cut(target.Upstream, "/")
	if !ok {
		return models.ActionResult{Message: fmt.Sprintf("cannot parse remote from upstream %q", target.Upstream)}
	}

	return r.inWorktree(ctx, target.Name, func(wt *Runner) models.ActionResult {
		_, err := wt.run(ctx, cmdMerge(source)...)
		if err != nil {
			wt.run(ctx, cmdMergeAbort()...) //nolint:errcheck // best-effort

			return models.ActionResult{Message: fmt.Sprintf("merge %s into %s failed: %v", source, target.Name, err)}
		}

		_, err = wt.run(ctx, cmdPushForceWithLease(remote, target.Name)...)
		if err != nil {
			return models.ActionResult{
				Message: fmt.Sprintf("merged %s into %s locally, but push failed: %v", source, target.Name, err),
			}
		}

		return models.ActionResult{OK: true, Message: fmt.Sprintf("merged %s into %s and pushed to %s", source, target.Name, remote)}
	})
}

// inWorktree creates a temporary worktree for the given branch, runs the
// provided function with a Runner pointing to the worktree, and cleans up.
// The main worktree is never touched.
func (r *Runner) inWorktree(ctx context.Context, branch string, fn func(wt *Runner) models.ActionResult) models.ActionResult {
	tmpDir, err := os.MkdirTemp("", "janitor-wt-*")
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("cannot create temp directory: %v", err)}
	}

	wtPath := filepath.Join(tmpDir, branch)

	defer func() {
		r.run(ctx, cmdWorktreeRemove(wtPath)...) //nolint:errcheck // best-effort
		os.RemoveAll(tmpDir)
	}()

	_, err = r.run(ctx, cmdWorktreeAdd(wtPath, branch)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("worktree add %s failed: %v", branch, err)}
	}

	wt := NewRunner(wtPath)
	wt.Timeout = r.Timeout

	return fn(wt)
}

// guardClean checks that the worktree is clean. Returns nil if clean,
// or a failure models.ActionResult if dirty or status cannot be determined.
func (r *Runner) guardClean(ctx context.Context) *models.ActionResult {
	status, err := r.Status(ctx)
	if err != nil {
		return &models.ActionResult{Message: fmt.Sprintf("cannot check status: %v", err)}
	}

	if status.IsDirty() {
		return &models.ActionResult{Message: "worktree has uncommitted changes — commit or stash first"}
	}

	return nil
}

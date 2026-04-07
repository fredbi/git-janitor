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

// StashDirty unstages any staged changes, then stashes all uncommitted work
// (including untracked files) with an optional message.
func (r *Runner) StashDirty(ctx context.Context, message string) models.ActionResult {
	// Unstage any staged changes first so the stash captures everything.
	_, _ = r.run(ctx, cmdResetHead()...) // non-fatal: reset may fail if no HEAD

	_, err := r.run(ctx, cmdStashSave(message)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("stash failed: %v", err)}
	}

	return models.ActionResult{OK: true, Message: "stashed uncommitted work"}
}

// CheckoutBranch switches to the given branch. Requires a clean worktree.
func (r *Runner) CheckoutBranch(ctx context.Context, branch string) models.ActionResult {
	if result := r.guardClean(ctx); result != nil {
		return *result
	}

	_, err := r.run(ctx, cmdCheckout(branch)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("checkout %s failed: %v", branch, err)}
	}

	return models.ActionResult{OK: true, Message: "switched to " + branch}
}

// CommitDirtyToNewBranch stashes dirty work, creates a new worktree+branch,
// pops the stash there, commits, pushes upstream, and cleans up.
// The main worktree is restored to a clean state on the original branch.
func (r *Runner) CommitDirtyToNewBranch(ctx context.Context, newBranch, startPoint, remote, message string) models.ActionResult {
	// 1. Unstage + stash everything.
	_, _ = r.run(ctx, cmdResetHead()...) // non-fatal

	_, err := r.run(ctx, cmdStashSave("")...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("stash dirty work failed: %v", err)}
	}

	// 2. Create temporary worktree with new branch.
	result := r.inNewBranchWorktree(ctx, newBranch, startPoint, func(wt *Runner) models.ActionResult {
		// 3. Apply the stash in the worktree (conflicts resolved by staging all).
		wt.run(ctx, cmdStashPop()...) //nolint:errcheck // conflicts are resolved below

		// 4. Stage everything + commit.
		_, _ = wt.run(ctx, cmdAddAll()...)

		_, commitErr := wt.run(ctx, cmdCommit(message)...)
		if commitErr != nil {
			return models.ActionResult{Message: fmt.Sprintf("commit in worktree failed: %v", commitErr)}
		}

		// 5. Push upstream.
		_, pushErr := wt.run(ctx, cmdPushBranchUpstream(remote, newBranch)...)
		if pushErr != nil {
			return models.ActionResult{
				OK:      true,
				Message: fmt.Sprintf("committed to %s but push failed: %v (branch exists locally)", newBranch, pushErr),
			}
		}

		return models.ActionResult{OK: true, Message: fmt.Sprintf("committed dirty work to %s and pushed to %s", newBranch, remote)}
	})

	return result
}

// CommitStashToNewBranch creates a new worktree+branch, applies the given stash
// ref there, commits, pushes upstream, and cleans up. The stash is dropped on success.
//
// The worktree is created from the stash's parent commit (the commit the stash
// was created on top of), so the stash applies cleanly without conflicts.
func (r *Runner) CommitStashToNewBranch(ctx context.Context, stashRef, newBranch, _, remote, message string) models.ActionResult {
	// Resolve the stash's parent commit — the commit it was created on.
	baseCommit, err := r.run(ctx, "rev-parse", stashRef+"^1")
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("cannot resolve base commit for %s: %v", stashRef, err)}
	}

	startPoint := strings.TrimSpace(baseCommit)

	return r.inNewBranchWorktree(ctx, newBranch, startPoint, func(wt *Runner) models.ActionResult {
		// Apply the stash. Since we branched from its parent commit, this should apply cleanly.
		_, applyErr := wt.run(ctx, "stash", "apply", stashRef)
		if applyErr != nil {
			// Fallback: stage everything as-is even with conflicts (archive, not merge).
			wt.run(ctx, "checkout", "--theirs", ".") //nolint:errcheck // best-effort conflict resolution
		}

		// Stage everything (including conflict markers) + commit.
		_, _ = wt.run(ctx, cmdAddAll()...)

		_, commitErr := wt.run(ctx, cmdCommit(message)...)
		if commitErr != nil {
			return models.ActionResult{Message: fmt.Sprintf("commit in worktree failed: %v", commitErr)}
		}

		// Push upstream.
		_, pushErr := wt.run(ctx, cmdPushBranchUpstream(remote, newBranch)...)
		if pushErr != nil {
			// Committed locally but push failed — still useful.
			return models.ActionResult{
				OK:      true,
				Message: fmt.Sprintf("committed stash to %s but push failed: %v (branch exists locally)", newBranch, pushErr),
			}
		}

		// Drop the stash now that it's safely committed and pushed.
		r.run(ctx, "stash", "drop", stashRef) //nolint:errcheck // best-effort: stash index may have shifted

		return models.ActionResult{OK: true, Message: fmt.Sprintf("committed stash to %s and pushed to %s", newBranch, remote)}
	})
}

// GenerateWIPBranch creates a unique wip/YYYY-MM-DD-auto-save-work-NNNN branch name.
// It checks existing branches to avoid collisions.
func (r *Runner) GenerateWIPBranch(ctx context.Context) string {
	prefix := fmt.Sprintf("wip/%s-auto-save-work", time.Now().Format("2006-01-02"))

	// Find existing wip branches to determine the next number.
	out, err := r.run(ctx, "branch", "--list", prefix+"-*", "--format=%(refname:short)")
	if err != nil {
		return prefix + "-0001"
	}

	maxNum := 0

	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract the trailing number.
		idx := strings.LastIndex(line, "-")
		if idx < 0 {
			continue
		}

		var num int
		if _, err := fmt.Sscanf(line[idx+1:], "%d", &num); err == nil && num > maxNum {
			maxNum = num
		}
	}

	return fmt.Sprintf("%s-%04d", prefix, maxNum+1)
}

// inNewBranchWorktree creates a temporary worktree with a new branch,
// runs the function, and cleans up. The worktree is removed afterwards
// but the branch is kept.
func (r *Runner) inNewBranchWorktree(ctx context.Context, newBranch, startPoint string, fn func(wt *Runner) models.ActionResult) models.ActionResult {
	tmpDir, err := os.MkdirTemp("", "janitor-wt-*")
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("cannot create temp directory: %v", err)}
	}

	wtPath := filepath.Join(tmpDir, newBranch)

	defer func() {
		r.run(ctx, cmdWorktreeRemove(wtPath)...) //nolint:errcheck // best-effort
		_ = os.RemoveAll(tmpDir)
	}()

	_, err = r.run(ctx, cmdWorktreeAddNewBranch(wtPath, newBranch, startPoint)...)
	if err != nil {
		return models.ActionResult{Message: fmt.Sprintf("worktree add -b %s failed: %v", newBranch, err)}
	}

	wt := NewRunner(wtPath)
	wt.Timeout = r.Timeout

	// Share the command log with the worktree runner.
	if r.logging {
		wt.logging = true
	}

	result := fn(wt)

	// Merge worktree command log back.
	r.CmdLog = append(r.CmdLog, wt.CmdLog...)

	return result
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

	if r.logging {
		wt.logging = true
	}

	result := fn(wt)
	r.CmdLog = append(r.CmdLog, wt.CmdLog...)

	return result
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

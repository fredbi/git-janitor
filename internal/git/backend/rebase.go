package backend

import (
	"bufio"
	"context"
	"strings"
)

// RebaseCheck holds the result of a dry-run rebase analysis.
type RebaseCheck struct {
	// CanRebase is true if replaying each commit one by one onto target succeeds.
	CanRebase bool

	// CanRebaseSquashed is true if squashing all commits first and then rebasing
	// onto target succeeds. This avoids intermediate conflicts that may occur
	// during a per-commit replay.
	CanRebaseSquashed bool

	// Conflicts lists file paths from whichever strategy was attempted last.
	// If CanRebase is true, this is empty.
	// If CanRebase is false but CanRebaseSquashed is true, this is empty.
	// If both are false, this contains the conflicts from the squashed attempt.
	Conflicts []string

	// FailedStep is the 1-based index of the commit that caused a conflict
	// during the direct per-commit rebase. 0 if direct rebase is clean.
	FailedStep int

	// TotalSteps is the number of commits between the merge-base and the branch tip.
	TotalSteps int
}

// CheckRebase performs a dry-run rebase analysis of branch onto target.
//
// It first attempts a direct per-commit replay. If that fails, it falls back
// to a squash-first strategy (equivalent to merge-tree).
//
// All operations use git plumbing commands (merge-tree, commit-tree, rev-list).
// No refs, worktree, or index are modified. Synthetic commit objects created
// during the check are unreferenced and will be garbage-collected.
//
// Requires git >= 2.38 for merge-tree --write-tree.
func (r *Runner) CheckRebase(ctx context.Context, target, branch string) RebaseCheck {
	// Find the fork point.
	baseOut, err := r.run(ctx, cmdMergeBase(target, branch)...)
	if err != nil {
		return RebaseCheck{}
	}

	mergeBase := strings.TrimSpace(baseOut)

	// List commits to replay (oldest first).
	revOut, err := r.run(ctx, cmdRevListReverse(mergeBase+".."+branch)...)
	if err != nil {
		return RebaseCheck{}
	}

	commits := parseRevList(revOut)
	result := RebaseCheck{TotalSteps: len(commits)}

	if len(commits) == 0 {
		// No commits to rebase — branch is at the merge-base.
		result.CanRebase = true
		result.CanRebaseSquashed = true

		return result
	}

	// Try direct per-commit rebase.
	result.CanRebase, result.FailedStep = r.tryDirectRebase(ctx, target, mergeBase, commits)

	if result.CanRebase {
		result.CanRebaseSquashed = true // if direct works, squash trivially works too

		return result
	}

	// Direct rebase failed — try squash-first strategy.
	// This is a single 3-way merge: merge-base as common ancestor,
	// target as "ours", branch tip as "theirs".
	squash := r.CanMerge(ctx, target, branch)
	result.CanRebaseSquashed = squash.Clean
	result.Conflicts = squash.Conflicts

	return result
}

// tryDirectRebase replays each commit one by one onto target using merge-tree.
// Returns (clean, failedStep). failedStep is 1-based; 0 means all clean.
func (r *Runner) tryDirectRebase(ctx context.Context, target, mergeBase string, commits []string) (bool, int) {
	current := target // the commit we're building on

	for i, commit := range commits {
		// Determine the parent for this commit's cherry-pick.
		// For the first commit, the parent is the merge-base.
		// For subsequent commits, it's the previous commit in the branch.
		parent := mergeBase
		if i > 0 {
			parent = commits[i-1]
		}

		// Simulate cherry-pick: 3-way merge with explicit base.
		treeOut, err := r.run(ctx, cmdMergeTreeWithBase(parent, current, commit)...)
		if err != nil {
			// Conflict at this step.
			return false, i + 1
		}

		tree := strings.TrimSpace(strings.SplitN(treeOut, "\n", 2)[0])

		// Create a synthetic commit so we can use it as the base for the next step.
		// This commit is unreferenced and will be garbage-collected.
		commitOut, err := r.run(ctx, cmdCommitTree(tree, current)...)
		if err != nil {
			return false, i + 1
		}

		current = strings.TrimSpace(commitOut)
	}

	return true, 0
}

// parseRevList parses the output of git rev-list into a slice of commit hashes.
func parseRevList(output string) []string {
	var commits []string

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			commits = append(commits, line)
		}
	}

	return commits
}

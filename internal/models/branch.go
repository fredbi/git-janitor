// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// Branch represents a git branch.
type Branch struct {
	// Name is the short branch name (e.g. "main", "feature/foo").
	Name string

	// IsRemote is true for remote-tracking branches (e.g. "origin/main").
	IsRemote bool

	// IsCurrent is true if this is the currently checked-out branch.
	IsCurrent bool

	// Upstream is the upstream tracking ref (e.g. "origin/main"), if configured.
	Upstream string

	// Ahead is the number of commits ahead of the upstream branch.
	Ahead int

	// Behind is the number of commits behind the upstream branch.
	Behind int

	// Gone is true when the upstream branch has been deleted from the remote.
	Gone bool

	// Merged is true when the branch tip is reachable from the default branch.
	Merged bool

	// MergeCheck reports whether the branch can be cleanly merged into the default branch.
	// nil means not yet checked.
	MergeCheck *MergeCheck

	// RebaseCheck reports whether the branch can be rebased onto the default branch.
	// nil means not yet checked.
	RebaseCheck *RebaseCheck

	// LastCommit is the author date of the most recent commit on this branch.
	LastCommit time.Time

	// Hash is the commit hash at the tip of the branch.
	Hash string
}

// HasUpstream reports whether this branch tracks a remote branch.
func (b Branch) HasUpstream() bool {
	return b.Upstream != ""
}

// MergeCheck holds the result of a dry-run merge check (git merge-tree).
type MergeCheck struct {
	// Clean is true if the merge would succeed without conflicts.
	Clean bool

	// Conflicts lists the file paths with merge conflicts (empty if Clean).
	Conflicts []string
}

// RebaseCheck holds the result of a dry-run rebase analysis.
type RebaseCheck struct {
	// CanRebase is true if replaying each commit one by one onto target succeeds.
	CanRebase bool

	// CanRebaseSquashed is true if squashing all commits first and then rebasing
	// onto target succeeds.
	CanRebaseSquashed bool

	// Conflicts lists file paths from whichever strategy was attempted last.
	Conflicts []string

	// FailedStep is the 1-based index of the commit that caused a conflict
	// during the direct per-commit rebase. 0 if direct rebase is clean.
	FailedStep int

	// TotalSteps is the number of commits between the merge-base and the branch tip.
	TotalSteps int
}

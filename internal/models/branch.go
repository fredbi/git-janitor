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

	// AheadOnly is true when the branch has commits ahead of the default branch
	// but the default branch is an ancestor (no divergence). Set for remote branches
	// during full collection. False means the branch has diverged.
	AheadOnly bool

	// UniqueBytes is the on-disk size of objects (commits, trees, blobs)
	// reachable from this branch tip but not from any other ref. This is
	// the storage that deleting this branch would make unreachable, and
	// that a subsequent deep-clean could reclaim.
	//
	// -1 means not computed. 0 means the branch holds no unique objects
	// (fully subsumed by other refs). Populated only for local,
	// non-default branches during full collection.
	UniqueBytes int64

	// Detail is populated on demand by CollectDetails (nil until requested).
	Detail *BranchDetail
}

// BranchDetail holds on-demand detail information for a branch.
type BranchDetail struct {
	// LastCommitMessage is the subject line of the tip commit.
	LastCommitMessage string

	// DiffStat is the output of git diff --shortstat <default>...<branch>.
	DiffStat string
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

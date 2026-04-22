// SPDX-License-Identifier: Apache-2.0

package models

import (
	"strings"
	"time"
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

	// PrunableReason describes why the worktree is prunable, when known
	// (e.g. "gitdir file points to non-existent location"). May be empty.
	PrunableReason string

	// Locked is true if the worktree is locked against pruning / moving.
	Locked bool

	// LockReason is the reason recorded when the worktree was locked.
	// May be empty.
	LockReason string

	// Dirty reports whether the worktree's working tree has any uncommitted
	// changes (staged, unstaged, or untracked). Populated only during full
	// collection and only for worktrees with an accessible path.
	Dirty bool

	// LastCommit is the author date of the commit at the worktree's HEAD.
	// Zero when the worktree is bare, prunable, or collection failed.
	LastCommit time.Time

	// LastCommitMessage is the subject line of the commit at the worktree's HEAD.
	// Empty when the worktree is bare, prunable, or collection failed.
	LastCommitMessage string
}

// BranchShort returns the short branch name (e.g. "main" from "refs/heads/main").
func (w Worktree) BranchShort() string {
	return strings.TrimPrefix(w.Branch, "refs/heads/")
}

// IsMain reports whether this is the main worktree (the original checkout).
func (w Worktree) IsMain() bool {
	return !w.Bare && w.Branch != "" && !w.Prunable
}

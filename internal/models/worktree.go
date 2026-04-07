// SPDX-License-Identifier: Apache-2.0

package models

import "strings"

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
func (w Worktree) BranchShort() string {
	return strings.TrimPrefix(w.Branch, "refs/heads/")
}

// IsMain reports whether this is the main worktree (the original checkout).
func (w Worktree) IsMain() bool {
	return !w.Bare && w.Branch != "" && !w.Prunable
}

// Package git provides built-in git checks that evaluate
// backend.RepoInfo and produce alerts for the models.
package git

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/git/checks"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

// AllChecks yields all built-in git checks.
func AllChecks() iter.Seq[ifaces.Check] {
	return slices.Values([]ifaces.Check{
		checks.NewHealthGCAdvised(),
		checks.NewSizeRepackAdvised(),
		checks.NewHealthFSCK(),
		checks.NewBranchLagging(),
		checks.NewBranchMergedNotDeleted(),
		checks.NewBranchGoneUpstream(),
		checks.NewBranchNoUpstream(),
		checks.NewBranchDiverged(),
		checks.NewBranchNotMergeable(),
		checks.NewRemoteNoOrigin(),
		checks.NewRemoteMisnamedUpstream(),
		checks.NewTagsLocalOnly(),
		checks.NewTagsRemoteOnly(),
		checks.NewDirtyWorktree(),
		checks.NewActivityStale(),
		checks.NewConfigNoEmail(),
		checks.NewConfigUnsigned(),
		checks.NewLargeFiles(),
		checks.NewBinaryFiles(),
		checks.NewShallow(),
		checks.NewSubmodules(),
		checks.NewLFS(),
		checks.NewInactiveDirty(),
		checks.NewStaleDirty(),
		checks.NewStaleStash(),
		checks.NewInactiveNondefault(),
	})
}

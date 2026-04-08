package git

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/git/actions"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

// AllActions yields all built-in git actions.
func AllActions() iter.Seq[ifaces.Action] {
	return slices.Values([]ifaces.Action{
		actions.NewDeleteBranch(),
		actions.NewUpdateBranch(),
		actions.NewRebaseBranch(),
		actions.NewPushBranch(),
		actions.NewDeleteLocalClone(),
		actions.NewRunGC(),
		actions.NewRunGCAggressive(),
		actions.NewRenameRemote(),
		actions.NewPushTag(),
		actions.NewFetchTags(),
		actions.NewStashDirty(),
		actions.NewCommitDirty(),
		actions.NewCommitStash(),
		actions.NewSwitchDefaultBranch(),
		actions.NewRenameBranch(),
		actions.NewDeleteRemoteBranch(),
		actions.NewDropStash(),
		actions.NewStripRemoteCredentials(),
		actions.NewRebaseRemoteBranch(),
	})
}

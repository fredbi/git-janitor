package alerts

import (
	"context"
	"errors"
)

var _ Alert = &GitLocalBranchLagging{}

type GitLocalBranchLagging struct{}

func (b *GitLocalBranchLagging) Name() string {
	return "git-local-branch-lagging"
}

func (b *GitLocalBranchLagging) Description() string {
	return `Some local branches are lagging their remote tracking.
We could update the local clone.
This check covers all local branches and reports updateable branches as Args.
`
}

func (b *GitLocalBranchLagging) Check(ctx context.Context, repo string) (*AlertResult, error) {
	// TODO: construct an AlertResult with all local branches on repo that are lagging their remote counterpart
	// and could be updated locally. The Args slice would contain all such branch references.
	// The suggested action is "git-pull-branch".
	_, _ = ctx, repo
	return nil, errors.New("not implemented")
}

var _ Alert = &GitLocalBranchDangling{}

type GitLocalBranchDangling struct{}

func (b *GitLocalBranchDangling) Name() string {
	return "git-local-branch-dangling"
}

func (b *GitLocalBranchDangling) Description() string {
	return `Some local branches do not have a remote counterpart.
Your local work could be lost and we could push these to the remote.
This check covers all local branches and reports pushable branches as Args.
`
}

func (b *GitLocalBranchDangling) Check(ctx context.Context, repo string) (*AlertResult, error) {
	// TODO: construct an AlertResult with all local branches on repo that are dangling without a remote counterpart
	// and could be pushed to remote. The Args slice would contain all such branch references.
	// The suggested action is "git-push-branch".
	_, _ = ctx, repo
	return nil, errors.New("not implemented")
}

var _ Alert = &GitLocalBranchResync{}

type GitLocalBranchResync struct{}

func (b *GitLocalBranchResync) Name() string {
	return "git-local-branch-resync"
}

func (b *GitLocalBranchResync) Description() string {
	return `Some local branches are mergeable yet do not branch out from the latest state of the default branch.
Your local work coud be difficult to merge later on, so we should either merge the default branch into
the branch or better so, rebase the branch onto the current defaut branch.

The recommended strategy is:
	* to rebase single-commit branches,
	* to rebase for multi-commit branches with all commits independtly mergeable,
	* to merge the default branch into the branch for other cases of a mergable branch
	
This check covers all local branches and reports mergeable branches with a lagging branch-out commit as Args.
`
}

func (b *GitLocalBranchResync) Check(ctx context.Context, repo string) (*AlertResult, error) {
	// TODO: construct an AlertResult with all local branches on repo that are mergeable into the
	// default branch, and have a branch-out commit from the default branch that is lagging.
	// The Args slice would contain all such branch references.
	// The suggested action is "git-rebase-or-update-branch".
	_, _ = ctx, repo
	return nil, errors.New("not implemented")
}

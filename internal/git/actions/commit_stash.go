// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// CommitStash auto-commits a stash entry to a new WIP branch.
//
// Steps: create worktree+branch → pop stash → commit → push → cleanup.
//
// The subject is the stash ref (e.g. "stash@{0}").
//
// Params (via ActionSubject.Params):
//
//	[0] branch name (empty = auto-generated wip/YYYY-MM-DD-auto-save-work-NNNN)
//	[1] commit message (empty = default message)
type CommitStash struct {
	gitAction
}

func NewCommitStash() CommitStash {
	return CommitStash{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"git-commit-stash",
				"auto-commit a stash entry to a new WIP branch and push upstream",
			),
		},
	}
}

func (CommitStash) ApplyTo() models.SubjectKind { return models.SubjectStash }

func (a CommitStash) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, params)
}

func (a CommitStash) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, params []string) (models.Result, error) {
	if len(params) == 0 {
		return models.Result{}, errors.New("git-commit-stash requires a stash ref as subject")
	}

	stashRef := params[0]
	branchName := paramAt(params, 1)
	message := paramAt(params, 2) //nolint:mnd // index 2 = commit message

	if branchName == "" {
		branchName = r.GenerateWIPBranch(ctx)
	}

	if message == "" {
		message = defaultCommitMessage
	}

	remote := models.DefaultPushRemote(info.Remotes)

	// The runner resolves the start point from the stash's parent commit,
	// so we pass "" — it's ignored.
	result := r.CommitStashToNewBranch(ctx, stashRef, branchName, "", remote, message)

	return result.ToResult(), nil
}

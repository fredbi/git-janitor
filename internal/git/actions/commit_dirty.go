// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

const defaultCommitMessage = "chore: auto-saved stale temporary work"

// CommitDirty auto-commits dirty work to a new WIP branch.
//
// Steps: unstage → stash → create worktree+branch → pop stash → commit → push → cleanup.
//
// Params (via ActionSubject.Params):
//
//	[0] branch name (empty = auto-generated wip/YYYY-MM-DD-auto-save-work-NNNN)
//	[1] commit message (empty = default message)
type CommitDirty struct {
	gitAction
}

func NewCommitDirty() CommitDirty {
	return CommitDirty{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"git-commit-dirty",
				"auto-commit dirty work to a new WIP branch and push upstream",
			),
		},
	}
}

func (CommitDirty) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a CommitDirty) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, params)
}

func (a CommitDirty) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, params []string) (models.Result, error) {
	branchName := paramAt(params, 0)
	message := paramAt(params, 1)

	if branchName == "" {
		branchName = r.GenerateWIPBranch(ctx)
	}

	if message == "" {
		message = defaultCommitMessage
	}

	remote := models.DefaultPushRemote(info.Remotes)
	startPoint := info.Status.Branch

	result := r.CommitDirtyToNewBranch(ctx, branchName, startPoint, remote, message)

	return result.ToResult(), nil
}

// paramAt returns params[i] or "" if out of bounds.
func paramAt(params []string, i int) string {
	if i < len(params) {
		return params[i]
	}

	return ""
}

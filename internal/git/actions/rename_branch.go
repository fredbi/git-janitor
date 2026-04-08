// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// RenameBranch renames a local branch.
//
// Params: [oldName, newName].
// If the branch is currently checked out, git branch -m handles it.
type RenameBranch struct {
	gitAction
}

func NewRenameBranch() RenameBranch {
	return RenameBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"rename-branch",
				"rename a local branch",
			),
		},
	}
}

func (RenameBranch) ApplyTo() models.SubjectKind { return models.SubjectBranch }

func (a RenameBranch) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, params)
}

const minRenameParams = 2

func (a RenameBranch) execute(ctx context.Context, r *backend.Runner, params []string) (models.Result, error) {
	if len(params) < minRenameParams {
		return models.Result{}, errors.New("rename-branch requires [oldName, newName] as params")
	}

	result := r.RenameBranch(ctx, params[0], params[1])

	return result.ToResult(), nil
}

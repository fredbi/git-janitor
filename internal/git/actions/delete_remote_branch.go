// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"strings"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// DeleteRemoteBranch deletes branches on a remote via git push --delete.
//
// Subjects are remote branch names in the form "remote/branch" (e.g. "upstream/feature").
type DeleteRemoteBranch struct {
	gitAction
}

func NewDeleteRemoteBranch() DeleteRemoteBranch {
	return DeleteRemoteBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"delete-remote-branch",
				"delete a branch on a remote (git push --delete)",
			),
		},
	}
}

func (DeleteRemoteBranch) ApplyTo() models.SubjectKind { return models.SubjectBranch }
func (DeleteRemoteBranch) Destructive() bool           { return true }

func (a DeleteRemoteBranch) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a DeleteRemoteBranch) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, subjects []string) (models.Result, error) {
	for _, fullName := range subjects {
		remote, branch, ok := strings.Cut(fullName, "/")
		if !ok {
			return models.Result{}, errors.New("delete-remote-branch: subject must be remote/branch, got: " + fullName)
		}

		result := r.DeleteRemoteBranch(ctx, remote, branch)
		if !result.OK {
			return result.ToResult(), nil
		}
	}

	return models.Result{OK: true, Message: "deleted remote branch(es)"}, nil
}

// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// DeleteLocalTag removes a tag from the local repository without touching
// any remote. Useful for orphan tags that no longer exist on the remote
// (e.g. history carried over from a subtree migration).
type DeleteLocalTag struct {
	gitAction
}

func NewDeleteLocalTag() DeleteLocalTag {
	return DeleteLocalTag{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"delete-local-tag",
				"delete a tag from the local repository (does not touch any remote)",
			),
		},
	}
}

func (DeleteLocalTag) ApplyTo() models.SubjectKind { return models.SubjectTag }
func (DeleteLocalTag) Destructive() bool           { return true }

func (a DeleteLocalTag) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a DeleteLocalTag) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, subjects []string) (models.Result, error) {
	for _, name := range subjects {
		result := r.DeleteLocalTag(ctx, name)
		if !result.OK {
			return result.ToResult(), nil
		}
	}

	return models.Result{OK: true, Message: "all local tags deleted"}, nil
}

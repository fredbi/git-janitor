// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/fs"
	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// DeleteLocalClone removes the local clone directory.
// This is destructive and irreversible.
type DeleteLocalClone struct {
	gitAction
}

func NewDeleteLocalClone() DeleteLocalClone {
	return DeleteLocalClone{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"delete-local-clone",
				"delete a git clone from the local file system",
			),
		},
	}
}

func (DeleteLocalClone) Destructive() bool           { return true }
func (DeleteLocalClone) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a DeleteLocalClone) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a DeleteLocalClone) execute(ctx context.Context, _ *backend.Runner, info *models.RepoInfo, _ []string) (models.Result, error) {
	if err := fs.DeleteLocalRepo(ctx, info); err != nil {
		return models.Result{}, err
	}

	return models.Result{
		OK:      true,
		Message: "deleted local clone: " + info.Path,
	}, nil
}

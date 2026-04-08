// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// DropStash deletes stash entries by ref.
//
// Subjects are stash refs (e.g. "stash@{0}").
// When multiple stashes are selected, they are processed highest-index-first
// by the engine's executePerSubject to avoid index shift.
type DropStash struct {
	gitAction
}

func NewDropStash() DropStash {
	return DropStash{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"drop-stash",
				"delete a stash entry",
			),
		},
	}
}

func (DropStash) ApplyTo() models.SubjectKind { return models.SubjectStash }
func (DropStash) Destructive() bool           { return true }

func (a DropStash) Execute(ctx context.Context, _ *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, subjects)
}

func (a DropStash) execute(ctx context.Context, r *backend.Runner, subjects []string) (models.Result, error) {
	for _, ref := range subjects {
		result := r.DropStash(ctx, ref)
		if !result.OK {
			return result.ToResult(), nil
		}
	}

	return models.Result{OK: true, Message: "dropped stash(es)"}, nil
}

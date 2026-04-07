// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// StashDirty auto-stashes uncommitted work (including untracked files).
// If staged changes exist, they are unstaged first so the stash captures everything.
//
// Optional parameter: stash message (empty = git default).
type StashDirty struct {
	gitAction
}

func NewStashDirty() StashDirty {
	return StashDirty{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"git-stash-dirty",
				"auto-stash uncommitted work (including untracked files)",
			),
		},
	}
}

func (StashDirty) ApplyTo() models.SubjectKind { return models.SubjectRepo }
func (StashDirty) ParamPrompt() string         { return "Stash message (optional):" }

func (a StashDirty) Execute(ctx context.Context, _ *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, params)
}

func (a StashDirty) execute(ctx context.Context, r *backend.Runner, params []string) (models.Result, error) {
	var message string
	if len(params) > 0 {
		message = params[0]
	}

	result := r.StashDirty(ctx, message)

	return result.ToResult(), nil
}

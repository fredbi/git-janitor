// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// PushLocalToUpstream pushes the local default branch to the upstream
// remote without --force and without changing upstream tracking. Intended
// for fork repos where the user's fork has fallen behind the local default.
type PushLocalToUpstream struct {
	gitAction
}

func NewPushLocalToUpstream() PushLocalToUpstream {
	return PushLocalToUpstream{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"push-local-to-upstream",
				"push the local default branch to the upstream remote (no --force)",
			),
		},
	}
}

func (PushLocalToUpstream) ApplyTo() models.SubjectKind { return models.SubjectBranch }
func (PushLocalToUpstream) Destructive() bool           { return true }

func (a PushLocalToUpstream) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a PushLocalToUpstream) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, _ []string) (models.Result, error) {
	if info == nil || info.DefaultBranch == "" {
		return models.Result{}, errors.New("default branch is unknown")
	}

	if models.FindRemote(info.Remotes, models.RemoteUpstream) == nil {
		return models.Result{}, fmt.Errorf("no %s remote configured", models.RemoteUpstream)
	}

	result := r.PushBranchPlain(ctx, models.RemoteUpstream, info.DefaultBranch)

	return result.ToResult(), nil
}

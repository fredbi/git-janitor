// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// PushOriginToUpstream pushes the origin remote's default branch ref
// (origin/<default>) to the upstream remote under the default branch name,
// without --force. Intended for fork repos where the user's fork has fallen
// behind the canonical source on origin.
type PushOriginToUpstream struct {
	gitAction
}

func NewPushOriginToUpstream() PushOriginToUpstream {
	return PushOriginToUpstream{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"push-origin-to-upstream",
				"push origin's default branch ref to the upstream remote (no --force)",
			),
		},
	}
}

func (PushOriginToUpstream) ApplyTo() models.SubjectKind { return models.SubjectBranch }
func (PushOriginToUpstream) Destructive() bool           { return true }

func (a PushOriginToUpstream) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a PushOriginToUpstream) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, _ []string) (models.Result, error) {
	if info == nil || info.DefaultBranch == "" {
		return models.Result{}, errors.New("default branch is unknown")
	}

	if models.FindRemote(info.Remotes, models.RemoteOrigin) == nil {
		return models.Result{}, fmt.Errorf("no %s remote configured", models.RemoteOrigin)
	}

	if models.FindRemote(info.Remotes, models.RemoteUpstream) == nil {
		return models.Result{}, fmt.Errorf("no %s remote configured", models.RemoteUpstream)
	}

	src := models.RemoteOrigin + "/" + info.DefaultBranch
	result := r.PushRefspec(ctx, models.RemoteUpstream, src, info.DefaultBranch)

	return result.ToResult(), nil
}

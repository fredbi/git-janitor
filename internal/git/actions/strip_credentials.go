// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// StripRemoteCredentials updates remote URLs to remove embedded credentials.
//
// Subjects are remote names. Params carry the cleaned URL.
// The engine's executePerSubject provides: params = [remoteName, cleanURL].
type StripRemoteCredentials struct {
	gitAction
}

func NewStripRemoteCredentials() StripRemoteCredentials {
	return StripRemoteCredentials{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"strip-remote-credentials",
				"remove credentials from remote URL",
			),
		},
	}
}

func (StripRemoteCredentials) ApplyTo() models.SubjectKind { return models.SubjectRemote }

func (a StripRemoteCredentials) Execute(ctx context.Context, _ *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, params)
}

const minStripParams = 2

func (a StripRemoteCredentials) execute(ctx context.Context, r *backend.Runner, params []string) (models.Result, error) {
	if len(params) < minStripParams {
		return models.Result{}, errors.New("strip-remote-credentials requires [remoteName, cleanURL]")
	}

	remoteName := params[0]
	cleanURL := params[1]

	result := r.SetRemoteURL(ctx, remoteName, cleanURL)

	return result.ToResult(), nil
}

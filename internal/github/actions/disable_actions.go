// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// DisableForkActions disables GitHub Actions on a fork repository.
type DisableForkActions struct {
	githubAction
}

func NewDisableForkActions() DisableForkActions {
	return DisableForkActions{
		githubAction: githubAction{
			Describer: models.NewDescriber(
				"disable-fork-actions",
				"disable GitHub Actions (CI) on a fork",
			),
		},
	}
}

func (DisableForkActions) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a DisableForkActions) Execute(ctx context.Context, repoInfo *models.RepoInfo, params []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	// Subject is the fork repo full name.
	if len(params) == 0 {
		return models.Result{}, errors.New("disable-fork-actions requires the target repo as subject")
	}

	target := params[0]

	owner, repo, ok := strings.Cut(target, "/")
	if !ok {
		return models.Result{}, fmt.Errorf("invalid repo full name: %q", target)
	}

	if err := runner.DisableActions(ctx, owner, repo); err != nil {
		return models.Result{
			Message: fmt.Sprintf("failed to disable Actions on %s: %v", target, err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: fmt.Sprintf("disabled GitHub Actions on %s", target),
	}, nil
}

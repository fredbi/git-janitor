// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// EnableDeleteBranchOnMerge enables the "Automatically delete head branches"
// setting on a repository. For forks, the subject is the parent (upstream) repo.
type EnableDeleteBranchOnMerge struct {
	githubAction
}

func NewEnableDeleteBranchOnMerge() EnableDeleteBranchOnMerge {
	return EnableDeleteBranchOnMerge{
		githubAction: githubAction{
			Describer: models.NewDescriber(
				"enable-delete-branch-on-merge",
				"enable auto-delete head branches on merge (upstream repo)",
			),
		},
	}
}

func (EnableDeleteBranchOnMerge) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a EnableDeleteBranchOnMerge) Execute(ctx context.Context, repoInfo *models.RepoInfo, params []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	// The subject is the upstream (parent) repo full name, e.g. "owner/repo".
	if len(params) == 0 {
		return models.Result{}, errors.New("enable-delete-branch-on-merge requires the target repo as subject")
	}

	target := params[0]

	owner, repo, ok := strings.Cut(target, "/")
	if !ok {
		return models.Result{}, fmt.Errorf("invalid repo full name: %q", target)
	}

	if err := runner.EnableDeleteBranchOnMerge(ctx, owner, repo); err != nil {
		return models.Result{
			Message: fmt.Sprintf("failed to enable delete-branch-on-merge on %s: %v", target, err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: fmt.Sprintf("enabled auto-delete head branches on %s", target),
	}, nil
}

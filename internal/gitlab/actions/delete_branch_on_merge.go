// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredbi/git-janitor/internal/models"
)

// EnableDeleteBranchOnMerge enables the "Remove source branch after merge"
// setting on a GitLab project.
type EnableDeleteBranchOnMerge struct {
	gitlabAction
}

// NewEnableDeleteBranchOnMerge creates a new EnableDeleteBranchOnMerge action.
func NewEnableDeleteBranchOnMerge() EnableDeleteBranchOnMerge {
	return EnableDeleteBranchOnMerge{
		gitlabAction: gitlabAction{
			Describer: models.NewDescriber(
				"gitlab-enable-delete-branch-on-merge",
				"enable auto-delete source branches on merge",
			),
		},
	}
}

func (EnableDeleteBranchOnMerge) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a EnableDeleteBranchOnMerge) Execute(ctx context.Context, repoInfo *models.RepoInfo, _ []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	data := repoInfo.Platform
	if data.ProjectID == 0 {
		return models.Result{}, errors.New("no GitLab project ID available")
	}

	if err := runner.EnableDeleteBranchOnMerge(ctx, data.ProjectID); err != nil {
		return models.Result{
			Message: fmt.Sprintf("failed to enable delete-branch-on-merge on %s: %v", data.FullName, err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: fmt.Sprintf("enabled auto-delete source branches on %s", data.FullName),
	}, nil
}

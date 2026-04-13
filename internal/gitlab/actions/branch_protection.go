// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredbi/git-janitor/internal/models"
)

// EnableBranchProtection enables a minimal branch protection rule on the
// default branch of a GitLab project.
type EnableBranchProtection struct {
	gitlabAction
}

// NewEnableBranchProtection creates a new EnableBranchProtection action.
func NewEnableBranchProtection() EnableBranchProtection {
	return EnableBranchProtection{
		gitlabAction: gitlabAction{
			Describer: models.NewDescriber(
				"gitlab-enable-branch-protection",
				"enable branch protection on the default branch",
			),
		},
	}
}

func (EnableBranchProtection) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a EnableBranchProtection) Execute(ctx context.Context, repoInfo *models.RepoInfo, _ []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	if !repoInfo.Platform.HasAdminAccess {
		return models.Result{
			Message: fmt.Sprintf("no maintainer access to %s — branch protection requires maintainer", repoInfo.Platform.FullName),
		}, nil
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	data := repoInfo.Platform
	if data.ProjectID == 0 {
		return models.Result{}, errors.New("no GitLab project ID available")
	}

	if err := runner.EnableBranchProtection(ctx, data.ProjectID, data.DefaultBranch); err != nil {
		return models.Result{
			Message: fmt.Sprintf("failed to enable branch protection on %s/%s: %v", data.FullName, data.DefaultBranch, err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: fmt.Sprintf("enabled branch protection on %s (branch: %s)", data.FullName, data.DefaultBranch),
	}, nil
}

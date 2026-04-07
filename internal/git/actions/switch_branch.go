// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/models"
)

// SwitchDefaultBranch checks out the default branch on a clean worktree.
type SwitchDefaultBranch struct {
	gitAction
}

func NewSwitchDefaultBranch() SwitchDefaultBranch {
	return SwitchDefaultBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"git-switch-default-branch",
				"checkout the default branch",
			),
		},
	}
}

func (SwitchDefaultBranch) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a SwitchDefaultBranch) Execute(ctx context.Context, info *models.RepoInfo, _ []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	if info.DefaultBranch == "" {
		return models.Result{Message: "no default branch detected"}, nil
	}

	result := runner.CheckoutBranch(ctx, info.DefaultBranch)

	return result.ToResult(), nil
}

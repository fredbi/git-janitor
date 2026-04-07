// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// DefaultBranchMismatch detects when the GitHub default branch
// differs from the local default branch (detected by git).
type DefaultBranchMismatch struct {
	githubCheck
}

func NewDefaultBranchMismatch() DefaultBranchMismatch {
	return DefaultBranchMismatch{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-default-branch-mismatch",
				"detects when GitHub default branch differs from local",
			),
		},
	}
}

func (c DefaultBranchMismatch) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if repoInfo.Platform == nil {
		return nil, nil
	}

	return c.evaluate(repoInfo.Platform)
}

func (c DefaultBranchMismatch) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {
	// Skip if we don't have local branch info to compare.
	if data.LocalDefaultBranch == "" || data.DefaultBranch == "" {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	if data.DefaultBranch == data.LocalDefaultBranch {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("default branch mismatch: GitHub=%q, local=%q", data.DefaultBranch, data.LocalDefaultBranch),
		Detail:    "The GitHub default branch differs from the local git default branch. This may indicate a recent rename or misconfiguration.",
	}), nil
}

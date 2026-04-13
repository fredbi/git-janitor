// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// DefaultBranchMismatch detects when the GitLab default branch
// differs from the local default branch (detected by git).
type DefaultBranchMismatch struct {
	gitlabCheck
}

// NewDefaultBranchMismatch creates a new DefaultBranchMismatch check.
func NewDefaultBranchMismatch() DefaultBranchMismatch {
	return DefaultBranchMismatch{
		gitlabCheck: gitlabCheck{
			Describer: models.NewDescriber(
				"gitlab-default-branch-mismatch",
				"detects when GitLab default branch differs from local",
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
		Summary:   fmt.Sprintf("default branch mismatch: GitLab=%q, local=%q", data.DefaultBranch, data.LocalDefaultBranch),
		Detail: fmt.Sprintf(
			"The GitLab default branch %q differs from the local default %q. "+
				"Renaming the local branch to match GitLab is recommended.",
			data.DefaultBranch, data.LocalDefaultBranch,
		),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "rename-branch",
			SubjectKind: models.SubjectBranch,
			Subjects: []models.ActionSubject{{
				Subject: data.LocalDefaultBranch,
				Params:  []string{data.DefaultBranch},
			}},
		}},
	}), nil
}

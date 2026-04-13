// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// DeleteBranchOnMergeMissing detects projects where the
// "Remove source branch after merge" setting is not enabled.
//
// When disabled, merged MR branches accumulate and require manual cleanup.
type DeleteBranchOnMergeMissing struct {
	gitlabCheck
}

// NewDeleteBranchOnMergeMissing creates a new DeleteBranchOnMergeMissing check.
func NewDeleteBranchOnMergeMissing() DeleteBranchOnMergeMissing {
	return DeleteBranchOnMergeMissing{
		gitlabCheck: gitlabCheck{
			Describer: models.NewDescriber(
				"gitlab-delete-branch-on-merge",
				"detects projects where source branch is not auto-deleted after merge",
			),
		},
	}
}

func (c DeleteBranchOnMergeMissing) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	fork := forkPlatform(repoInfo)
	if fork == nil {
		return noAlert(c.Name())
	}

	return c.evaluate(fork)
}

func (c DeleteBranchOnMergeMissing) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {
	// Not fetched or no access — skip silently.
	if data.DeleteBranchOnMerge < 0 {
		return noAlert(c.Name())
	}

	// Already enabled — all good.
	if data.DeleteBranchOnMerge > 0 {
		return noAlert(c.Name())
	}

	detail := fmt.Sprintf(
		"Fork %s does not auto-delete source branches after merge. "+
			"Merged MR branches will accumulate.",
		data.FullName,
	)

	var suggestions []models.ActionSuggestion
	if data.HasAdminAccess {
		suggestions = []models.ActionSuggestion{
			repoSuggestion("gitlab-enable-delete-branch-on-merge", simpleSubject(data.FullName)),
		}
	} else {
		detail += " (maintainer access required to change this setting)"
	}

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     fmt.Sprintf("fork %s: auto-delete source branches not enabled", data.FullName),
		Detail:      detail,
		Suggestions: suggestions,
	}), nil
}

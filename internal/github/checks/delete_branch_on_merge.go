// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// DeleteBranchOnMergeMissing detects fork repos where the parent (upstream)
// does not have the "Automatically delete head branches" setting enabled.
//
// This setting is important for forks: when PRs from the fork are merged
// on the upstream repo, the head branch on the fork should be auto-deleted
// to avoid stale branches accumulating.
type DeleteBranchOnMergeMissing struct {
	githubCheck
}

func NewDeleteBranchOnMergeMissing() DeleteBranchOnMergeMissing {
	return DeleteBranchOnMergeMissing{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-delete-branch-on-merge",
				"detects fork repos where upstream lacks auto-delete head branches",
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
		"Fork %s does not auto-delete head branches after merge. "+
			"Merged PR branches will accumulate.",
		data.FullName,
	)

	var suggestions []models.ActionSuggestion
	if data.HasAdminAccess {
		suggestions = []models.ActionSuggestion{
			repoSuggestion("enable-delete-branch-on-merge", simpleSubject(data.FullName)),
		}
	} else {
		detail += " (admin access required to change this setting)"
	}

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     fmt.Sprintf("fork %s: auto-delete head branches not enabled", data.FullName),
		Detail:      detail,
		Suggestions: suggestions,
	}), nil
}

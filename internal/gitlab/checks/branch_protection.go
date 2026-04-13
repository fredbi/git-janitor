// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// BranchProtectionMissing detects projects where the default branch
// has no branch protection rules configured on GitLab.
//
// Note: GitLab auto-protects default branches at project creation,
// so this check will rarely fire unless protection was explicitly removed.
type BranchProtectionMissing struct {
	gitlabCheck
}

// NewBranchProtectionMissing creates a new BranchProtectionMissing check.
func NewBranchProtectionMissing() BranchProtectionMissing {
	return BranchProtectionMissing{
		gitlabCheck: gitlabCheck{
			Describer: models.NewDescriber(
				"gitlab-branch-protection-missing",
				"detects projects with no branch protection on the default branch",
			),
		},
	}
}

func (c BranchProtectionMissing) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if repoInfo.Platform == nil {
		return nil, nil //nolint:nilnil // no platform data, skip
	}

	return c.evaluate(repoInfo.Platform)
}

func (c BranchProtectionMissing) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {
	// Not fetched or no access — skip silently.
	if data.DefaultBranchProtected < 0 {
		return noAlert(c.Name())
	}

	// Protected — all good.
	if data.DefaultBranchProtected > 0 {
		return noAlert(c.Name())
	}

	// Not protected — suggest enabling.
	severity := models.SeverityMedium
	detail := fmt.Sprintf("%s branch %q has no branch protection rules.", data.FullName, data.DefaultBranch)

	// Only suggest the action if the user has admin/maintainer access.
	var suggestions []models.ActionSuggestion
	if data.HasAdminAccess {
		suggestions = []models.ActionSuggestion{
			repoSuggestion("gitlab-enable-branch-protection", simpleSubject(data.FullName)),
		}
	} else {
		detail += " (maintainer access required to enable protection)"
	}

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    severity,
		Summary:     "no branch protection on default branch",
		Detail:      detail,
		Suggestions: suggestions,
	}), nil
}

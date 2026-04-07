// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// ForkActionsEnabled detects fork repos where GitHub Actions (CI) is still
// enabled. CI on forks is usually unnecessary — it consumes Actions minutes
// and often fails due to missing secrets.
type ForkActionsEnabled struct {
	githubCheck
}

func NewForkActionsEnabled() ForkActionsEnabled {
	return ForkActionsEnabled{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-fork-actions-enabled",
				"detects forks where GitHub Actions (CI) is still enabled",
			),
		},
	}
}

func (c ForkActionsEnabled) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	fork := forkPlatform(repoInfo)
	if fork == nil {
		return noAlert(c.Name())
	}

	return c.evaluate(fork)
}

func (c ForkActionsEnabled) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {

	// Not fetched or no access — skip.
	if data.ActionsEnabled < 0 {
		return noAlert(c.Name())
	}

	// Already disabled — all good.
	if data.ActionsEnabled == 0 {
		return noAlert(c.Name())
	}

	detail := fmt.Sprintf(
		"Fork %s has GitHub Actions enabled. CI on forks often fails "+
			"(missing secrets) and wastes Actions minutes.",
		data.FullName,
	)

	var suggestions []models.ActionSuggestion
	if data.HasAdminAccess {
		suggestions = []models.ActionSuggestion{
			repoSuggestion("disable-fork-actions", simpleSubject(data.FullName)),
		}
	} else {
		detail += " (admin access required to disable Actions)"
	}

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     fmt.Sprintf("fork %s: GitHub Actions is enabled", data.FullName),
		Detail:      detail,
		Suggestions: suggestions,
	}), nil
}

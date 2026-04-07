// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

func singleAlert(a models.Alert) iter.Seq[models.Alert] {
	return func(yield func(models.Alert) bool) {
		yield(a)
	}
}

func noAlert(name string) (iter.Seq[models.Alert], error) {
	return singleAlert(models.Alert{
		CheckName: name,
		Severity:  models.SeverityNone,
	}), nil
}

func simpleSubject(names ...string) []models.ActionSubject {
	subjects := make([]models.ActionSubject, 0, len(names))
	for _, name := range names {
		subjects = append(subjects, models.ActionSubject{
			Subject: name,
		})
	}

	return subjects
}

// forkPlatform returns the PlatformInfo for the fork side of the relationship.
// Checks Platform first (origin is the fork), then UpstreamPlatform (upstream is the fork).
// Returns nil if no fork exists.
func forkPlatform(info *models.RepoInfo) *models.PlatformInfo {
	if info.Platform != nil && info.Platform.IsFork {
		return info.Platform
	}

	if info.UpstreamPlatform != nil && info.UpstreamPlatform.IsFork {
		return info.UpstreamPlatform
	}

	return nil
}

func repoSuggestion(action string, subjects []models.ActionSubject) models.ActionSuggestion {
	return models.ActionSuggestion{
		ActionName:  action,
		SubjectKind: models.SubjectRepo,
		Subjects:    subjects,
	}
}

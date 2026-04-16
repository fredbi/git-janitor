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

// forkPlatform returns the PlatformInfo for the user's own fork.
//
// Remote convention in this project: origin points at the source/canonical
// repo (never a target for admin-level fixes), and upstream points at the
// user's personal fork. Actionable checks must therefore prefer
// UpstreamPlatform when present.
//
// Platform (origin) is only used as a fallback for clones that have no
// upstream remote (the user cloned their own fork directly as origin).
//
// In a fork-of-a-fork setup — e.g. stretchr/testify ← go-openapi/testify ←
// fredbi/testify, cloned with origin=go-openapi/testify and
// upstream=fredbi/testify — origin is itself a fork but still must not be
// acted on; only the upstream (fredbi/testify) is the user's fork.
func forkPlatform(info *models.RepoInfo) *models.PlatformInfo {
	if info.UpstreamPlatform != nil && info.UpstreamPlatform.IsFork {
		return info.UpstreamPlatform
	}

	if info.UpstreamPlatform == nil && info.Platform != nil && info.Platform.IsFork {
		return info.Platform
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

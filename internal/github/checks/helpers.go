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

func repoSuggestion(action string, subjects []models.ActionSubject) models.ActionSuggestion {
	return models.ActionSuggestion{
		ActionName:  action,
		SubjectKind: models.SubjectRepo,
		Subjects:    subjects,
	}
}

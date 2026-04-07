package checks

import (
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

func filterBranches(info *models.RepoInfo, filter func(models.Branch) bool) []models.ActionSubject {
	var subjects []models.ActionSubject

	for _, b := range info.Branches {
		if !filter(b) {
			continue
		}

		subjects = append(subjects, models.ActionSubject{Subject: b.Name})
	}

	return subjects
}

func filterTags(info *models.RepoInfo, filter func(models.Tag) bool) []models.ActionSubject {
	var subjects []models.ActionSubject

	for _, t := range info.Tags {
		if !filter(t) {
			continue
		}

		subjects = append(subjects, models.ActionSubject{Subject: t.Name})
	}

	return subjects
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

func branchSuggestion(action string, subjects []models.ActionSubject) models.ActionSuggestion {
	return models.ActionSuggestion{
		ActionName:  action,
		SubjectKind: models.SubjectBranch,
		Subjects:    subjects,
	}
}

func tagSuggestion(action string, subjects []models.ActionSubject) models.ActionSuggestion {
	return models.ActionSuggestion{
		ActionName:  action,
		SubjectKind: models.SubjectTag,
		Subjects:    subjects,
	}
}

func noAlert(name string) (iter.Seq[models.Alert], error) {
	return singleAlert(models.Alert{
		CheckName: name,
		Severity:  models.SeverityNone,
	}), nil
}

func subjectsDetail(subjects []models.ActionSubject) string {
	names := make([]string, 0, len(subjects))
	for _, subject := range subjects {
		names = append(names, subject.Subject)
	}

	return strings.Join(names, ", ")
}

func singleAlert(a models.Alert) iter.Seq[models.Alert] {
	return func(yield func(models.Alert) bool) {
		yield(a)
	}
}

// humanizeBytes formats a byte count into a human-readable string.
func humanizeBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)

	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

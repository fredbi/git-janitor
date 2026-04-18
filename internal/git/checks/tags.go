package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// TagsLocalOnly detects tags that exist locally but not on the remote.
type TagsLocalOnly struct {
	gitCheck
}

func NewTagsLocalOnly() TagsLocalOnly {
	return TagsLocalOnly{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"tags-local-only",
				"detects tags that exist locally but not on the remote",
			),
		},
	}
}

// Evaluate inspects Tags for local-only tags.
func (c TagsLocalOnly) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c TagsLocalOnly) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterTags(info, func(t models.Tag) bool {
		return t.LocalOnly
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	// Two legitimate intents for local-only tags:
	//  - Forgotten-to-push tags → "push-tag" publishes them.
	//  - Orphan tags (e.g. carried over from a subtree migration that no
	//    longer lives on the remote) → "delete-local-tag" cleans them up.
	// The user picks the right action per tag.
	pushSuggestion := tagSuggestion("push-tag", subjects)
	deleteSuggestion := tagSuggestion("delete-local-tag", subjects)

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("%d tag(s) exist locally but not on remote", len(subjects)),
		Detail:    strings.Join(pushSuggestion.SubjectNames(), ", "),
		Suggestions: []models.ActionSuggestion{
			pushSuggestion,
			deleteSuggestion,
		},
	}), nil
}

// TagsRemoteOnly detects tags that exist on the remote but not locally.
type TagsRemoteOnly struct {
	gitCheck
}

func NewTagsRemoteOnly() TagsRemoteOnly {
	return TagsRemoteOnly{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"tags-remote-only",
				"detects tags that exist on the remote but not locally",
			),
		},
	}
}

// Evaluate inspects Tags for remote-only tags.
func (c TagsRemoteOnly) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c TagsRemoteOnly) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterTags(info, func(t models.Tag) bool {
		return t.RemoteOnly
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("%d tag(s) exist on remote but not locally", len(subjects)),
		Detail:    subjectsDetail(subjects),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "fetch-tags",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

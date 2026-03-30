package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/git/backend"
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
func (c TagsLocalOnly) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c TagsLocalOnly) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterTags(info, func(t backend.Tag) bool {
		return t.LocalOnly
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	suggestion := tagSuggestion("push-tag", subjects)
	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     fmt.Sprintf("%d tag(s) exist locally but not on remote", len(subjects)),
		Detail:      strings.Join(suggestion.SubjectNames(), ", "),
		Suggestions: []models.ActionSuggestion{suggestion},
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
func (c TagsRemoteOnly) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c TagsRemoteOnly) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterTags(info, func(t backend.Tag) bool {
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

package gitchecks

import (
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckTagsLocalOnly detects tags that exist locally but not on the remote.
type CheckTagsLocalOnly struct {
	engine.GitCheck
}

// Evaluate inspects Tags for local-only tags.
func (c CheckTagsLocalOnly) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, t := range info.Tags {
		if t.LocalOnly {
			subjects = append(subjects, t.Name)
		}
	}

	if len(subjects) == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityLow,
		Summary:   fmt.Sprintf("%d tag(s) exist locally but not on remote", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "push-tag",
			SubjectKind: engine.SubjectTag,
			Subjects:    subjects,
		}},
	}), nil
}

// CheckTagsRemoteOnly detects tags that exist on the remote but not locally.
type CheckTagsRemoteOnly struct {
	engine.GitCheck
}

// Evaluate inspects Tags for remote-only tags.
func (c CheckTagsRemoteOnly) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var names []string

	for _, t := range info.Tags {
		if t.RemoteOnly {
			names = append(names, t.Name)
		}
	}

	if len(names) == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityLow,
		Summary:   fmt.Sprintf("%d tag(s) exist on remote but not locally", len(names)),
		Detail:    strings.Join(names, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "fetch-tags",
			SubjectKind: engine.SubjectRepo,
			Subjects:    []string{info.Path},
		}},
	}), nil
}

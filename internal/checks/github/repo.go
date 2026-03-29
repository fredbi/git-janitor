// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

// CheckRepoArchived detects repositories archived on GitHub.
// Archived repos are read-only: push, branch, and tag operations will fail.
type CheckRepoArchived struct {
	engine.GitHubCheck
}

func (c CheckRepoArchived) Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error) {
	if !data.IsArchived {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityMedium,
		Summary:   "repository is archived on GitHub",
		Detail:    fmt.Sprintf("%s is archived — read-only, push/branch/tag operations will fail.", data.FullName),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "delete-local-clone",
			SubjectKind: engine.SubjectRepo,
			Subjects:    []string{data.FullName},
		}},
	}), nil
}

// CheckDescriptionMissing detects repos with no GitHub description.
type CheckDescriptionMissing struct {
	engine.GitHubCheck
}

func (c CheckDescriptionMissing) Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error) {
	if data.Description != "" {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityLow,
		Summary:   "no description set on GitHub",
		Detail:    fmt.Sprintf("%s has no description.", data.FullName),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "set-repo-description",
			SubjectKind: engine.SubjectRepo,
			Subjects:    []string{data.FullName},
		}},
	}), nil
}

// CheckVisibilityPrivate reports when a repo is private (informational).
type CheckVisibilityPrivate struct {
	engine.GitHubCheck
}

func (c CheckVisibilityPrivate) Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error) {
	if !data.IsPrivate {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   "repository is private",
	}), nil
}

// CheckRepoForkParent identifies fork parents (informational).
type CheckRepoForkParent struct {
	engine.GitHubCheck
}

func (c CheckRepoForkParent) Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error) {
	if !data.IsFork || data.ParentFullName == "" {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   fmt.Sprintf("fork of %s", data.ParentFullName),
	}), nil
}

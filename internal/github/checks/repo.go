// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// RepoArchived detects repositories archived on GitHub.
// Archived repos are read-only: push, branch, and tag operations will fail.
type RepoArchived struct {
	githubCheck
}

func NewRepoArchived() RepoArchived {
	return RepoArchived{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-repo-archived",
				"detects repositories archived on GitHub (read-only)",
			),
		},
	}
}

func (c RepoArchived) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if repoInfo.Platform == nil {
		return nil, nil
	}

	return c.evaluate(repoInfo.Platform)
}

func (c RepoArchived) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {
	if !data.IsArchived {
		return noAlert(c.Name())
	}

	suggestion := repoSuggestion("delete-local-clone", simpleSubject(data.FullName)) // NOTE: this is a git action

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityMedium,
		Summary:     "repository is archived on GitHub",
		Detail:      data.FullName + " is archived — read-only, push/branch/tag operations will fail.",
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// DescriptionMissing detects repos with no GitHub description.
type DescriptionMissing struct {
	githubCheck
}

func NewDescriptionMissing() DescriptionMissing {
	return DescriptionMissing{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-description-missing",
				"detects repositories with no description on GitHub",
			),
		},
	}
}

func (c DescriptionMissing) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if repoInfo.Platform == nil {
		return nil, nil
	}

	return c.evaluate(repoInfo.Platform)
}

func (c DescriptionMissing) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {
	if data.Description != "" {
		return noAlert(c.Name())
	}

	suggestion := repoSuggestion("set-repo-description", simpleSubject(data.FullName))

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     "no description set on GitHub",
		Detail:      data.FullName + " has no description.",
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// VisibilityPrivate reports when a repo is private (informational).
type VisibilityPrivate struct {
	githubCheck
}

func NewVisibilityPrivate() VisibilityPrivate {
	return VisibilityPrivate{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-visibility-private",
				"reports when a repository is private (informational)",
			),
		},
	}
}

func (c VisibilityPrivate) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if repoInfo.Platform == nil {
		return nil, nil
	}

	return c.evaluate(repoInfo.Platform)
}

func (c VisibilityPrivate) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {
	if !data.IsPrivate {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "repository is private",
	}), nil
}

// RepoForkParent identifies fork parents (informational).
type RepoForkParent struct {
	githubCheck
}

func NewRepoForkParent() RepoForkParent {
	return RepoForkParent{
		githubCheck: githubCheck{
			Describer: models.NewDescriber(
				"github-repo-fork-parent",
				"identifies fork parent repository (informational)",
			),
		},
	}
}

func (c RepoForkParent) Evaluate(_ context.Context, repoInfo *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if repoInfo.Platform == nil {
		return nil, nil
	}

	return c.evaluate(repoInfo.Platform)
}

func (c RepoForkParent) evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error) {
	if !data.IsFork || data.ParentFullName == "" {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "fork of " + data.ParentFullName,
	}), nil
}

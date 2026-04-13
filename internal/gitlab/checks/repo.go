// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// RepoArchived detects projects archived on GitLab.
// Archived projects are read-only: push, branch, and tag operations will fail.
type RepoArchived struct {
	gitlabCheck
}

// NewRepoArchived creates a new RepoArchived check.
func NewRepoArchived() RepoArchived {
	return RepoArchived{
		gitlabCheck: gitlabCheck{
			Describer: models.NewDescriber(
				"gitlab-repo-archived",
				"detects projects archived on GitLab (read-only)",
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

	suggestion := repoSuggestion("delete-local-clone", simpleSubject(data.FullName))

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityMedium,
		Summary:     "project is archived on GitLab",
		Detail:      data.FullName + " is archived — read-only, push/branch/tag operations will fail.",
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// DescriptionMissing detects projects with no GitLab description.
type DescriptionMissing struct {
	gitlabCheck
}

// NewDescriptionMissing creates a new DescriptionMissing check.
func NewDescriptionMissing() DescriptionMissing {
	return DescriptionMissing{
		gitlabCheck: gitlabCheck{
			Describer: models.NewDescriber(
				"gitlab-description-missing",
				"detects projects with no description on GitLab",
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

	suggestion := repoSuggestion("gitlab-set-project-description", simpleSubject(data.FullName))

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     "no description set on GitLab",
		Detail:      data.FullName + " has no description.",
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// VisibilityPrivate reports when a project is private (informational).
type VisibilityPrivate struct {
	gitlabCheck
}

// NewVisibilityPrivate creates a new VisibilityPrivate check.
func NewVisibilityPrivate() VisibilityPrivate {
	return VisibilityPrivate{
		gitlabCheck: gitlabCheck{
			Describer: models.NewDescriber(
				"gitlab-visibility-private",
				"reports when a project is private (informational)",
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
		Summary:   "project is private",
	}), nil
}

// RepoForkParent identifies fork parents (informational).
type RepoForkParent struct {
	gitlabCheck
}

// NewRepoForkParent creates a new RepoForkParent check.
func NewRepoForkParent() RepoForkParent {
	return RepoForkParent{
		gitlabCheck: gitlabCheck{
			Describer: models.NewDescriber(
				"gitlab-repo-fork-parent",
				"identifies fork parent project (informational)",
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

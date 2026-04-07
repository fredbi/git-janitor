// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

const (
	inactiveDirtyThreshold      = 7 * 24 * time.Hour  // 7 days
	staleDirtyThreshold         = 30 * 24 * time.Hour // 30 days
	staleStashThreshold         = 30 * 24 * time.Hour // 30 days
	inactiveNondefaultThreshold = 30 * 24 * time.Hour // 30 days
)

// InactiveDirty detects repos that are dirty but haven't been touched in 7+ days.
// Suggests stashing the inactive dirty work.
type InactiveDirty struct {
	gitCheck
}

func NewInactiveDirty() InactiveDirty {
	return InactiveDirty{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-inactive-dirty",
				"detects dirty repos inactive for 7+ days",
			),
		},
	}
}

func (c InactiveDirty) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.Status.IsDirty() {
		return noAlert(c.Name())
	}

	if info.LastLocalUpdate.IsZero() || time.Since(info.LastLocalUpdate) < inactiveDirtyThreshold {
		return noAlert(c.Name())
	}

	// Don't fire if the stale-dirty check would fire instead (30+ days).
	if time.Since(info.LastLocalUpdate) >= staleDirtyThreshold {
		return noAlert(c.Name())
	}

	stashMsg := "Auto-stash of inactive dirty. On top of:\n" + info.LastCommitMessage

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("dirty worktree inactive for %d days", int(time.Since(info.LastLocalUpdate).Hours()/24)), //nolint:mnd // 24h/day
		Detail:    "uncommitted changes haven't been touched recently — consider stashing",
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "git-stash-dirty",
			SubjectKind: models.SubjectRepo,
			Subjects: []models.ActionSubject{{
				Subject: info.Path,
				Params:  []string{stashMsg},
			}},
		}},
	}), nil
}

// StaleDirty detects repos that are dirty but haven't been touched in 30+ days.
// Suggests committing the stale dirty work to a WIP branch.
type StaleDirty struct {
	gitCheck
}

func NewStaleDirty() StaleDirty {
	return StaleDirty{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-stale-dirty",
				"detects dirty repos inactive for 30+ days",
			),
		},
	}
}

func (c StaleDirty) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.Status.IsDirty() {
		return noAlert(c.Name())
	}

	if info.LastLocalUpdate.IsZero() || time.Since(info.LastLocalUpdate) < staleDirtyThreshold {
		return noAlert(c.Name())
	}

	commitMsg := "Auto-saved stale dirty work.\n\nOn top of:\n" + info.LastCommitMessage

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("dirty worktree stale for %d days", int(time.Since(info.LastLocalUpdate).Hours()/24)), //nolint:mnd // 24h/day
		Detail:    "uncommitted changes haven't been touched in over 30 days — consider saving to a WIP branch",
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "git-commit-dirty",
			SubjectKind: models.SubjectRepo,
			Subjects: []models.ActionSubject{{
				Subject: info.Path,
				Params:  []string{"", commitMsg}, // empty branch name = auto-generate
			}},
		}},
	}), nil
}

// StaleStash detects stash entries older than 30 days.
// Suggests committing the stale stash to a WIP branch.
type StaleStash struct {
	gitCheck
}

func NewStaleStash() StaleStash {
	return StaleStash{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-stale-stash",
				"detects stash entries older than 30 days",
			),
		},
	}
}

func (c StaleStash) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	var staleStashes []models.ActionSubject

	for _, s := range info.Stashes {
		if s.LastUpdatedAt.IsZero() || time.Since(s.LastUpdatedAt) < staleStashThreshold {
			continue
		}

		commitMsg := "Auto-saved stale stash.\n\n" + s.Message

		staleStashes = append(staleStashes, models.ActionSubject{
			Subject: s.Ref,
			Params:  []string{"", commitMsg}, // empty branch name = auto-generate
		})
	}

	if len(staleStashes) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("%d stash(es) older than 30 days", len(staleStashes)),
		Detail:    "old stashes may contain forgotten work — consider saving to WIP branches",
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "git-commit-stash",
			SubjectKind: models.SubjectStash,
			Subjects:    staleStashes,
		}},
	}), nil
}

// InactiveNondefault detects clean repos on a non-default branch
// that haven't been updated in 30+ days.
// Suggests switching to the default branch.
type InactiveNondefault struct {
	gitCheck
}

func NewInactiveNondefault() InactiveNondefault {
	return InactiveNondefault{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-inactive-nondefault",
				"detects clean repos on non-default branch inactive for 30+ days",
			),
		},
	}
}

func (c InactiveNondefault) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	// Only applies to clean repos.
	if info.Status.IsDirty() {
		return noAlert(c.Name())
	}

	// Must have a default branch and be on a different branch.
	if info.DefaultBranch == "" || info.Status.Branch == info.DefaultBranch {
		return noAlert(c.Name())
	}

	// Must be inactive for 30+ days.
	if info.LastLocalUpdate.IsZero() || time.Since(info.LastLocalUpdate) < inactiveNondefaultThreshold {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("on branch %q but inactive for %d days", info.Status.Branch, int(time.Since(info.LastLocalUpdate).Hours()/24)), //nolint:mnd // 24h/day
		Detail:    "clean repo on non-default branch — consider switching back to " + info.DefaultBranch,
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "git-switch-default-branch",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

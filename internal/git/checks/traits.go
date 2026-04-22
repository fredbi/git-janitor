package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// Shallow detects shallow clones (incomplete history).
type Shallow struct {
	gitCheck
}

func NewShallow() Shallow {
	return Shallow{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"traits-shallow",
				"detects shallow clones (incomplete history)",
			),
		},
	}
}

// Evaluate inspects the IsShallow field from RepoInfo.
func (c Shallow) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c Shallow) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.IsShallow {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "repository is a shallow clone",
	}), nil
}

// Submodules detects repositories using git submodules.
type Submodules struct {
	gitCheck
}

func NewSubmodules() Submodules {
	return Submodules{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"traits-submodules",
				"detects repositories using git submodules",
			),
		},
	}
}

// Evaluate inspects the HasSubmodules field from RepoInfo.
func (c Submodules) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c Submodules) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.HasSubmodules {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "repository uses git submodules",
	}), nil
}

// StaleSubmodules detects orphaned directories under .git/modules/
// whose submodule name is no longer referenced by .git/config.
// These are residue from removed or renamed submodules and hold space
// that a standard git gc cannot reclaim.
type StaleSubmodules struct {
	gitCheck
}

func NewStaleSubmodules() StaleSubmodules {
	return StaleSubmodules{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"stale-submodule-dirs",
				"detects orphaned .git/modules/* directories from removed submodules",
			),
		},
	}
}

// Evaluate inspects the StaleSubmoduleDirs field from RepoInfo.
func (c StaleSubmodules) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c StaleSubmodules) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if len(info.StaleSubmoduleDirs) == 0 {
		return noAlert(c.Name())
	}

	var total int64

	const maxReported = 10

	limit := min(len(info.StaleSubmoduleDirs), maxReported)
	lines := make([]string, 0, limit)

	for _, s := range info.StaleSubmoduleDirs {
		total += s.SizeBytes
	}

	for _, s := range info.StaleSubmoduleDirs[:limit] {
		lines = append(lines, fmt.Sprintf("%s (%s)", s.Name, models.FormatBytes(s.SizeBytes)))
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary: fmt.Sprintf("%d orphan .git/modules/* dir(s) using %s",
			len(info.StaleSubmoduleDirs), models.FormatBytes(total)),
		Detail: strings.Join(lines, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "clean-stale-submodule-dirs",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

// worktreeDirtyInactiveThreshold governs when a dirty linked worktree
// is considered "left alone long enough to stash".
const worktreeDirtyInactiveThreshold = 7 * 24 * time.Hour

// WorktreePrunable detects linked worktrees whose working copy has
// disappeared from disk AND are not locked — these can be pruned
// immediately via `git worktree prune`. Locked+prunable worktrees are
// reported separately by WorktreeStaleLocked.
type WorktreePrunable struct {
	gitCheck
}

func NewWorktreePrunable() WorktreePrunable {
	return WorktreePrunable{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-worktree-prunable",
				"detects linked worktrees whose working copy is gone and can be pruned",
			),
		},
	}
}

// Evaluate inspects the Prunable flag on each worktree.
func (c WorktreePrunable) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c WorktreePrunable) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	var lines []string

	for _, w := range info.Worktrees {
		if !w.Prunable || w.Locked {
			continue
		}

		if w.PrunableReason != "" {
			lines = append(lines, fmt.Sprintf("%s (%s)", w.Path, w.PrunableReason))
		} else {
			lines = append(lines, w.Path)
		}
	}

	if len(lines) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   fmt.Sprintf("%d prunable worktree(s)", len(lines)),
		Detail:    strings.Join(lines, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "prune-worktrees",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

// WorktreeStaleLocked detects linked worktrees that are prunable (their
// working directory is gone) but still locked, which prevents
// `git worktree prune` from acting. The suggested fix is to unlock first.
type WorktreeStaleLocked struct {
	gitCheck
}

func NewWorktreeStaleLocked() WorktreeStaleLocked {
	return WorktreeStaleLocked{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-worktree-stale-locked",
				"detects orphaned linked worktrees that cannot be pruned because they are locked",
			),
		},
	}
}

// Evaluate inspects Prunable+Locked combination on each worktree.
func (c WorktreeStaleLocked) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c WorktreeStaleLocked) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	var (
		lines    []string
		subjects []models.ActionSubject
	)

	for _, w := range info.Worktrees {
		if !w.Prunable || !w.Locked {
			continue
		}

		reason := w.LockReason
		if reason == "" {
			reason = "(no reason)"
		}

		lines = append(lines, fmt.Sprintf("%s — locked: %s", w.Path, reason))
		subjects = append(subjects, models.ActionSubject{Subject: w.Path})
	}

	if len(lines) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityMedium,
		Summary:   fmt.Sprintf("%d locked orphan worktree(s)", len(lines)),
		Detail:    strings.Join(lines, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "unlock-worktree",
			SubjectKind: models.SubjectWorktree,
			Subjects:    subjects,
		}},
	}), nil
}

// WorktreeDirty flags any linked (non-main, non-prunable, non-bare)
// worktree that has uncommitted work. High severity — a linked
// worktree is a dedicated branch session; leaving it dirty usually
// indicates forgotten work. No suggested action: the user decides
// whether to commit, stash, or reconcile manually.
type WorktreeDirty struct {
	gitCheck
}

func NewWorktreeDirty() WorktreeDirty {
	return WorktreeDirty{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-worktree-dirty",
				"detects linked worktrees left in a dirty state",
			),
		},
	}
}

// Evaluate inspects the Dirty flag on each linked worktree.
func (c WorktreeDirty) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c WorktreeDirty) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	var lines []string

	for _, w := range info.Worktrees {
		if w.Bare || w.Prunable || w.Path == info.Path {
			continue
		}

		if !w.Dirty {
			continue
		}

		// Skip when also inactive — git-worktree-dirty-inactive handles that.
		if isWorktreeInactive(w) {
			continue
		}

		lines = append(lines, w.Path)
	}

	if len(lines) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityHigh,
		Summary:   fmt.Sprintf("%d linked worktree(s) with uncommitted work", len(lines)),
		Detail:    strings.Join(lines, "; "),
	}), nil
}

// WorktreeDirtyInactive flags linked worktrees that are dirty AND have
// not seen a commit in the last week. Suggests stashing per worktree.
type WorktreeDirtyInactive struct {
	gitCheck
}

func NewWorktreeDirtyInactive() WorktreeDirtyInactive {
	return WorktreeDirtyInactive{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"git-worktree-dirty-inactive",
				"detects linked worktrees left dirty and idle for more than a week",
			),
		},
	}
}

// Evaluate combines Dirty + inactivity per linked worktree.
func (c WorktreeDirtyInactive) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c WorktreeDirtyInactive) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	var (
		lines    []string
		subjects []models.ActionSubject
	)

	for _, w := range info.Worktrees {
		if w.Bare || w.Prunable || w.Path == info.Path {
			continue
		}

		if !w.Dirty || !isWorktreeInactive(w) {
			continue
		}

		lines = append(lines, w.Path)
		subjects = append(subjects, models.ActionSubject{Subject: w.Path})
	}

	if len(lines) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityMedium,
		Summary:   fmt.Sprintf("%d inactive dirty worktree(s)", len(lines)),
		Detail:    strings.Join(lines, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "stash-worktree-dirty",
			SubjectKind: models.SubjectWorktree,
			Subjects:    subjects,
		}},
	}), nil
}

// isWorktreeInactive reports whether a worktree has been idle for more
// than worktreeDirtyInactiveThreshold. Zero LastCommit counts as
// inactive (we can't age it, and a worktree with no commits on HEAD is
// unusual enough to flag).
func isWorktreeInactive(w models.Worktree) bool {
	if w.LastCommit.IsZero() {
		return true
	}

	return time.Since(w.LastCommit) >= worktreeDirtyInactiveThreshold
}

// LFS detects repositories using Git LFS.
type LFS struct {
	gitCheck
}

func NewLFS() LFS {
	return LFS{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"traits-lfs",
				"detects repositories using Git LFS",
			),
		},
	}
}

// Evaluate inspects the HasLFS field from RepoInfo.
func (c LFS) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c LFS) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.HasLFS {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "repository uses Git LFS",
	}), nil
}

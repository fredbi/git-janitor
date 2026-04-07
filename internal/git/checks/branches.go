package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// BranchLagging detects local branches that are behind their remote tracking branch.
type BranchLagging struct {
	gitCheck
}

func NewBranchLagging() BranchLagging {
	return BranchLagging{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"branch-lagging",
				"detects local branches that are behind their remote tracking branch",
			),
		},
	}
}

// Evaluate inspects Branches for local branches with Behind > 0 and an upstream configured.
// The default branch is always checked (it can lag even though it's "merged into itself").
func (c BranchLagging) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c BranchLagging) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterBranches(info, func(b models.Branch) bool {
		if b.IsRemote || !b.HasUpstream() || b.Behind == 0 {
			return false
		}

		// Skip merged feature branches (they should be deleted, not updated).
		// But always check the default branch — it can lag its upstream.
		if b.Merged && b.Name != info.DefaultBranch {
			return false
		}

		return true
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	suggestion := branchSuggestion("update-branch", subjects)

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     fmt.Sprintf("%d branch(es) behind upstream", len(subjects)),
		Detail:      strings.Join(suggestion.SubjectNames(), ", "),
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// BranchMergedNotDeleted detects local branches already merged into the default branch.
type BranchMergedNotDeleted struct {
	gitCheck
}

func NewBranchMergedNotDeleted() BranchMergedNotDeleted {
	return BranchMergedNotDeleted{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"branch-merged-not-deleted",
				"detects local branches already merged into the default branch",
			),
		},
	}
}

// Evaluate inspects Branches for merged local branches that are not the default branch.
func (c BranchMergedNotDeleted) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c BranchMergedNotDeleted) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterBranches(info, func(b models.Branch) bool {
		if !b.IsRemote && b.Merged && b.Name != info.DefaultBranch {
			return true
		}

		return false
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	suggestion := branchSuggestion("delete-branch", subjects)

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityMedium,
		Summary:     fmt.Sprintf("%d merged branch(es) can be deleted", len(subjects)),
		Detail:      strings.Join(suggestion.SubjectNames(), ", "),
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// BranchGoneUpstream detects local branches whose upstream has been deleted from the remote.
type BranchGoneUpstream struct {
	gitCheck
}

func NewBranchGoneUpstream() BranchGoneUpstream {
	return BranchGoneUpstream{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"branch-gone-upstream",
				"detects local branches whose upstream has been deleted from the remote",
			),
		},
	}
}

// Evaluate inspects Branches for local branches with Gone set to true.
func (c BranchGoneUpstream) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c BranchGoneUpstream) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterBranches(info, func(b models.Branch) bool {
		if b.IsRemote || b.Merged || !b.Gone {
			return false
		}

		return true
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	suggestion := branchSuggestion("delete-branch", subjects)

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityMedium,
		Summary:     fmt.Sprintf("%d branch(es) with deleted upstream — remote branch was removed, local can be cleaned up", len(subjects)),
		Detail:      strings.Join(suggestion.SubjectNames(), ", "),
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// BranchNoUpstream detects local branches not tracking any remote branch.
// Skips merged branches (those should be deleted, not pushed) and the current branch.
type BranchNoUpstream struct {
	gitCheck
}

func NewBranchNoUpstream() BranchNoUpstream {
	return BranchNoUpstream{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"branch-no-upstream",
				"detects local branches not tracking any remote branch",
			),
		},
	}
}

// Evaluate inspects Branches for local non-current, non-merged branches without an upstream.
func (c BranchNoUpstream) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c BranchNoUpstream) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterBranches(info, func(b models.Branch) bool {
		if b.IsRemote || b.IsCurrent || b.Merged || b.Name == info.DefaultBranch {
			return false
		}

		if b.Upstream != "" {
			return false
		}

		return true
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	suggestion := branchSuggestion("push-branch", subjects)

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     fmt.Sprintf("%d branch(es) never pushed — no remote tracking configured", len(subjects)),
		Detail:      strings.Join(suggestion.SubjectNames(), ", "),
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// BranchDiverged detects local branches that have diverged from their upstream
// (both ahead and behind). Only suggests rebase when the branch is actually rebasable.
type BranchDiverged struct {
	gitCheck
}

func NewBranchDiverged() BranchDiverged {
	return BranchDiverged{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"branch-diverged",
				"detects local branches that have diverged from their upstream (both ahead and behind)",
			),
		},
	}
}

// Evaluate inspects Branches for local branches with both Ahead > 0 and Behind > 0.
// Rebase is only suggested when RebaseCheck confirms the branch can be rebased
// (directly or via squash).
func (c BranchDiverged) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c BranchDiverged) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	allDiverged := filterBranches(info, func(b models.Branch) bool {
		if b.IsRemote || b.Merged || b.Ahead == 0 || b.Behind == 0 {
			return false
		}

		return true
	})

	if len(allDiverged) == 0 {
		return noAlert(c.Name())
	}

	rebasable := filterBranches(info, func(b models.Branch) bool {
		if b.IsRemote || b.Merged || b.Ahead == 0 || b.Behind == 0 {
			return false
		}

		if b.RebaseCheck != nil && (b.RebaseCheck.CanRebase || b.RebaseCheck.CanRebaseSquashed) {
			return true
		}

		return false
	})

	alert := models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityMedium,
		Summary:   fmt.Sprintf("%d branch(es) diverged from upstream", len(allDiverged)),
		Detail:    subjectsDetail(allDiverged),
	}

	if len(rebasable) > 0 {
		alert.Suggestions = []models.ActionSuggestion{{
			ActionName:  "rebase-branch",
			SubjectKind: models.SubjectBranch,
			Subjects:    rebasable,
		}}
	}

	return singleAlert(alert), nil
}

// BranchNotMergeable detects local branches that cannot be merged or rebased
// onto the default branch. This indicates manual conflict resolution is needed.
type BranchNotMergeable struct {
	gitCheck
}

func NewBranchNotMergeable() BranchNotMergeable {
	return BranchNotMergeable{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"branch-not-mergeable",
				"detects local branches that cannot be merged or rebased onto the default branch",
			),
		},
	}
}

// Evaluate inspects Branches for local branches where both MergeCheck and RebaseCheck
// indicate failure.
func (c BranchNotMergeable) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c BranchNotMergeable) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	subjects := filterBranches(info, func(b models.Branch) bool {
		if b.IsRemote || b.Merged || b.Name == info.DefaultBranch {
			return false
		}

		mergeOK := b.MergeCheck != nil && b.MergeCheck.Clean
		rebaseOK := b.RebaseCheck != nil && (b.RebaseCheck.CanRebase || b.RebaseCheck.CanRebaseSquashed)

		// Only flag if both checks have been performed and both failed.
		if b.MergeCheck != nil && b.RebaseCheck != nil && !mergeOK && !rebaseOK {
			return true
		}

		return false
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityMedium,
		Summary:   fmt.Sprintf("%d branch(es) cannot be merged or rebased", len(subjects)),
		Detail:    "manual conflict resolution needed: " + subjectsDetail(subjects),
	}), nil
}

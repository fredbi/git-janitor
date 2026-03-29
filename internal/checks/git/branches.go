package gitchecks

import (
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckBranchLagging detects local branches that are behind their remote tracking branch.
type CheckBranchLagging struct {
	engine.GitCheck
}

// Evaluate inspects Branches for local branches with Behind > 0 and an upstream configured.
// The default branch is always checked (it can lag even though it's "merged into itself").
func (c CheckBranchLagging) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if b.IsRemote || !b.HasUpstream() || b.Behind == 0 {
			continue
		}

		// Skip merged feature branches (they should be deleted, not updated).
		// But always check the default branch — it can lag its upstream.
		if b.Merged && b.Name != info.DefaultBranch {
			continue
		}

		subjects = append(subjects, b.Name)
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
		Summary:   fmt.Sprintf("%d branch(es) behind upstream", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "update-branch",
			SubjectKind: engine.SubjectBranch,
			Subjects:    subjects,
		}},
	}), nil
}

// CheckBranchMergedNotDeleted detects local branches already merged into the default branch.
type CheckBranchMergedNotDeleted struct {
	engine.GitCheck
}

// Evaluate inspects Branches for merged local branches that are not the default branch.
func (c CheckBranchMergedNotDeleted) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if !b.IsRemote && b.Merged && b.Name != info.DefaultBranch {
			subjects = append(subjects, b.Name)
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
		Severity:  engine.SeverityMedium,
		Summary:   fmt.Sprintf("%d merged branch(es) can be deleted", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "delete-branch",
			SubjectKind: engine.SubjectBranch,
			Subjects:    subjects,
		}},
	}), nil
}

// CheckBranchGoneUpstream detects local branches whose upstream has been deleted from the remote.
type CheckBranchGoneUpstream struct {
	engine.GitCheck
}

// Evaluate inspects Branches for local branches with Gone set to true.
func (c CheckBranchGoneUpstream) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if !b.IsRemote && !b.Merged && b.Gone {
			subjects = append(subjects, b.Name)
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
		Severity:  engine.SeverityMedium,
		Summary:   fmt.Sprintf("%d branch(es) with deleted upstream — remote branch was removed, local can be cleaned up", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "delete-branch",
			SubjectKind: engine.SubjectBranch,
			Subjects:    subjects,
		}},
	}), nil
}

// CheckBranchNoUpstream detects local branches not tracking any remote branch.
// Skips merged branches (those should be deleted, not pushed) and the current branch.
type CheckBranchNoUpstream struct {
	engine.GitCheck
}

// Evaluate inspects Branches for local non-current, non-merged branches without an upstream.
func (c CheckBranchNoUpstream) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if b.IsRemote || b.IsCurrent || b.Merged || b.Name == info.DefaultBranch {
			continue
		}

		if b.Upstream == "" {
			subjects = append(subjects, b.Name)
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
		Summary:   fmt.Sprintf("%d branch(es) never pushed — no remote tracking configured", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "push-branch",
			SubjectKind: engine.SubjectBranch,
			Subjects:    subjects,
		}},
	}), nil
}

// CheckBranchDiverged detects local branches that have diverged from their upstream
// (both ahead and behind). Only suggests rebase when the branch is actually rebasable.
type CheckBranchDiverged struct {
	engine.GitCheck
}

// Evaluate inspects Branches for local branches with both Ahead > 0 and Behind > 0.
// Rebase is only suggested when RebaseCheck confirms the branch can be rebased
// (directly or via squash).
func (c CheckBranchDiverged) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var rebasable []string

	var allDiverged []string

	for _, b := range info.Branches {
		if b.IsRemote || b.Merged || b.Ahead == 0 || b.Behind == 0 {
			continue
		}

		allDiverged = append(allDiverged, b.Name)

		if b.RebaseCheck != nil && (b.RebaseCheck.CanRebase || b.RebaseCheck.CanRebaseSquashed) {
			rebasable = append(rebasable, b.Name)
		}
	}

	if len(allDiverged) == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	alert := engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityMedium,
		Summary:   fmt.Sprintf("%d branch(es) diverged from upstream", len(allDiverged)),
		Detail:    strings.Join(allDiverged, ", "),
	}

	if len(rebasable) > 0 {
		alert.Suggestions = []engine.ActionSuggestion{{
			ActionName:  "rebase-branch",
			SubjectKind: engine.SubjectBranch,
			Subjects:    rebasable,
		}}
	}

	return singleAlert(alert), nil
}

// CheckBranchNotMergeable detects local branches that cannot be merged or rebased
// onto the default branch. This indicates manual conflict resolution is needed.
type CheckBranchNotMergeable struct {
	engine.GitCheck
}

// Evaluate inspects Branches for local branches where both MergeCheck and RebaseCheck
// indicate failure.
func (c CheckBranchNotMergeable) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if b.IsRemote || b.Merged || b.Name == info.DefaultBranch {
			continue
		}

		mergeOK := b.MergeCheck != nil && b.MergeCheck.Clean
		rebaseOK := b.RebaseCheck != nil && (b.RebaseCheck.CanRebase || b.RebaseCheck.CanRebaseSquashed)

		// Only flag if both checks have been performed and both failed.
		if b.MergeCheck != nil && b.RebaseCheck != nil && !mergeOK && !rebaseOK {
			subjects = append(subjects, b.Name)
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
		Severity:  engine.SeverityMedium,
		Summary:   fmt.Sprintf("%d branch(es) cannot be merged or rebased", len(subjects)),
		Detail:    "manual conflict resolution needed: " + strings.Join(subjects, ", "),
	}), nil
}

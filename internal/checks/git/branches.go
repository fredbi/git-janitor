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
func (c CheckBranchLagging) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if !b.IsRemote && b.HasUpstream() && b.Behind > 0 {
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
		if !b.IsRemote && b.Gone {
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
		Summary:   fmt.Sprintf("%d branch(es) with deleted upstream", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "delete-branch",
			SubjectKind: engine.SubjectBranch,
			Subjects:    subjects,
		}},
	}), nil
}

// CheckBranchNoUpstream detects local branches not tracking any remote branch.
type CheckBranchNoUpstream struct {
	engine.GitCheck
}

// Evaluate inspects Branches for local non-current branches without an upstream.
func (c CheckBranchNoUpstream) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if !b.IsRemote && !b.IsCurrent && b.Upstream == "" {
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
		Summary:   fmt.Sprintf("%d branch(es) without upstream tracking", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
	}), nil
}

// CheckBranchDiverged detects local branches that have diverged from their upstream
// (both ahead and behind).
type CheckBranchDiverged struct {
	engine.GitCheck
}

// Evaluate inspects Branches for local branches with both Ahead > 0 and Behind > 0.
func (c CheckBranchDiverged) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	var subjects []string

	for _, b := range info.Branches {
		if !b.IsRemote && b.Ahead > 0 && b.Behind > 0 {
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
		Summary:   fmt.Sprintf("%d branch(es) diverged from upstream", len(subjects)),
		Detail:    strings.Join(subjects, ", "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "rebase-branch",
			SubjectKind: engine.SubjectBranch,
			Subjects:    subjects,
		}},
	}), nil
}

package gitchecks

import (
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckHealthGCAdvised detects when git gc would be beneficial,
// based on loose objects, prune-packable duplicates, pack count, and garbage.
type CheckHealthGCAdvised struct {
	engine.GitCheck
}

// Evaluate inspects the HealthReport from RepoInfo.
func (c CheckHealthGCAdvised) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.Health == nil || !info.Health.GCAdvised {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityLow,
		Summary:   "garbage collection recommended",
		Detail:    strings.Join(info.Health.GCReasons, "; "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "run-gc",
			SubjectKind: engine.SubjectRepo,
			Subjects:    []string{info.Path},
		}},
	}), nil
}

// CheckSizeRepackAdvised detects when git repack would be beneficial,
// based on pack count, loose/packed ratio, .git size, and bloat ratio.
type CheckSizeRepackAdvised struct {
	engine.GitCheck
}

// Evaluate inspects the RepoSize from RepoInfo.
func (c CheckSizeRepackAdvised) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.Size == nil || !info.Size.RepackAdvised {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityLow,
		Summary:   "repository repack recommended",
		Detail:    strings.Join(info.Size.RepackReasons, "; "),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "run-gc-aggressive",
			SubjectKind: engine.SubjectRepo,
			Subjects:    []string{info.Path},
		}},
	}), nil
}

// CheckHealthFSCK detects repository corruption found by git fsck.
type CheckHealthFSCK struct {
	engine.GitCheck
}

// Evaluate inspects the HealthReport for fsck errors.
func (c CheckHealthFSCK) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.Health == nil || len(info.Health.FSCKErrors) == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityHigh,
		Summary:   "repository integrity issues detected",
		Detail:    strings.Join(info.Health.FSCKErrors, "\n"),
	}), nil
}

func singleAlert(a engine.Alert) iter.Seq[engine.Alert] {
	return func(yield func(engine.Alert) bool) {
		yield(a)
	}
}

package gitchecks

import (
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckActivityStale detects repositories with reduced commit activity.
// Produces granular labels based on the inactivity window:
//   - 30d inactive: "low activity"
//   - 90d inactive (Staleness == "stale"): "very low activity"
//   - 360d inactive (Staleness == "dormant"): "repository is stale"
type CheckActivityStale struct {
	engine.GitCheck
}

// Evaluate inspects the Activity from RepoInfo.
func (c CheckActivityStale) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.Activity == nil {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	// For clones: if any branch is behind its upstream, the remote is active.
	if info.Kind == git.KindClone && hasLaggingBranch(info) {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	a := info.Activity

	switch {
	case a.Staleness == git.StalenessDormant:
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityLow,
			Summary:   "repository is stale",
			Detail:    fmt.Sprintf("no merged commit in the past 360 days (last commit: %s)", info.LastCommit.Format("2006-01-02")),
		}), nil

	case a.Staleness == git.StalenessStale:
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityLow,
			Summary:   "very low activity",
			Detail: fmt.Sprintf("no merged commit in the past 90 days (360d: %d commits, last: %s)",
				a.Commits360d, info.LastCommit.Format("2006-01-02")),
		}), nil

	case a.Commits30d == 0 && a.Commits90d > 0:
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityInfo,
			Summary:   "low activity",
			Detail: fmt.Sprintf("no merged commit in the past 30 days (90d: %d, 360d: %d)",
				a.Commits90d, a.Commits360d),
		}), nil

	default:
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}
}

// hasLaggingBranch reports whether any local branch is behind its upstream.
func hasLaggingBranch(info *git.RepoInfo) bool {
	for _, b := range info.Branches {
		if !b.IsRemote && b.HasUpstream() && b.Behind > 0 {
			return true
		}
	}

	return false
}

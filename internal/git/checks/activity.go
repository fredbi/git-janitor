package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// ActivityStale detects repositories with reduced commit activity.
// Produces granular labels based on the inactivity window:
//   - 30d inactive: "low activity"
//   - 90d inactive (Staleness == "stale"): "very low activity"
//   - 360d inactive (Staleness == "dormant"): "repository is stale"
type ActivityStale struct {
	gitCheck
}

func NewActivityStale() ActivityStale {
	return ActivityStale{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"activity-stale",
				"detects repositories with reduced commit activity (30d/90d/360d inactivity)",
			),
		},
	}
}

// Evaluate inspects the Activity from RepoInfo.
func (c ActivityStale) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c ActivityStale) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Activity == nil {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	// For clones: if any branch is behind its upstream, the remote is active.
	if info.Kind == models.RepoKindClone && hasLaggingBranch(info) {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	a := info.Activity

	switch {
	case a.Staleness == models.StalenessDormant:
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityLow,
			Summary:   "repository is stale",
			Detail:    fmt.Sprintf("no merged commit in the past 360 days (last commit: %s)", info.LastCommit.Format("2006-01-02")),
		}), nil

	case a.Staleness == models.StalenessStale:
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityLow,
			Summary:   "very low activity",
			Detail: fmt.Sprintf("no merged commit in the past 90 days (360d: %d commits, last: %s)",
				a.Commits360d, info.LastCommit.Format("2006-01-02")),
		}), nil

	case a.Commits30d == 0 && a.Commits90d > 0:
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityInfo,
			Summary:   "low activity",
			Detail: fmt.Sprintf("no merged commit in the past 30 days (90d: %d, 360d: %d)",
				a.Commits90d, a.Commits360d),
		}), nil

	default:
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}
}

// hasLaggingBranch reports whether any local branch is behind its upstream.
func hasLaggingBranch(info *models.RepoInfo) bool {
	for _, b := range info.Branches {
		if !b.IsRemote && b.HasUpstream() && b.Behind > 0 {
			return true
		}
	}

	return false
}

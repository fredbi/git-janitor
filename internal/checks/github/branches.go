// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

// CheckDefaultBranchMismatch detects when the GitHub default branch
// differs from the local default branch (detected by git).
type CheckDefaultBranchMismatch struct {
	engine.GitHubCheck
}

func (c CheckDefaultBranchMismatch) Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error) {
	// Skip if we don't have local branch info to compare.
	if data.LocalDefaultBranch == "" || data.DefaultBranch == "" {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	if data.DefaultBranch == data.LocalDefaultBranch {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityLow,
		Summary:   fmt.Sprintf("default branch mismatch: GitHub=%q, local=%q", data.DefaultBranch, data.LocalDefaultBranch),
		Detail:    "The GitHub default branch differs from the local git default branch. This may indicate a recent rename or misconfiguration.",
	}), nil
}

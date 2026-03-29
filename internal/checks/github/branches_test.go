// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

func TestCheckDefaultBranchMismatch(t *testing.T) {
	check := CheckDefaultBranchMismatch{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{CheckName: "github-default-branch-mismatch"},
	}}

	t.Run("matching branches", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DefaultBranch = "main"
		data.LocalDefaultBranch = "main"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("mismatched branches", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DefaultBranch = "main"
		data.LocalDefaultBranch = "master"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityLow)
	})

	t.Run("no local branch info", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DefaultBranch = "main"
		// LocalDefaultBranch is "" (not set)
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("no github branch info", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.LocalDefaultBranch = "main"
		// DefaultBranch is "" (not set)
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})
}

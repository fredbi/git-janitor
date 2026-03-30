// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

const (
	testOwner         = "owner"
	testRepo          = "repo"
	testDefaultBranch = "main"
	testRepoURL       = "https://github.com/" + testOwner + "/" + testRepo
)

func TestDefaultBranchMismatch(t *testing.T) {
	check := NewDefaultBranchMismatch()

	t.Run("matching branches", func(t *testing.T) {
		data := backend.NewRepoInfo("owner", testRepo)
		data.DefaultBranch = testDefaultBranch
		data.LocalDefaultBranch = testDefaultBranch
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("mismatched branches", func(t *testing.T) {
		data := backend.NewRepoInfo("owner", testRepo)
		data.DefaultBranch = testDefaultBranch
		data.LocalDefaultBranch = "master"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityLow)
	})

	t.Run("no local branch info", func(t *testing.T) {
		data := backend.NewRepoInfo("owner", testRepo)
		data.DefaultBranch = testDefaultBranch
		// LocalDefaultBranch is "" (not set)
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("no github branch info", func(t *testing.T) {
		data := backend.NewRepoInfo("owner", testRepo)
		data.LocalDefaultBranch = testDefaultBranch
		// DefaultBranch is "" (not set)
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})
}

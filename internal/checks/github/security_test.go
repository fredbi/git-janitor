// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

func TestCheckSecurityNotEnabled(t *testing.T) {
	check := CheckSecurityNotEnabled{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{CheckName: "github-security-not-enabled"},
	}}

	t.Run("all inaccessible with admin", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.HasAdminAccess = true
		// Defaults: all -1
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityMedium)
	})

	t.Run("all inaccessible no admin", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		// Defaults: all -1, no admin
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityInfo)
	})

	t.Run("one accessible", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.SecretScanningAlerts = 0
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("all accessible", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DependabotAlerts = 0
		data.CodeScanningAlerts = 0
		data.SecretScanningAlerts = 0
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})
}

func TestCheckSecurityAlerts(t *testing.T) {
	check := CheckSecurityAlerts{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{CheckName: "github-security-alerts"},
	}}

	t.Run("not fetched", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		// All security fields default to -1
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("no alerts", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DependabotAlerts = 0
		data.CodeScanningAlerts = 0
		data.SecretScanningAlerts = 0
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("dependabot only", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DependabotAlerts = 3
		data.CodeScanningAlerts = 0
		data.SecretScanningAlerts = -1 // not accessible
		data.HTMLURL = "https://github.com/owner/repo"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityHigh)

		if alerts[0].Summary != "3 open security alert(s)" {
			t.Errorf("unexpected summary: %q", alerts[0].Summary)
		}
	})

	t.Run("mixed scanners", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DependabotAlerts = 2
		data.CodeScanningAlerts = 1
		data.SecretScanningAlerts = 1
		data.HTMLURL = "https://github.com/owner/repo"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityHigh)

		if alerts[0].Summary != "4 open security alert(s)" {
			t.Errorf("unexpected summary: %q", alerts[0].Summary)
		}
	})

	t.Run("partial access", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.DependabotAlerts = 5
		data.CodeScanningAlerts = -1 // 403
		data.SecretScanningAlerts = -1 // 403
		data.HTMLURL = "https://github.com/owner/repo"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityHigh)

		if alerts[0].Summary != "5 open security alert(s)" {
			t.Errorf("unexpected summary: %q", alerts[0].Summary)
		}
	})
}

// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestSecurityNotEnabled(t *testing.T) {
	check := NewSecurityNotEnabled()

	t.Run("all inaccessible with admin", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		data.HasAdminAccess = true
		// Defaults: all -1
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityMedium)
	})

	t.Run("all inaccessible no admin", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		// Defaults: all -1, no admin
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityInfo)
	})

	t.Run("one accessible", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		data.SecretScanningAlerts = 0
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("all accessible", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		data.DependabotAlerts = 0
		data.CodeScanningAlerts = 0
		data.SecretScanningAlerts = 0
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})
}

func TestSecurityAlerts(t *testing.T) {
	check := NewSecurityAlerts()

	t.Run("not fetched", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		// All security fields default to -1
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("no alerts", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		data.DependabotAlerts = 0
		data.CodeScanningAlerts = 0
		data.SecretScanningAlerts = 0
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("dependabot only", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		data.DependabotAlerts = 3
		data.CodeScanningAlerts = 0
		data.SecretScanningAlerts = -1 // not accessible
		data.HTMLURL = testRepoURL
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityHigh)

		if alerts[0].Summary != "3 open security alert(s)" {
			t.Errorf("unexpected summary: %q", alerts[0].Summary)
		}
	})

	t.Run("mixed scanners", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		data.DependabotAlerts = 2
		data.CodeScanningAlerts = 1
		data.SecretScanningAlerts = 1
		data.HTMLURL = testRepoURL
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityHigh)

		if alerts[0].Summary != "4 open security alert(s)" {
			t.Errorf("unexpected summary: %q", alerts[0].Summary)
		}
	})

	t.Run("partial access", func(t *testing.T) {
		data := models.NewPlatformInfo("owner", testRepo)
		data.DependabotAlerts = 5
		data.CodeScanningAlerts = -1   // 403
		data.SecretScanningAlerts = -1 // 403
		data.HTMLURL = testRepoURL
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityHigh)

		if alerts[0].Summary != "5 open security alert(s)" {
			t.Errorf("unexpected summary: %q", alerts[0].Summary)
		}
	})
}

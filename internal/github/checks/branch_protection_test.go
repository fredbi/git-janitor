// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestBranchProtectionMissing(t *testing.T) {
	check := NewBranchProtectionMissing()

	t.Run("unknown (not fetched or no admin access) — silent", func(t *testing.T) {
		data := models.NewPlatformInfo(testOwner, testRepo)
		data.DefaultBranch = testDefaultBranch
		// DefaultBranchProtected stays at -1 (the NewPlatformInfo default).
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("protected — silent", func(t *testing.T) {
		data := models.NewPlatformInfo(testOwner, testRepo)
		data.DefaultBranch = testDefaultBranch
		data.DefaultBranchProtected = 1
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("unprotected — fires medium", func(t *testing.T) {
		data := models.NewPlatformInfo(testOwner, testRepo)
		data.DefaultBranch = testDefaultBranch
		data.DefaultBranchProtected = 0
		data.HasAdminAccess = true
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityMedium)

		if len(alerts[0].Suggestions) != 1 {
			t.Fatalf("expected 1 suggestion (admin), got %d", len(alerts[0].Suggestions))
		}
	})

	t.Run("unprotected without admin — fires medium without action", func(t *testing.T) {
		data := models.NewPlatformInfo(testOwner, testRepo)
		data.DefaultBranch = testDefaultBranch
		data.DefaultBranchProtected = 0
		// HasAdminAccess is false.
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityMedium)

		if len(alerts[0].Suggestions) != 0 {
			t.Errorf("expected no suggestion when non-admin, got %d", len(alerts[0].Suggestions))
		}
	})
}

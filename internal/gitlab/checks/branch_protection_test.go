// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestBranchProtectionMissing(t *testing.T) {
	check := NewBranchProtectionMissing()

	t.Run("not fetched", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("protected", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.DefaultBranchProtected = 1
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("not protected with admin", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.DefaultBranchProtected = 0
		data.DefaultBranch = "main"
		data.FullName = "group/project"
		data.HasAdminAccess = true
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityMedium)

		if len(alerts[0].Suggestions) == 0 {
			t.Error("expected action suggestion when user has admin access")
		}
	})

	t.Run("not protected without admin", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.DefaultBranchProtected = 0
		data.DefaultBranch = "main"
		data.FullName = "group/project"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityMedium)

		if len(alerts[0].Suggestions) != 0 {
			t.Error("expected no action suggestion when user lacks admin access")
		}
	})
}

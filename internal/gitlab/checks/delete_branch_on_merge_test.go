// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestDeleteBranchOnMergeMissing(t *testing.T) {
	check := NewDeleteBranchOnMergeMissing()

	t.Run("not fetched", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.IsFork = true
		data.ParentFullName = "upstream/project"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("enabled", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.IsFork = true
		data.ParentFullName = "upstream/project"
		data.DeleteBranchOnMerge = 1
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("disabled with admin", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.IsFork = true
		data.ParentFullName = "upstream/project"
		data.DeleteBranchOnMerge = 0
		data.FullName = "group/project"
		data.HasAdminAccess = true
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityLow)

		if len(alerts[0].Suggestions) == 0 {
			t.Error("expected action suggestion when user has admin access")
		}
	})

	t.Run("disabled without admin", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.IsFork = true
		data.ParentFullName = "upstream/project"
		data.DeleteBranchOnMerge = 0
		data.FullName = "group/project"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityLow)

		if len(alerts[0].Suggestions) != 0 {
			t.Error("expected no action suggestion when user lacks admin access")
		}
	})
}

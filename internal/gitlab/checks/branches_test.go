// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestDefaultBranchMismatch(t *testing.T) {
	check := NewDefaultBranchMismatch()

	t.Run("matching branches", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.DefaultBranch = "main"
		data.LocalDefaultBranch = "main"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("mismatched branches", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.DefaultBranch = "main"
		data.LocalDefaultBranch = "master"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityLow)
	})

	t.Run("missing local branch info", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.DefaultBranch = "main"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})
}

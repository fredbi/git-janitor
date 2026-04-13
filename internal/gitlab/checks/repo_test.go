// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestRepoArchived(t *testing.T) {
	check := NewRepoArchived()

	t.Run("not archived", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("archived", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.IsArchived = true
		data.FullName = "group/project"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityMedium)
	})
}

func TestDescriptionMissing(t *testing.T) {
	check := NewDescriptionMissing()

	t.Run("has description", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.Description = "A cool project"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("no description", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityLow)
	})
}

func TestVisibilityPrivate(t *testing.T) {
	check := NewVisibilityPrivate()

	t.Run("public", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("private", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		data.IsPrivate = true
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityInfo)
	})
}

func TestRepoForkParent(t *testing.T) {
	check := NewRepoForkParent()

	t.Run("not a fork", func(t *testing.T) {
		data := models.NewPlatformInfo("group", "project")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityNone)
	})

	t.Run("fork with parent", func(t *testing.T) {
		data := models.NewPlatformInfo("fredbi", "go-swagger")
		data.IsFork = true
		data.ParentFullName = "go-swagger/go-swagger"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, models.SeverityInfo)

		if len(alerts) == 0 {
			t.Fatal("expected at least one alert")
		}

		if alerts[0].Summary == "" {
			t.Error("expected non-empty summary")
		}
	})
}

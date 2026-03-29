// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

func TestCheckRepoArchived(t *testing.T) {
	check := CheckRepoArchived{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{CheckName: "github-repo-archived"},
	}}

	t.Run("not archived", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("archived", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.IsArchived = true
		data.FullName = "owner/repo"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityMedium)
	})
}

func TestCheckDescriptionMissing(t *testing.T) {
	check := CheckDescriptionMissing{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{CheckName: "github-description-missing"},
	}}

	t.Run("has description", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.Description = "A cool project"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("no description", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityLow)
	})
}

func TestCheckVisibilityPrivate(t *testing.T) {
	check := CheckVisibilityPrivate{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{CheckName: "github-visibility-private"},
	}}

	t.Run("public", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("private", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		data.IsPrivate = true
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityInfo)
	})
}

func TestCheckRepoForkParent(t *testing.T) {
	check := CheckRepoForkParent{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{CheckName: "github-repo-fork-parent"},
	}}

	t.Run("not a fork", func(t *testing.T) {
		data := github.NewRepoData("owner", "repo")
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityNone)
	})

	t.Run("fork with parent", func(t *testing.T) {
		data := github.NewRepoData("fredbi", "go-swagger")
		data.IsFork = true
		data.ParentFullName = "go-swagger/go-swagger"
		alerts := collectAlerts(t, check, data)
		requireSeverity(t, alerts, engine.SeverityInfo)

		if len(alerts) == 0 {
			t.Fatal("expected at least one alert")
		}

		if alerts[0].Summary == "" {
			t.Error("expected non-empty summary")
		}
	})
}

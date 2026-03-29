// SPDX-License-Identifier: Apache-2.0

package githubchecks

import (
	"iter"
	"testing"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/github"
)

// githubEvaluator is the interface satisfied by all GitHub checks.
type githubEvaluator interface {
	Evaluate(data *github.RepoData) (iter.Seq[engine.Alert], error)
}

// collectAlerts runs a check and collects all alerts into a slice.
func collectAlerts(t *testing.T, check githubEvaluator, data *github.RepoData) []engine.Alert {
	t.Helper()

	seq, err := check.Evaluate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var alerts []engine.Alert
	if seq != nil {
		for a := range seq {
			alerts = append(alerts, a)
		}
	}

	return alerts
}

// requireSeverity asserts that the first alert has the expected severity.
func requireSeverity(t *testing.T, alerts []engine.Alert, want engine.Severity) {
	t.Helper()

	if len(alerts) == 0 {
		t.Fatal("expected at least one alert")
	}

	if got := alerts[0].Severity; got != want {
		t.Errorf("severity: got %v, want %v", got, want)
	}
}

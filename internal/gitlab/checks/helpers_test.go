// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"iter"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

// gitlabEvaluator is the interface satisfied by all GitLab checks.
type gitlabEvaluator interface {
	evaluate(data *models.PlatformInfo) (iter.Seq[models.Alert], error)
}

// collectAlerts runs a check and collects all alerts into a slice.
func collectAlerts(t *testing.T, check gitlabEvaluator, data *models.PlatformInfo) []models.Alert {
	t.Helper()

	seq, err := check.evaluate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var alerts []models.Alert
	if seq != nil {
		for a := range seq {
			alerts = append(alerts, a)
		}
	}

	return alerts
}

// requireSeverity asserts that the first alert has the expected severity.
func requireSeverity(t *testing.T, alerts []models.Alert, want models.Severity) {
	t.Helper()

	if len(alerts) == 0 {
		t.Fatal("expected at least one alert")
	}

	if got := alerts[0].Severity; got != want {
		t.Errorf("severity: got %v, want %v", got, want)
	}
}

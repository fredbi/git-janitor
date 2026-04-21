// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestStaleSubmodules(t *testing.T) {
	check := NewStaleSubmodules()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name:         "no stale dirs — silent",
			info:         &models.RepoInfo{},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "one orphan — fires low",
			info: &models.RepoInfo{StaleSubmoduleDirs: []models.StaleSubmoduleDir{
				{Name: "vendor/old", Path: "/tmp/r/.git/modules/vendor/old", SizeBytes: 5 * 1024 * 1024},
			}},
			wantSeverity: models.SeverityLow,
		},
		{
			name: "multiple orphans — fires low with aggregated size",
			info: &models.RepoInfo{StaleSubmoduleDirs: []models.StaleSubmoduleDir{
				{Name: "a", Path: "/tmp/r/.git/modules/a", SizeBytes: 1 * 1024 * 1024},
				{Name: "b", Path: "/tmp/r/.git/modules/b", SizeBytes: 2 * 1024 * 1024},
			}},
			wantSeverity: models.SeverityLow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seq, err := check.Evaluate(context.Background(), tc.info)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got models.Severity

			for a := range seq {
				got = a.Severity
				break
			}

			if got != tc.wantSeverity {
				t.Errorf("severity: got %v, want %v", got, tc.wantSeverity)
			}
		})
	}
}

func TestStaleSubmodulesSuggestion(t *testing.T) {
	check := NewStaleSubmodules()
	info := &models.RepoInfo{
		Path: "/tmp/repo",
		StaleSubmoduleDirs: []models.StaleSubmoduleDir{
			{Name: "vendor/old", Path: "/tmp/repo/.git/modules/vendor/old", SizeBytes: 5 * 1024 * 1024},
		},
	}

	seq, err := check.Evaluate(context.Background(), info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var alert models.Alert

	for a := range seq {
		alert = a
		break
	}

	if len(alert.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(alert.Suggestions))
	}

	s := alert.Suggestions[0]
	if s.ActionName != "clean-stale-submodule-dirs" {
		t.Errorf("action name: got %q, want clean-stale-submodule-dirs", s.ActionName)
	}

	if s.SubjectKind != models.SubjectRepo {
		t.Errorf("subject kind: got %v, want SubjectRepo", s.SubjectKind)
	}
}

// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestSizeRepackAdvised(t *testing.T) {
	check := NewSizeRepackAdvised()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name:         "no size report — silent",
			info:         &models.RepoInfo{Size: nil},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "not advised — silent",
			info: &models.RepoInfo{Size: &models.RepoSize{
				RepackAdvised: false,
				GitDirBytes:   500 * 1024 * 1024,
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "advised but .git below 10 MiB floor — silent",
			info: &models.RepoInfo{Size: &models.RepoSize{
				RepackAdvised: true,
				RepackReasons: []string{"21 pack files (consolidation recommended)"},
				GitDirBytes:   9 * 1024 * 1024,
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "advised and .git at floor — fires",
			info: &models.RepoInfo{Size: &models.RepoSize{
				RepackAdvised: true,
				RepackReasons: []string{"21 pack files (consolidation recommended)"},
				GitDirBytes:   gcAdviceMinGitDirBytes,
			}},
			wantSeverity: models.SeverityLow,
		},
		{
			name: "advised and .git well above floor — fires",
			info: &models.RepoInfo{Size: &models.RepoSize{
				RepackAdvised: true,
				RepackReasons: []string{"loose bloat"},
				GitDirBytes:   50 * 1024 * 1024,
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

func TestHealthGCAdvised(t *testing.T) {
	check := NewHealthGCAdvised()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name:         "no health report — silent",
			info:         &models.RepoInfo{Health: nil},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "not advised — silent",
			info: &models.RepoInfo{
				Health: &models.HealthReport{GCAdvised: false},
				Size:   &models.RepoSize{GitDirBytes: 50 * 1024 * 1024},
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "advised but .git below 10 MiB floor — silent",
			info: &models.RepoInfo{
				Health: &models.HealthReport{GCAdvised: true, GCReasons: []string{"151 objects are both loose and packed (prune-packable)"}},
				Size:   &models.RepoSize{GitDirBytes: 5 * 1024 * 1024},
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "advised, no size report — fires (don't silence on absence of size data)",
			info: &models.RepoInfo{
				Health: &models.HealthReport{GCAdvised: true, GCReasons: []string{"too many loose objects"}},
				Size:   nil,
			},
			wantSeverity: models.SeverityLow,
		},
		{
			name: "advised and .git above floor — fires",
			info: &models.RepoInfo{
				Health: &models.HealthReport{GCAdvised: true, GCReasons: []string{"too many loose objects"}},
				Size:   &models.RepoSize{GitDirBytes: 50 * 1024 * 1024},
			},
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

func TestUnreachableBloat(t *testing.T) {
	check := NewUnreachableBloat()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name:         "no size report — silent",
			info:         &models.RepoInfo{Size: nil},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "not bloated — silent",
			info: &models.RepoInfo{Size: &models.RepoSize{
				UnreachableBloat: false,
				GitDirBytes:      500 * 1024 * 1024,
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "bloated but .git below 10 MiB floor — silent",
			info: &models.RepoInfo{Size: &models.RepoSize{
				UnreachableBloat:        true,
				UnreachableBloatReasons: []string{".git (9 MiB) is 3.0x larger than reachable objects (3 MiB)"},
				GitDirBytes:             9 * 1024 * 1024,
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "bloated and .git well above floor — fires",
			info: &models.RepoInfo{Size: &models.RepoSize{
				UnreachableBloat:        true,
				UnreachableBloatReasons: []string{".git (200 MiB) is 4.0x larger than reachable objects (50 MiB)"},
				GitDirBytes:             200 * 1024 * 1024,
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

func TestUnreachableBloatSuggestion(t *testing.T) {
	check := NewUnreachableBloat()
	info := &models.RepoInfo{
		Path: "/tmp/repo",
		Size: &models.RepoSize{
			UnreachableBloat:        true,
			UnreachableBloatReasons: []string{".git (200 MiB) is 4.0x larger than reachable objects (50 MiB)"},
			GitDirBytes:             200 * 1024 * 1024,
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
	if s.ActionName != "run-gc-deep-clean" {
		t.Errorf("action name: got %q, want run-gc-deep-clean", s.ActionName)
	}

	if s.SubjectKind != models.SubjectRepo {
		t.Errorf("subject kind: got %v, want SubjectRepo", s.SubjectKind)
	}
}

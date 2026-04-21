// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"strings"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestBranchUniqueSize(t *testing.T) {
	check := NewBranchUniqueSize()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name:         "no branches — silent",
			info:         &models.RepoInfo{DefaultBranch: "master"},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "all branches below threshold — silent",
			info: &models.RepoInfo{
				DefaultBranch: "master",
				Branches: []models.Branch{
					{Name: "master", UniqueBytes: 0},
					{Name: "feature/a", UniqueBytes: 500 * 1024}, // 500 KB < 1 MiB
				},
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "remote-tracking and default are excluded",
			info: &models.RepoInfo{
				DefaultBranch: "master",
				Branches: []models.Branch{
					{Name: "master", UniqueBytes: 50 * 1024 * 1024},
					{Name: "origin/master", IsRemote: true, UniqueBytes: 50 * 1024 * 1024},
				},
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "one branch above threshold — fires info",
			info: &models.RepoInfo{
				DefaultBranch: "master",
				Branches: []models.Branch{
					{Name: "master", UniqueBytes: 0},
					{Name: "feature/big", UniqueBytes: 10 * 1024 * 1024},
				},
			},
			wantSeverity: models.SeverityInfo,
		},
		{
			name: "uncomputed (-1) excluded",
			info: &models.RepoInfo{
				DefaultBranch: "master",
				Branches: []models.Branch{
					{Name: "master", UniqueBytes: -1},
					{Name: "feature/a", UniqueBytes: -1},
				},
			},
			wantSeverity: models.SeverityNone,
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

func TestBranchUniqueSize_Sorting(t *testing.T) {
	check := NewBranchUniqueSize()

	info := &models.RepoInfo{
		DefaultBranch: "master",
		Branches: []models.Branch{
			{Name: "small", UniqueBytes: 2 * 1024 * 1024},
			{Name: "big", UniqueBytes: 50 * 1024 * 1024},
			{Name: "medium", UniqueBytes: 10 * 1024 * 1024},
			{Name: "tiny", UniqueBytes: 500 * 1024}, // below threshold
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

	if alert.Severity != models.SeverityInfo {
		t.Fatalf("severity: got %v, want Info", alert.Severity)
	}

	// Detail should list big first, then medium, then small. "tiny" excluded.
	if got := alert.Detail; !strings.HasPrefix(got, "big (") {
		t.Errorf("sort: detail should start with 'big (...)': %q", got)
	}

	if strings.Contains(alert.Detail, "tiny") {
		t.Errorf("tiny should have been filtered out by min threshold: %q", alert.Detail)
	}

	if len(alert.Suggestions) != 0 {
		t.Errorf("info-only check should not suggest actions, got %d", len(alert.Suggestions))
	}
}

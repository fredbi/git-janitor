// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestWorktreePrunable(t *testing.T) {
	check := NewWorktreePrunable()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name:         "no worktrees — silent",
			info:         &models.RepoInfo{},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "none prunable — silent",
			info: &models.RepoInfo{Worktrees: []models.Worktree{
				{Path: "/tmp/main", Branch: "refs/heads/main"},
				{Path: "/tmp/feature", Branch: "refs/heads/feature"},
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "one prunable, not locked — fires info",
			info: &models.RepoInfo{Worktrees: []models.Worktree{
				{Path: "/tmp/main", Branch: "refs/heads/main"},
				{Path: "/tmp/gone", Prunable: true, PrunableReason: "gitdir file points to non-existent location"},
			}},
			wantSeverity: models.SeverityInfo,
		},
		{
			name: "prunable but locked — silent (stale-locked owns that case)",
			info: &models.RepoInfo{Worktrees: []models.Worktree{
				{Path: "/tmp/main", Branch: "refs/heads/main"},
				{Path: "/tmp/gone", Prunable: true, Locked: true, LockReason: "pinned"},
			}},
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

func TestWorktreePrunable_Suggestion(t *testing.T) {
	check := NewWorktreePrunable()
	info := &models.RepoInfo{
		Path: "/tmp/repo",
		Worktrees: []models.Worktree{
			{Path: "/tmp/gone", Prunable: true},
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

	if alert.Suggestions[0].ActionName != "prune-worktrees" {
		t.Errorf("action name: got %q, want prune-worktrees", alert.Suggestions[0].ActionName)
	}

	if alert.Suggestions[0].SubjectKind != models.SubjectRepo {
		t.Errorf("subject kind: got %v, want SubjectRepo", alert.Suggestions[0].SubjectKind)
	}
}

// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"testing"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestWorktreeStaleLocked(t *testing.T) {
	check := NewWorktreeStaleLocked()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name: "prunable but not locked — silent",
			info: &models.RepoInfo{Worktrees: []models.Worktree{
				{Path: "/tmp/gone", Prunable: true},
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "locked but not prunable — silent",
			info: &models.RepoInfo{Worktrees: []models.Worktree{
				{Path: "/tmp/live", Locked: true, LockReason: "pinned"},
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "prunable AND locked — fires medium",
			info: &models.RepoInfo{Worktrees: []models.Worktree{
				{Path: "/tmp/gone", Prunable: true, Locked: true, LockReason: "waiting for review"},
			}},
			wantSeverity: models.SeverityMedium,
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

func TestWorktreeDirty(t *testing.T) {
	check := NewWorktreeDirty()

	now := time.Now()
	recent := now.Add(-1 * time.Hour)
	ancient := now.Add(-30 * 24 * time.Hour)

	main := "/tmp/repo"

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name: "main worktree dirty — silent (handled by dirty-worktree)",
			info: &models.RepoInfo{Path: main, Worktrees: []models.Worktree{
				{Path: main, Branch: "refs/heads/main", Dirty: true, LastCommit: recent},
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "linked worktree dirty + recent — fires high",
			info: &models.RepoInfo{Path: main, Worktrees: []models.Worktree{
				{Path: main, Branch: "refs/heads/main"},
				{Path: "/tmp/wt", Branch: "refs/heads/feature", Dirty: true, LastCommit: recent},
			}},
			wantSeverity: models.SeverityHigh,
		},
		{
			name: "linked worktree dirty + inactive — silent (dirty-inactive owns it)",
			info: &models.RepoInfo{Path: main, Worktrees: []models.Worktree{
				{Path: main, Branch: "refs/heads/main"},
				{Path: "/tmp/wt", Branch: "refs/heads/feature", Dirty: true, LastCommit: ancient},
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "prunable linked — silent (prunable workflow)",
			info: &models.RepoInfo{Path: main, Worktrees: []models.Worktree{
				{Path: main, Branch: "refs/heads/main"},
				{Path: "/tmp/wt", Prunable: true, Dirty: true, LastCommit: recent},
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

func TestWorktreeDirtyInactive(t *testing.T) {
	check := NewWorktreeDirtyInactive()

	now := time.Now()
	recent := now.Add(-1 * time.Hour)
	ancient := now.Add(-30 * 24 * time.Hour)
	main := "/tmp/repo"

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name: "linked dirty + recent — silent",
			info: &models.RepoInfo{Path: main, Worktrees: []models.Worktree{
				{Path: main, Branch: "refs/heads/main"},
				{Path: "/tmp/wt", Dirty: true, LastCommit: recent},
			}},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "linked dirty + inactive — fires medium",
			info: &models.RepoInfo{Path: main, Worktrees: []models.Worktree{
				{Path: main, Branch: "refs/heads/main"},
				{Path: "/tmp/wt", Dirty: true, LastCommit: ancient},
			}},
			wantSeverity: models.SeverityMedium,
		},
		{
			name: "linked clean + inactive — silent",
			info: &models.RepoInfo{Path: main, Worktrees: []models.Worktree{
				{Path: main, Branch: "refs/heads/main"},
				{Path: "/tmp/wt", Dirty: false, LastCommit: ancient},
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

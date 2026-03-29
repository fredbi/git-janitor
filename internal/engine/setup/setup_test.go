package setup

import (
	"context"
	"testing"
	"time"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

func TestNewEngine_RegistrationsPresent(t *testing.T) {
	e := NewEngine()

	// Checks should be registered.
	if e.Checks.Len() == 0 {
		t.Fatal("no checks registered")
	}

	for name, check := range e.Checks.All() {
		t.Logf("check: %s (kind=%d) — %s", name, check.Kind(), check.Description())
	}

	// Actions should be registered.
	if e.Actions.Len() == 0 {
		t.Fatal("no actions registered")
	}

	for name, action := range e.Actions.All() {
		t.Logf("action: %s (kind=%d, applyTo=%s, destructive=%v) — %s",
			name, action.Kind(), action.ApplyTo(), action.Destructive(), action.Description())
	}
}

func TestFullSlice_HealthGCAdvised(t *testing.T) {
	e := NewEngine()

	// Synthetic RepoInfo with GC advised.
	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Health: &git.HealthReport{
			OK:           true,
			GCAdvised:    true,
			GCReasons:    []string{"4000 loose objects (threshold: 6700)"},
			LooseObjects: 4000,
		},
	}

	// Run all checks.
	alerts := e.EvaluateRepo(context.Background(), info, nil)

	// Find the health-gc-advised alert.
	var gcAlert *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "health-gc-advised" && alerts[i].Severity != engine.SeverityNone {
			gcAlert = &alerts[i]

			break
		}
	}

	if gcAlert == nil {
		t.Fatal("expected health-gc-advised alert, got none")
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", gcAlert.Severity, gcAlert.Summary, gcAlert.Detail)

	if gcAlert.Severity != engine.SeverityLow {
		t.Errorf("severity = %v, want Low", gcAlert.Severity)
	}

	if len(gcAlert.Suggestions) != 1 {
		t.Fatalf("suggestions = %d, want 1", len(gcAlert.Suggestions))
	}

	sug := gcAlert.Suggestions[0]
	if sug.ActionName != "run-gc" {
		t.Errorf("suggestion ActionName = %q, want run-gc", sug.ActionName)
	}

	if sug.SubjectKind != engine.SubjectRepo {
		t.Errorf("suggestion SubjectKind = %v, want SubjectRepo", sug.SubjectKind)
	}

	// Verify the action exists in the registry and matches SubjectKind.
	action, ok := e.Actions.Get("run-gc")
	if !ok {
		t.Fatal("run-gc action not found in registry")
	}

	if action.ApplyTo() != engine.SubjectRepo {
		t.Errorf("run-gc ApplyTo = %v, want SubjectRepo", action.ApplyTo())
	}

	t.Logf("action: %s — destructive=%v — %s", action.Name(), action.Destructive(), action.Description())
}

func TestFullSlice_HealthClean(t *testing.T) {
	e := NewEngine()

	// Healthy repo — no GC needed, has origin remote.
	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Kind: git.KindClone,
		Remotes: []git.Remote{
			{Name: git.RemoteOrigin, FetchURL: "https://github.com/test/repo"},
		},
		Health: &git.HealthReport{
			OK:           true,
			GCAdvised:    false,
			LooseObjects: 10,
		},
		Size: &git.RepoSize{
			GitDirBytes:    1024 * 1024,
			ReachableBytes: 900 * 1024,
			RepackAdvised:  false,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	// All alerts should be SeverityNone (checks ran, nothing wrong).
	for _, a := range alerts {
		if a.Severity != engine.SeverityNone {
			t.Errorf("check %q raised severity %v on a clean repo", a.CheckName, a.Severity)
		}
	}

	t.Logf("clean repo: %d checks ran, all SeverityNone", len(alerts))
}

func TestFullSlice_FSCKErrors(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/corrupt-repo",
		Health: &git.HealthReport{
			OK:         false,
			FSCKErrors: []string{"missing blob abc1234", "broken link to tree def5678"},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var fsckAlert *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "health-fsck-errors" && alerts[i].Severity != engine.SeverityNone {
			fsckAlert = &alerts[i]

			break
		}
	}

	if fsckAlert == nil {
		t.Fatal("expected health-fsck-errors alert")
	}

	if fsckAlert.Severity != engine.SeverityCritical {
		t.Errorf("severity = %v, want Critical", fsckAlert.Severity)
	}

	// FSCK alerts should have no suggestions (manual fix needed).
	if len(fsckAlert.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0 (manual fix)", len(fsckAlert.Suggestions))
	}

	t.Logf("fsck alert: %q detail=%q", fsckAlert.Summary, fsckAlert.Detail)
}

func TestFullSlice_BranchLagging(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main", Behind: 0},
			{Name: "feature-a", Upstream: "origin/feature-a", Behind: 3},
			{Name: "feature-b", Upstream: "origin/feature-b", Behind: 1},
			{Name: "origin/main", IsRemote: true, Behind: 5}, // remote branch, should be skipped
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "branch-lagging" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected branch-lagging alert")
	}

	if found.Severity != engine.SeverityLow {
		t.Errorf("severity = %v, want Low", found.Severity)
	}

	if len(found.Suggestions) != 1 {
		t.Fatalf("suggestions = %d, want 1", len(found.Suggestions))
	}

	sug := found.Suggestions[0]
	if sug.ActionName != "update-branch" {
		t.Errorf("ActionName = %q, want update-branch", sug.ActionName)
	}

	if sug.SubjectKind != engine.SubjectBranch {
		t.Errorf("SubjectKind = %v, want SubjectBranch", sug.SubjectKind)
	}

	if len(sug.Subjects) != 2 {
		t.Errorf("Subjects = %v, want 2 branches", sug.Subjects)
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_BranchLagging_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main", Behind: 0},
			{Name: "feature", Upstream: "origin/feature", Behind: 0},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "branch-lagging" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_BranchMergedNotDeleted(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:          "/tmp/test-repo",
		DefaultBranch: "main",
		Branches: []git.Branch{
			{Name: "main", Merged: true},
			{Name: "old-feature", Merged: true},
			{Name: "another-merged", Merged: true},
			{Name: "in-progress", Merged: false},
			{Name: "origin/old-feature", IsRemote: true, Merged: true}, // remote, should be skipped
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "branch-merged-not-deleted" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected branch-merged-not-deleted alert")
	}

	if found.Severity != engine.SeverityMedium {
		t.Errorf("severity = %v, want Medium", found.Severity)
	}

	if len(found.Suggestions) != 1 {
		t.Fatalf("suggestions = %d, want 1", len(found.Suggestions))
	}

	sug := found.Suggestions[0]
	if sug.ActionName != "delete-branch" {
		t.Errorf("ActionName = %q, want delete-branch", sug.ActionName)
	}

	if sug.SubjectKind != engine.SubjectBranch {
		t.Errorf("SubjectKind = %v, want SubjectBranch", sug.SubjectKind)
	}

	// Should be old-feature and another-merged (not "main", not remote)
	if len(sug.Subjects) != 2 {
		t.Errorf("Subjects = %v, want 2 branches", sug.Subjects)
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_BranchMergedNotDeleted_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:          "/tmp/clean-repo",
		DefaultBranch: "main",
		Branches: []git.Branch{
			{Name: "main", Merged: true},
			{Name: "active-feature", Merged: false},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "branch-merged-not-deleted" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_BranchGoneUpstream(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main"},
			{Name: "old-pr", Gone: true},
			{Name: "origin/main", IsRemote: true, Gone: true}, // remote, should be skipped
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "branch-gone-upstream" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected branch-gone-upstream alert")
	}

	if found.Severity != engine.SeverityMedium {
		t.Errorf("severity = %v, want Medium", found.Severity)
	}

	if len(found.Suggestions) != 1 {
		t.Fatalf("suggestions = %d, want 1", len(found.Suggestions))
	}

	sug := found.Suggestions[0]
	if sug.ActionName != "delete-branch" {
		t.Errorf("ActionName = %q, want delete-branch", sug.ActionName)
	}

	if len(sug.Subjects) != 1 || sug.Subjects[0] != "old-pr" {
		t.Errorf("Subjects = %v, want [old-pr]", sug.Subjects)
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_BranchGoneUpstream_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main"},
			{Name: "feature", Upstream: "origin/feature"},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "branch-gone-upstream" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_BranchNoUpstream(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main"},
			{Name: "local-only", Upstream: ""},
			{Name: "current-wip", Upstream: "", IsCurrent: true}, // current branch, should be skipped
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "branch-no-upstream" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected branch-no-upstream alert")
	}

	if found.Severity != engine.SeverityLow {
		t.Errorf("severity = %v, want Low", found.Severity)
	}

	if len(found.Suggestions) != 1 {
		t.Fatalf("suggestions = %d, want 1", len(found.Suggestions))
	}

	sug := found.Suggestions[0]
	if sug.ActionName != "push-branch" {
		t.Errorf("ActionName = %q, want push-branch", sug.ActionName)
	}

	if sug.SubjectKind != engine.SubjectBranch {
		t.Errorf("SubjectKind = %v, want SubjectBranch", sug.SubjectKind)
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_BranchNoUpstream_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main"},
			{Name: "feature", Upstream: "origin/feature"},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "branch-no-upstream" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_BranchDiverged(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main", Ahead: 0, Behind: 0},
			{Name: "diverged-rebasable", Upstream: "origin/diverged-rebasable", Ahead: 2, Behind: 3,
				RebaseCheck: &git.RebaseCheck{CanRebase: true}},
			{Name: "diverged-no-check", Upstream: "origin/diverged-no-check", Ahead: 1, Behind: 1},
			{Name: "only-ahead", Upstream: "origin/only-ahead", Ahead: 5, Behind: 0},
			{Name: "origin/diverged-rebasable", IsRemote: true, Ahead: 1, Behind: 1}, // remote, skip
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "branch-diverged" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected branch-diverged alert")
	}

	if found.Severity != engine.SeverityMedium {
		t.Errorf("severity = %v, want Medium", found.Severity)
	}

	// 2 diverged branches, but only 1 is rebasable — suggestion should have 1 subject.
	if len(found.Suggestions) != 1 {
		t.Fatalf("suggestions = %d, want 1", len(found.Suggestions))
	}

	sug := found.Suggestions[0]
	if sug.ActionName != "rebase-branch" {
		t.Errorf("ActionName = %q, want rebase-branch", sug.ActionName)
	}

	if sug.SubjectKind != engine.SubjectBranch {
		t.Errorf("SubjectKind = %v, want SubjectBranch", sug.SubjectKind)
	}

	if len(sug.Subjects) != 1 || sug.Subjects[0] != "diverged-rebasable" {
		t.Errorf("Subjects = %v, want [diverged-rebasable]", sug.Subjects)
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_BranchDiverged_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Branches: []git.Branch{
			{Name: "main", Upstream: "origin/main", Ahead: 0, Behind: 0},
			{Name: "feature", Upstream: "origin/feature", Ahead: 2, Behind: 0},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "branch-diverged" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_DirtyWorktree(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Status: git.Status{
			Entries: []git.StatusEntry{
				{XY: "M.", Path: "staged.go"},
				{XY: ".M", Path: "unstaged.go"},
				{XY: ".M", Path: "unstaged2.go"},
				{XY: "??", Path: "untracked.txt"},
			},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "dirty-worktree" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected dirty-worktree alert")
	}

	if found.Severity != engine.SeverityHigh {
		t.Errorf("severity = %v, want High", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", found.Severity, found.Summary, found.Detail)
}

func TestFullSlice_DirtyWorktree_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:   "/tmp/clean-repo",
		Status: git.Status{},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "dirty-worktree" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_ActivityStale(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:       "/tmp/test-repo",
		LastCommit: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Activity: &git.Activity{
			Commits360d: 5,
			Staleness:   git.StalenessStale,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "activity-stale" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected activity-stale alert")
	}

	if found.Severity != engine.SeverityLow {
		t.Errorf("severity = %v, want Low", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", found.Severity, found.Summary, found.Detail)
}

func TestFullSlice_ActivityStale_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Activity: &git.Activity{
			Commits30d: 10,
			Staleness:  git.StalenessActive,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "activity-stale" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_ActivityDormant(t *testing.T) {
	// Dormant repos should show "repository is stale" via the unified activity-stale check.
	e := NewEngine()

	info := &git.RepoInfo{
		Path:       "/tmp/test-repo",
		LastCommit: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Activity: &git.Activity{
			Staleness: git.StalenessDormant,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "activity-stale" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected activity-stale alert for dormant repo")
	}

	if found.Severity != engine.SeverityLow {
		t.Errorf("severity = %v, want Low", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", found.Severity, found.Summary, found.Detail)
}

func TestFullSlice_ActivityDormant_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Activity: &git.Activity{
			Commits30d: 5,
			Commits90d: 10,
			Staleness:  git.StalenessActive,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "activity-stale" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on active repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_ActivityLowActivity(t *testing.T) {
	// 30d inactive but 90d has commits → "low activity"
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Activity: &git.Activity{
			Commits30d:  0,
			Commits90d:  5,
			Commits360d: 60,
			Staleness:   git.StalenessRecent,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "activity-stale" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected activity-stale alert for low activity repo")
	}

	if found.Summary != "low activity" {
		t.Errorf("Summary = %q, want 'low activity'", found.Summary)
	}

	t.Logf("alert: %s — %s", found.Summary, found.Detail)
}

func TestFullSlice_ConfigNoEmail(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Config: &git.RepoConfig{
			UserEmail: git.ConfigEntry{Key: "user.email", Value: ""},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "config-no-email" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected config-no-email alert")
	}

	if found.Severity != engine.SeverityMedium {
		t.Errorf("severity = %v, want Medium", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", found.Severity, found.Summary, found.Detail)
}

func TestFullSlice_ConfigNoEmail_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Config: &git.RepoConfig{
			UserEmail: git.ConfigEntry{Key: "user.email", Value: "user@example.com", Scope: git.ScopeGlobal},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "config-no-email" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_ConfigUnsigned(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		Config: &git.RepoConfig{
			CommitSign: git.ConfigEntry{Key: "commit.gpgsign", Value: "false", Scope: git.ScopeGlobal},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "config-unsigned" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected config-unsigned alert")
	}

	if found.Severity != engine.SeverityInfo {
		t.Errorf("severity = %v, want Info", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", found.Severity, found.Summary, found.Detail)
}

func TestFullSlice_ConfigUnsigned_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		Config: &git.RepoConfig{
			CommitSign: git.ConfigEntry{Key: "commit.gpgsign", Value: "true", Scope: git.ScopeLocal},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "config-unsigned" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_LargeFiles(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		FileStats: &git.FileStats{
			LargeFiles: []git.FileEntry{
				{Path: "big.bin", Size: 10 * 1024 * 1024},
				{Path: "data.csv", Size: 2 * 1024 * 1024},
			},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "filestats-large-files" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected filestats-large-files alert")
	}

	if found.Severity != engine.SeverityLow {
		t.Errorf("severity = %v, want Low", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", found.Severity, found.Summary, found.Detail)
}

func TestFullSlice_LargeFiles_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		FileStats: &git.FileStats{
			LargeFiles: nil,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "filestats-large-files" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_BinaryFiles(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/test-repo",
		FileStats: &git.FileStats{
			BinaryFiles: []string{"image.png", "archive.zip", "font.woff"},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "filestats-binary" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected filestats-binary alert")
	}

	if found.Severity != engine.SeverityInfo {
		t.Errorf("severity = %v, want Info", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q detail=%q", found.Severity, found.Summary, found.Detail)
}

func TestFullSlice_BinaryFiles_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
		FileStats: &git.FileStats{
			BinaryFiles: nil,
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "filestats-binary" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_Shallow(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:      "/tmp/test-repo",
		IsShallow: true,
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "traits-shallow" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected traits-shallow alert")
	}

	if found.Severity != engine.SeverityInfo {
		t.Errorf("severity = %v, want Info", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_Shallow_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:      "/tmp/clean-repo",
		IsShallow: false,
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "traits-shallow" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_Submodules(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:          "/tmp/test-repo",
		HasSubmodules: true,
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "traits-submodules" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected traits-submodules alert")
	}

	if found.Severity != engine.SeverityInfo {
		t.Errorf("severity = %v, want Info", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_Submodules_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:          "/tmp/clean-repo",
		HasSubmodules: false,
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "traits-submodules" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

func TestFullSlice_LFS(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:   "/tmp/test-repo",
		HasLFS: true,
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	var found *engine.Alert

	for i := range alerts {
		if alerts[i].CheckName == "traits-lfs" && alerts[i].Severity != engine.SeverityNone {
			found = &alerts[i]

			break
		}
	}

	if found == nil {
		t.Fatal("expected traits-lfs alert")
	}

	if found.Severity != engine.SeverityInfo {
		t.Errorf("severity = %v, want Info", found.Severity)
	}

	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0", len(found.Suggestions))
	}

	t.Logf("alert: severity=%s summary=%q", found.Severity, found.Summary)
}

func TestFullSlice_LFS_Clean(t *testing.T) {
	e := NewEngine()

	info := &git.RepoInfo{
		Path:   "/tmp/clean-repo",
		HasLFS: false,
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	for _, a := range alerts {
		if a.CheckName == "traits-lfs" && a.Severity != engine.SeverityNone {
			t.Errorf("check %q fired on clean repo with severity %v", a.CheckName, a.Severity)
		}
	}
}

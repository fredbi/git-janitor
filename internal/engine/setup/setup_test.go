package setup

import (
	"context"
	"testing"

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

	// Healthy repo — no GC needed.
	info := &git.RepoInfo{
		Path: "/tmp/clean-repo",
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

	if fsckAlert.Severity != engine.SeverityHigh {
		t.Errorf("severity = %v, want High", fsckAlert.Severity)
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

	// No suggestions (informational only)
	if len(found.Suggestions) != 0 {
		t.Errorf("suggestions = %d, want 0 (informational)", len(found.Suggestions))
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
			{Name: "diverged-branch", Upstream: "origin/diverged-branch", Ahead: 2, Behind: 3},
			{Name: "only-ahead", Upstream: "origin/only-ahead", Ahead: 5, Behind: 0},
			{Name: "origin/diverged-branch", IsRemote: true, Ahead: 1, Behind: 1}, // remote, skip
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

	if len(sug.Subjects) != 1 || sug.Subjects[0] != "diverged-branch" {
		t.Errorf("Subjects = %v, want [diverged-branch]", sug.Subjects)
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

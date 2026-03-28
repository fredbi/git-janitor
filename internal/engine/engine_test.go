package engine

import (
	"context"
	"iter"
	"testing"

	"github.com/fredbi/git-janitor/internal/git"
)

// --- Test check and action implementations ---

type testBranchCheck struct {
	GitCheck
}

func (c testBranchCheck) Evaluate(info *git.RepoInfo) (iter.Seq[Alert], error) {
	var lagging []string

	for _, b := range info.Branches {
		if !b.IsRemote && b.Behind > 0 {
			lagging = append(lagging, b.Name)
		}
	}

	if len(lagging) == 0 {
		return singleAlert(Alert{
			CheckName: c.Name(),
			Severity:  SeverityNone,
		}), nil
	}

	return singleAlert(Alert{
		CheckName: c.Name(),
		Severity:  SeverityLow,
		Summary:   "branches lagging behind remote",
		Suggestions: []ActionSuggestion{{
			ActionName:  "update-branch",
			SubjectKind: SubjectBranch,
			Subjects:    lagging,
		}},
	}), nil
}

type testRepoCheck struct {
	GitCheck
}

func (c testRepoCheck) Evaluate(info *git.RepoInfo) (iter.Seq[Alert], error) {
	if info.Status.IsDirty() {
		return singleAlert(Alert{
			CheckName: c.Name(),
			Severity:  SeverityHigh,
			Summary:   "worktree has uncommitted changes",
		}), nil
	}

	return singleAlert(Alert{
		CheckName: c.Name(),
		Severity:  SeverityNone,
	}), nil
}

type testUpdateAction struct {
	GitAction
}

func (testUpdateAction) ApplyTo() SubjectKind { return SubjectBranch }

func (a testUpdateAction) Execute(_ context.Context, _ *git.Runner, _ *git.RepoInfo, subjects []string) (Result, error) {
	return Result{OK: true, Message: "updated " + subjects[0]}, nil
}

func singleAlert(a Alert) iter.Seq[Alert] {
	return func(yield func(Alert) bool) {
		yield(a)
	}
}

// --- Registry tests ---

func TestCheckRegistry(t *testing.T) {
	r := NewCheckRegistry()

	check := testBranchCheck{GitCheck: GitCheck{
		Describer: Describer{CheckName: "branch-lagging", CheckDescription: "detects lagging branches"},
	}}

	r.Register(check)

	if r.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", r.Len())
	}

	got, ok := r.Get("branch-lagging")
	if !ok {
		t.Fatal("Get() returned false")
	}

	if got.Name() != "branch-lagging" {
		t.Errorf("Name() = %q", got.Name())
	}

	if got.Description() != "detects lagging branches" {
		t.Errorf("Description() = %q", got.Description())
	}

	if got.Kind() != CheckKindGit {
		t.Errorf("Kind() = %v, want CheckKindGit", got.Kind())
	}
}

func TestCheckRegistry_DuplicatePanics(t *testing.T) {
	r := NewCheckRegistry()

	check := testBranchCheck{GitCheck: GitCheck{
		Describer: Describer{CheckName: "dup"},
	}}

	r.Register(check)

	defer func() {
		if recover() == nil {
			t.Error("expected panic on duplicate register")
		}
	}()

	r.Register(check) // should panic
}

func TestCheckRegistry_All(t *testing.T) {
	r := NewCheckRegistry()

	r.Register(testBranchCheck{GitCheck: GitCheck{Describer: Describer{CheckName: "a"}}})
	r.Register(testRepoCheck{GitCheck: GitCheck{Describer: Describer{CheckName: "b"}}})
	r.Register(testBranchCheck{GitCheck: GitCheck{Describer: Describer{CheckName: "c"}}})

	var names []string

	for name := range r.All() {
		names = append(names, name)
	}

	if len(names) != 3 {
		t.Fatalf("All() yielded %d, want 3", len(names))
	}

	// Should be in insertion order.
	if names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Errorf("order = %v, want [a b c]", names)
	}
}

func TestActionRegistry(t *testing.T) {
	r := NewActionRegistry()

	action := testUpdateAction{GitAction: GitAction{
		Describer: Describer{CheckName: "update-branch", CheckDescription: "fast-forward a branch"},
	}}

	r.Register(action)

	got, ok := r.Get("update-branch")
	if !ok {
		t.Fatal("Get() returned false")
	}

	if got.ApplyTo() != SubjectBranch {
		t.Errorf("ApplyTo() = %v, want SubjectBranch", got.ApplyTo())
	}

	if got.Destructive() {
		t.Error("Destructive() should be false")
	}
}

// --- Engine dispatch tests ---

func TestEngine_EvaluateRepo(t *testing.T) {
	e := New()

	e.Checks.Register(testBranchCheck{GitCheck: GitCheck{
		Describer: Describer{CheckName: "branch-lagging", CheckDescription: "lagging"},
	}})
	e.Checks.Register(testRepoCheck{GitCheck: GitCheck{
		Describer: Describer{CheckName: "dirty-worktree", CheckDescription: "dirty"},
	}})

	info := &git.RepoInfo{
		Branches: []git.Branch{
			{Name: "feature/old", Behind: 3},
			{Name: "main", Behind: 0},
		},
		Status: git.Status{
			Entries: []git.StatusEntry{{XY: "M.", Path: "file.go"}},
		},
	}

	alerts := e.EvaluateRepo(context.Background(), info, nil)

	// Expect 2 alerts: dirty-worktree (high) and branch-lagging (low).
	if len(alerts) != 2 {
		t.Fatalf("got %d alerts, want 2", len(alerts))
	}

	// Should be sorted by severity descending.
	if alerts[0].Severity != SeverityHigh {
		t.Errorf("alerts[0].Severity = %v, want High", alerts[0].Severity)
	}

	if alerts[0].CheckName != "dirty-worktree" {
		t.Errorf("alerts[0].CheckName = %q, want dirty-worktree", alerts[0].CheckName)
	}

	if alerts[1].Severity != SeverityLow {
		t.Errorf("alerts[1].Severity = %v, want Low", alerts[1].Severity)
	}

	// Branch check should suggest update-branch with 1 subject.
	if len(alerts[1].Suggestions) != 1 {
		t.Fatalf("alerts[1] has %d suggestions, want 1", len(alerts[1].Suggestions))
	}

	sug := alerts[1].Suggestions[0]
	if sug.ActionName != "update-branch" {
		t.Errorf("suggestion ActionName = %q", sug.ActionName)
	}

	if sug.SubjectKind != SubjectBranch {
		t.Errorf("suggestion SubjectKind = %v", sug.SubjectKind)
	}

	if len(sug.Subjects) != 1 || sug.Subjects[0] != "feature/old" {
		t.Errorf("suggestion Subjects = %v", sug.Subjects)
	}
}

func TestEngine_EvaluateRepo_FilterByEnabled(t *testing.T) {
	e := New()

	e.Checks.Register(testBranchCheck{GitCheck: GitCheck{
		Describer: Describer{CheckName: "branch-lagging"},
	}})
	e.Checks.Register(testRepoCheck{GitCheck: GitCheck{
		Describer: Describer{CheckName: "dirty-worktree"},
	}})

	info := &git.RepoInfo{
		Branches: []git.Branch{{Name: "old", Behind: 5}},
		Status:   git.Status{Entries: []git.StatusEntry{{XY: "M.", Path: "f.go"}}},
	}

	// Only enable branch-lagging.
	alerts := e.EvaluateRepo(context.Background(), info, []string{"branch-lagging"})

	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}

	if alerts[0].CheckName != "branch-lagging" {
		t.Errorf("CheckName = %q, want branch-lagging", alerts[0].CheckName)
	}
}

func TestEngine_Execute_SubjectKindValidation(t *testing.T) {
	e := New()

	e.Actions.Register(testUpdateAction{GitAction: GitAction{
		Describer: Describer{CheckName: "update-branch"},
	}})

	// Mismatched SubjectKind.
	_, err := e.Execute(context.Background(), nil, nil, ActionSuggestion{
		ActionName:  "update-branch",
		SubjectKind: SubjectTag, // wrong — action applies to Branch
		Subjects:    []string{"v1.0.0"},
	})

	if err == nil {
		t.Error("expected error for SubjectKind mismatch")
	}
}

func TestEngine_Execute_Success(t *testing.T) {
	e := New()

	e.Actions.Register(testUpdateAction{GitAction: GitAction{
		Describer: Describer{CheckName: "update-branch"},
	}})

	result, err := e.Execute(context.Background(), nil, nil, ActionSuggestion{
		ActionName:  "update-branch",
		SubjectKind: SubjectBranch,
		Subjects:    []string{"feature/foo"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.OK {
		t.Errorf("result.OK = false, want true")
	}

	if result.Message != "updated feature/foo" {
		t.Errorf("result.Message = %q", result.Message)
	}
}

// --- History tests ---

func TestHistory(t *testing.T) {
	h := NewHistory(3)

	h.Append(HistoryEntry{ActionName: "a"})
	h.Append(HistoryEntry{ActionName: "b"})
	h.Append(HistoryEntry{ActionName: "c"})

	entries := h.Entries()
	if len(entries) != 3 {
		t.Fatalf("Len() = %d, want 3", len(entries))
	}

	// Newest first.
	if entries[0].ActionName != "c" || entries[2].ActionName != "a" {
		t.Errorf("order: %s %s %s", entries[0].ActionName, entries[1].ActionName, entries[2].ActionName)
	}

	// Overflow: oldest evicted.
	h.Append(HistoryEntry{ActionName: "d"})

	if h.Len() != 3 {
		t.Fatalf("Len() after overflow = %d, want 3", h.Len())
	}

	entries = h.Entries()
	if entries[0].ActionName != "d" || entries[2].ActionName != "b" {
		t.Errorf("after overflow: %s %s %s", entries[0].ActionName, entries[1].ActionName, entries[2].ActionName)
	}
}

// --- Severity/SubjectKind string tests ---

func TestSeverityString(t *testing.T) {
	tests := []struct {
		s    Severity
		want string
	}{
		{SeverityNone, "none"},
		{SeverityInfo, "info"},
		{SeverityLow, "low"},
		{SeverityMedium, "medium"},
		{SeverityHigh, "high"},
	}

	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestSubjectKindString(t *testing.T) {
	if SubjectBranch.String() != "branch" {
		t.Errorf("SubjectBranch.String() = %q", SubjectBranch.String())
	}

	if SubjectTag.String() != "tag" {
		t.Errorf("SubjectTag.String() = %q", SubjectTag.String())
	}
}

package engine

import (
	"context"
	"iter"
	"testing"

	"github.com/fredbi/git-janitor/internal/git"
)

// --- Test check and action implementations ---

type testBranchmodels.Check struct {
	GitCheck
}

func (c testBranchmodels.Check) Evaluate(info *git.RepoInfo) (iter.Seq[Alert], error) {
	var lagging []string

	for _, b := range info.Branches {
		if !b.IsRemote && b.Behind > 0 {
			lagging = append(lagging, b.Name)
		}
	}

	if len(lagging) == 0 {
		return singleAlert(Alert{
			models.CheckName: c.Name(),
			Severity:  SeverityNone,
		}), nil
	}

	return singleAlert(Alert{
		models.CheckName: c.Name(),
		Severity:  SeverityLow,
		Summary:   "branches lagging behind remote",
		Suggestions: []models.ActionSuggestion{{
			models.ActionName:  "update-branch",
			models.SubjectKind: SubjectBranch,
			Subjects:    lagging,
		}},
	}), nil
}

type testRepomodels.Check struct {
	GitCheck
}

func (c testRepomodels.Check) Evaluate(info *git.RepoInfo) (iter.Seq[Alert], error) {
	if info.Status.IsDirty() {
		return singleAlert(Alert{
			models.CheckName: c.Name(),
			Severity:  SeverityHigh,
			Summary:   "worktree has uncommitted changes",
		}), nil
	}

	return singleAlert(Alert{
		models.CheckName: c.Name(),
		Severity:  SeverityNone,
	}), nil
}

type testUpdatemodels.Action struct {
	GitAction
}

func (testUpdatemodels.Action) ApplyTo() models.SubjectKind { return SubjectBranch }

func (a testUpdatemodels.Action) Execute(_ context.Context, _ *git.Runner, _ *git.RepoInfo, subjects []string) (models.Result, error) {
	return models.Result{OK: true, Message: "updated " + subjects[0]}, nil
}

func singleAlert(a Alert) iter.Seq[Alert] {
	return func(yield func(Alert) bool) {
		yield(a)
	}
}

// --- Registry tests ---

func Testmodels.CheckRegistry(t *testing.T) {
	r := Newmodels.CheckRegistry()

	check := testBranchmodels.Check{GitCheck: GitCheck{
		Describer: Describer{models.CheckName: "branch-lagging", models.CheckDescription: "detects lagging branches"},
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

	if got.Kind() != models.CheckKindGit {
		t.Errorf("Kind() = %v, want models.CheckKindGit", got.Kind())
	}
}

func Testmodels.CheckRegistry_DuplicatePanics(t *testing.T) {
	r := Newmodels.CheckRegistry()

	check := testBranchmodels.Check{GitCheck: GitCheck{
		Describer: Describer{models.CheckName: "dup"},
	}}

	r.Register(check)

	defer func() {
		if recover() == nil {
			t.Error("expected panic on duplicate register")
		}
	}()

	r.Register(check) // should panic
}

func Testmodels.CheckRegistry_All(t *testing.T) {
	r := Newmodels.CheckRegistry()

	r.Register(testBranchmodels.Check{GitCheck: GitCheck{Describer: Describer{models.CheckName: "a"}}})
	r.Register(testRepomodels.Check{GitCheck: GitCheck{Describer: Describer{models.CheckName: "b"}}})
	r.Register(testBranchmodels.Check{GitCheck: GitCheck{Describer: Describer{models.CheckName: "c"}}})

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

func Testmodels.ActionRegistry(t *testing.T) {
	r := Newmodels.ActionRegistry()

	action := testUpdatemodels.Action{GitAction: GitAction{
		Describer: Describer{models.CheckName: "update-branch", models.CheckDescription: "fast-forward a branch"},
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

	e.models.Checks.Register(testBranchmodels.Check{GitCheck: GitCheck{
		Describer: Describer{models.CheckName: "branch-lagging", models.CheckDescription: "lagging"},
	}})
	e.models.Checks.Register(testRepomodels.Check{GitCheck: GitCheck{
		Describer: Describer{models.CheckName: "dirty-worktree", models.CheckDescription: "dirty"},
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

	if alerts[0].models.CheckName != "dirty-worktree" {
		t.Errorf("alerts[0].models.CheckName = %q, want dirty-worktree", alerts[0].models.CheckName)
	}

	if alerts[1].Severity != SeverityLow {
		t.Errorf("alerts[1].Severity = %v, want Low", alerts[1].Severity)
	}

	// Branch check should suggest update-branch with 1 subject.
	if len(alerts[1].Suggestions) != 1 {
		t.Fatalf("alerts[1] has %d suggestions, want 1", len(alerts[1].Suggestions))
	}

	sug := alerts[1].Suggestions[0]
	if sug.models.ActionName != "update-branch" {
		t.Errorf("suggestion models.ActionName = %q", sug.models.ActionName)
	}

	if sug.models.SubjectKind != SubjectBranch {
		t.Errorf("suggestion models.SubjectKind = %v", sug.models.SubjectKind)
	}

	if len(sug.Subjects) != 1 || sug.Subjects[0] != "feature/old" {
		t.Errorf("suggestion Subjects = %v", sug.Subjects)
	}
}

func TestEngine_EvaluateRepo_FilterByEnabled(t *testing.T) {
	e := New()

	e.models.Checks.Register(testBranchmodels.Check{GitCheck: GitCheck{
		Describer: Describer{models.CheckName: "branch-lagging"},
	}})
	e.models.Checks.Register(testRepomodels.Check{GitCheck: GitCheck{
		Describer: Describer{models.CheckName: "dirty-worktree"},
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

	if alerts[0].models.CheckName != "branch-lagging" {
		t.Errorf("models.CheckName = %q, want branch-lagging", alerts[0].models.CheckName)
	}
}

func TestEngine_Execute_models.SubjectKindValidation(t *testing.T) {
	e := New()

	e.models.Actions.Register(testUpdatemodels.Action{GitAction: GitAction{
		Describer: Describer{models.CheckName: "update-branch"},
	}})

	// Mismatched models.SubjectKind.
	_, err := e.Execute(context.Background(), nil, nil, models.ActionSuggestion{
		models.ActionName:  "update-branch",
		models.SubjectKind: SubjectTag, // wrong — action applies to Branch
		Subjects:    []string{"v1.0.0"},
	})

	if err == nil {
		t.Error("expected error for models.SubjectKind mismatch")
	}
}

func TestEngine_Execute_Success(t *testing.T) {
	e := New()

	e.models.Actions.Register(testUpdatemodels.Action{GitAction: GitAction{
		Describer: Describer{models.CheckName: "update-branch"},
	}})

	result, err := e.Execute(context.Background(), nil, nil, models.ActionSuggestion{
		models.ActionName:  "update-branch",
		models.SubjectKind: SubjectBranch,
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

	h.Append(HistoryEntry{models.ActionName: "a"})
	h.Append(HistoryEntry{models.ActionName: "b"})
	h.Append(HistoryEntry{models.ActionName: "c"})

	entries := h.Entries()
	if len(entries) != 3 {
		t.Fatalf("Len() = %d, want 3", len(entries))
	}

	// Newest first.
	if entries[0].models.ActionName != "c" || entries[2].models.ActionName != "a" {
		t.Errorf("order: %s %s %s", entries[0].models.ActionName, entries[1].models.ActionName, entries[2].models.ActionName)
	}

	// Overflow: oldest evicted.
	h.Append(HistoryEntry{models.ActionName: "d"})

	if h.Len() != 3 {
		t.Fatalf("Len() after overflow = %d, want 3", h.Len())
	}

	entries = h.Entries()
	if entries[0].models.ActionName != "d" || entries[2].models.ActionName != "b" {
		t.Errorf("after overflow: %s %s %s", entries[0].models.ActionName, entries[1].models.ActionName, entries[2].models.ActionName)
	}
}

// --- Severity/models.SubjectKind string tests ---

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

func Testmodels.SubjectKindString(t *testing.T) {
	if SubjectBranch.String() != "branch" {
		t.Errorf("SubjectBranch.String() = %q", SubjectBranch.String())
	}

	if SubjectTag.String() != "tag" {
		t.Errorf("SubjectTag.String() = %q", SubjectTag.String())
	}
}

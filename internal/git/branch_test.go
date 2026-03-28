package git

import (
	"context"
	"testing"
)

func TestParseBranches(t *testing.T) {
	input := "*|main|abc1234|origin/main|[ahead 1]|2026-03-27T18:30:20+01:00\n" +
		" |feature/foo|def5678|origin/feature/foo|[behind 2]|2026-01-15T10:00:00+00:00\n" +
		" |origin/main|abc1234|||2026-03-27T18:30:20+01:00\n" +
		" |origin/feature/foo|def5678|||2026-01-15T10:00:00+00:00\n" +
		" |origin/HEAD|abc1234|||\n"

	branches := parseBranches(input)

	// origin/HEAD should be filtered out.
	for _, b := range branches {
		if b.Name == "origin/HEAD" {
			t.Error("origin/HEAD should be filtered out")
		}
	}

	// Check current branch.
	var found bool
	for _, b := range branches {
		if b.Name == "main" && b.IsCurrent {
			found = true

			break
		}
	}

	if !found {
		t.Error("expected main to be the current branch")
	}
}

func TestParseBranches_LocalVsRemote(t *testing.T) {
	input := " |main|abc1234|origin/main||\n" +
		" |origin/main|abc1234|||\n"

	branches := parseBranches(input)

	if len(branches) != 2 {
		t.Fatalf("got %d branches, want 2", len(branches))
	}

	// "main" is local (no slash in the usual case).
	if branches[0].IsRemote {
		t.Error("main should not be remote")
	}

	// "origin/main" is remote.
	if !branches[1].IsRemote {
		t.Error("origin/main should be remote")
	}
}

func TestParseBranches_Empty(t *testing.T) {
	branches := parseBranches("")

	if len(branches) != 0 {
		t.Errorf("got %d branches, want 0", len(branches))
	}
}

func TestParseBranches_WithUpstream(t *testing.T) {
	input := "*|develop|abc1234|origin/develop|[ahead 3, behind 2]|2026-02-10T12:00:00+00:00\n"

	branches := parseBranches(input)

	if len(branches) != 1 {
		t.Fatalf("got %d branches, want 1", len(branches))
	}

	if branches[0].Upstream != "origin/develop" {
		t.Errorf("Upstream = %q, want %q", branches[0].Upstream, "origin/develop")
	}

	if branches[0].Hash != "abc1234" {
		t.Errorf("Hash = %q, want %q", branches[0].Hash, "abc1234")
	}

	if branches[0].Ahead != 3 {
		t.Errorf("Ahead = %d, want 3", branches[0].Ahead)
	}

	if branches[0].Behind != 2 {
		t.Errorf("Behind = %d, want 2", branches[0].Behind)
	}

	if branches[0].LastCommit.IsZero() {
		t.Error("expected LastCommit to be set")
	}

	if branches[0].LastCommit.Year() != 2026 {
		t.Errorf("LastCommit year = %d, want 2026", branches[0].LastCommit.Year())
	}
}

func TestParseBranches_LastCommit(t *testing.T) {
	input := " |recent|abc1234||[ahead 1]|2026-03-27T18:30:20+01:00\n" +
		" |old|def5678|||2024-06-15T08:00:00+00:00\n" +
		" |nodate|fff0000|||\n"

	branches := parseBranches(input)

	if len(branches) != 3 {
		t.Fatalf("got %d branches, want 3", len(branches))
	}

	if branches[0].LastCommit.Year() != 2026 || branches[0].LastCommit.Month() != 3 {
		t.Errorf("recent: LastCommit = %v, want 2026-03", branches[0].LastCommit)
	}

	if branches[1].LastCommit.Year() != 2024 || branches[1].LastCommit.Month() != 6 {
		t.Errorf("old: LastCommit = %v, want 2024-06", branches[1].LastCommit)
	}

	if !branches[2].LastCommit.IsZero() {
		t.Errorf("nodate: expected zero LastCommit, got %v", branches[2].LastCommit)
	}
}

func TestParseBranches_AheadBehind(t *testing.T) {
	input := "*|main|abc1234|origin/main|[ahead 1]|\n" +
		" |fix|def5678|origin/fix|[behind 4]|\n" +
		" |dev|111aaaa|origin/dev|[ahead 2, behind 3]|\n" +
		" |clean|222bbbb|origin/clean||\n"

	branches := parseBranches(input)

	if len(branches) != 4 {
		t.Fatalf("got %d branches, want 4", len(branches))
	}

	tests := []struct {
		name   string
		ahead  int
		behind int
	}{
		{"main", 1, 0},
		{"fix", 0, 4},
		{"dev", 2, 3},
		{"clean", 0, 0},
	}

	for i, tt := range tests {
		b := branches[i]
		if b.Ahead != tt.ahead || b.Behind != tt.behind {
			t.Errorf("%s: ahead/behind = %d/%d, want %d/%d", tt.name, b.Ahead, b.Behind, tt.ahead, tt.behind)
		}
	}
}

func TestParseBranches_Gone(t *testing.T) {
	input := " |stale|abc1234|origin/stale|[gone]|\n" +
		" |active|def5678|origin/active||\n" +
		" |local|fff0000|||\n"

	branches := parseBranches(input)

	if len(branches) != 3 {
		t.Fatalf("got %d branches, want 3", len(branches))
	}

	if !branches[0].Gone {
		t.Error("stale should have Gone=true")
	}

	if branches[1].Gone {
		t.Error("active should have Gone=false")
	}

	if branches[2].Gone {
		t.Error("local should have Gone=false")
	}
}

func TestParseBranches_HasUpstream(t *testing.T) {
	input := " |tracked|abc1234|origin/tracked||\n" +
		" |untracked|def5678|||\n"

	branches := parseBranches(input)

	if len(branches) != 2 {
		t.Fatalf("got %d branches, want 2", len(branches))
	}

	if !branches[0].HasUpstream() {
		t.Error("tracked should have upstream")
	}

	if branches[1].HasUpstream() {
		t.Error("untracked should not have upstream")
	}
}

func TestMarkMerged(t *testing.T) {
	branches := []Branch{
		{Name: "main"},
		{Name: "feature/done"},
		{Name: "feature/wip"},
	}

	merged := map[string]bool{
		"main":         true,
		"feature/done": true,
	}

	// nil runner: only reachability check, no cherry fallback.
	MarkMerged(context.Background(), nil, branches, "main", merged)

	if !branches[0].Merged {
		t.Error("main should be marked as merged")
	}

	if !branches[1].Merged {
		t.Error("feature/done should be marked as merged")
	}

	if branches[2].Merged {
		t.Error("feature/wip should not be marked as merged")
	}
}

func TestParseMergeTreeConflicts(t *testing.T) {
	// Clean merge: just a tree hash.
	clean := parseMergeTreeConflicts("f4d8637ae42ace85af04e1c288fe3bf87731ac2c\n")
	// This function is called on error path, so a hash-only output
	// with no extra lines means no files could be parsed.
	if len(clean.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", clean.Conflicts)
	}

	// Conflict: tree hash + file names.
	conflicted := parseMergeTreeConflicts(
		"f4d8637ae42ace85af04e1c288fe3bf87731ac2c\n\nREADME.md\nsrc/main.go\n",
	)
	if conflicted.Clean {
		t.Error("expected Clean=false")
	}

	if len(conflicted.Conflicts) != 2 {
		t.Fatalf("got %d conflicts, want 2: %v", len(conflicted.Conflicts), conflicted.Conflicts)
	}

	if conflicted.Conflicts[0] != "README.md" {
		t.Errorf("conflict[0] = %q, want README.md", conflicted.Conflicts[0])
	}

	if conflicted.Conflicts[1] != "src/main.go" {
		t.Errorf("conflict[1] = %q, want src/main.go", conflicted.Conflicts[1])
	}
}

func TestIsHexHash(t *testing.T) {
	if !isHexHash("f4d8637ae42ace85af04e1c288fe3bf87731ac2c") {
		t.Error("expected true for 40-char hex")
	}

	if isHexHash("README.md") {
		t.Error("expected false for filename")
	}

	if isHexHash("not-a-hash-at-all-but-is-40-chars-long!") {
		t.Error("expected false for non-hex 40-char string")
	}
}

func TestParseUpstreamTrack(t *testing.T) {
	tests := []struct {
		input  string
		ahead  int
		behind int
		gone   bool
	}{
		{"", 0, 0, false},
		{"[ahead 3]", 3, 0, false},
		{"[behind 5]", 0, 5, false},
		{"[ahead 1, behind 2]", 1, 2, false},
		{"[gone]", 0, 0, true},
	}

	for _, tt := range tests {
		ahead, behind, gone := parseUpstreamTrack(tt.input)
		if ahead != tt.ahead || behind != tt.behind || gone != tt.gone {
			t.Errorf("parseUpstreamTrack(%q) = (%d, %d, %v), want (%d, %d, %v)",
				tt.input, ahead, behind, gone, tt.ahead, tt.behind, tt.gone)
		}
	}
}

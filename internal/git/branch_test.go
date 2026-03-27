package git

import (
	"testing"
)

func TestParseBranches(t *testing.T) {
	input := "*|main|abc1234|origin/main\n" +
		" |feature/foo|def5678|origin/feature/foo\n" +
		" |origin/main|abc1234|\n" +
		" |origin/feature/foo|def5678|\n" +
		" |origin/HEAD|abc1234|\n"

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
	input := " |main|abc1234|origin/main\n" +
		" |origin/main|abc1234|\n"

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
	input := "*|develop|abc1234|origin/develop\n"

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
}

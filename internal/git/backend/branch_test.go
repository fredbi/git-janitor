package backend

import (
	"context"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

// Test inputs include the 7th field: %(refname) for reliable local/remote detection.
// Format: HEAD|refname:short|objectname:short|upstream:short|upstream:track|creatordate|refname

func TestParseBranches(t *testing.T) {
	input := "*|main|abc1234|origin/main|[ahead 1]|2026-03-27T18:30:20+01:00|refs/heads/main\n" +
		" |feature/foo|def5678|origin/feature/foo|[behind 2]|2026-01-15T10:00:00+00:00|refs/heads/feature/foo\n" +
		" |origin/main|abc1234|||2026-03-27T18:30:20+01:00|refs/remotes/origin/main\n" +
		" |origin/feature/foo|def5678|||2026-01-15T10:00:00+00:00|refs/remotes/origin/feature/foo\n" +
		" |origin/HEAD|abc1234||||refs/remotes/origin/HEAD\n"

	branches := parseBranches(input)

	// origin/HEAD should be filtered out.
	for _, b := range branches {
		if b.Name == "origin/HEAD" {
			t.Error("origin/HEAD should be filtered out")
		}
	}

	// "feature/foo" should be local, not remote (has slash but refs/heads/).
	for _, b := range branches {
		if b.Name == "feature/foo" && b.IsRemote {
			t.Error("feature/foo should be local, not remote")
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
	input := " |main|abc1234|origin/main|||refs/heads/main\n" +
		" |origin/main|abc1234||||refs/remotes/origin/main\n"

	branches := parseBranches(input)

	if len(branches) != 2 {
		t.Fatalf("got %d branches, want 2", len(branches))
	}

	if branches[0].IsRemote {
		t.Error("main should not be remote")
	}

	if !branches[1].IsRemote {
		t.Error("origin/main should be remote")
	}
}

func TestParseBranches_BareRemoteFiltered(t *testing.T) {
	// Bare remote ref roots (e.g. "origin", "upstream") should be filtered out.
	input := " |master|abc1234|origin/master|||refs/heads/master\n" +
		" |origin|abc1234||||refs/remotes/origin\n" +
		" |origin/master|abc1234||||refs/remotes/origin/master\n" +
		" |upstream|def5678||||refs/remotes/upstream\n" +
		" |upstream/master|def5678||||refs/remotes/upstream/master\n"

	branches := parseBranches(input)

	for _, b := range branches {
		if b.Name == "origin" || b.Name == "upstream" {
			t.Errorf("bare remote %q should have been filtered out", b.Name)
		}
	}

	// Should have: master, origin/master, upstream/master
	if len(branches) != 3 {
		t.Errorf("got %d branches, want 3 (master + origin/master + upstream/master)", len(branches))

		for _, b := range branches {
			t.Logf("  %s isRemote=%v", b.Name, b.IsRemote)
		}
	}
}

func TestParseBranches_SlashedLocalBranch(t *testing.T) {
	// Local branches with slashes (feature/foo, chore/lint) must NOT be remote.
	input := " |chore/lint|c1cf011||||refs/heads/chore/lint\n" +
		" |fix/auto-merge|bed738f||||refs/heads/fix/auto-merge\n" +
		" |origin/master|b0916e0||||refs/remotes/origin/master\n"

	branches := parseBranches(input)

	if len(branches) != 3 {
		t.Fatalf("got %d branches, want 3", len(branches))
	}

	for _, b := range branches {
		if b.Name == "chore/lint" && b.IsRemote {
			t.Error("chore/lint should be local, not remote")
		}

		if b.Name == "fix/auto-merge" && b.IsRemote {
			t.Error("fix/auto-merge should be local, not remote")
		}

		if b.Name == "origin/master" && !b.IsRemote {
			t.Error("origin/master should be remote")
		}
	}
}

func TestParseBranches_Empty(t *testing.T) {
	branches := parseBranches("")

	if len(branches) != 0 {
		t.Errorf("got %d branches, want 0", len(branches))
	}
}

func TestParseBranches_WithUpstream(t *testing.T) {
	input := "*|develop|abc1234|origin/develop|[ahead 3, behind 2]|2026-02-10T12:00:00+00:00|refs/heads/develop\n"

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
	input := " |old-branch|abc1234|||2025-06-15T08:30:00+00:00|refs/heads/old-branch\n"

	branches := parseBranches(input)

	if len(branches) != 1 {
		t.Fatalf("got %d branches, want 1", len(branches))
	}

	if branches[0].LastCommit.IsZero() {
		t.Error("expected LastCommit to be set")
	}

	if branches[0].LastCommit.Month() != 6 || branches[0].LastCommit.Day() != 15 {
		t.Errorf("LastCommit = %v", branches[0].LastCommit)
	}
}

func TestParseBranches_AheadBehind(t *testing.T) {
	tests := []struct {
		track  string
		ahead  int
		behind int
		gone   bool
	}{
		{"", 0, 0, false},
		{"[ahead 3]", 3, 0, false},
		{"[behind 2]", 0, 2, false},
		{"[ahead 3, behind 2]", 3, 2, false},
		{"[gone]", 0, 0, true},
	}

	for _, tt := range tests {
		ahead, behind, gone := parseUpstreamTrack(tt.track)
		if ahead != tt.ahead || behind != tt.behind || gone != tt.gone {
			t.Errorf("parseUpstreamTrack(%q) = (%d, %d, %v), want (%d, %d, %v)",
				tt.track, ahead, behind, gone, tt.ahead, tt.behind, tt.gone)
		}
	}
}

func TestParseBranches_Gone(t *testing.T) {
	input := " |stale|abc1234|origin/stale|[gone]||refs/heads/stale\n"

	branches := parseBranches(input)

	if len(branches) != 1 {
		t.Fatalf("got %d branches, want 1", len(branches))
	}

	if !branches[0].Gone {
		t.Error("expected Gone to be true")
	}
}

func TestParseBranches_HasUpstream(t *testing.T) {
	b := models.Branch{Upstream: "origin/main"}
	if !b.HasUpstream() {
		t.Error("expected HasUpstream() = true")
	}

	b2 := models.Branch{}
	if b2.HasUpstream() {
		t.Error("expected HasUpstream() = false")
	}
}

// Integration test against the actual git-janitor repo.
func TestIntegration_Branches(t *testing.T) {
	r := &Runner{Dir: "/home/fred/src/github.com/fredbi/git-janitor"}
	ctx := context.Background()

	branches, err := r.Branches(ctx)
	if err != nil {
		t.Fatalf("Branches() error: %v", err)
	}

	var current string

	for _, b := range branches {
		if b.IsCurrent {
			current = b.Name
			t.Logf("current branch: %s (%s)", b.Name, b.Hash)
		}
	}

	if current == "" {
		t.Error("no current branch found")
	}

	t.Logf("total branches: %d", len(branches))

	// Verify no bare remote names appear as branches.
	for _, b := range branches {
		if !b.IsRemote && (b.Name == "origin" || b.Name == "upstream") {
			t.Errorf("bare remote %q should not appear as a local branch", b.Name)
		}
	}
}

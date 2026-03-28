package git

import (
	"context"
	"testing"
)

func TestParseRevList(t *testing.T) {
	input := "abc1234\ndef5678\n\nghi9012\n"

	commits := parseRevList(input)

	if len(commits) != 3 {
		t.Fatalf("got %d commits, want 3", len(commits))
	}

	if commits[0] != "abc1234" {
		t.Errorf("commits[0] = %q, want abc1234", commits[0])
	}

	if commits[2] != "ghi9012" {
		t.Errorf("commits[2] = %q, want ghi9012", commits[2])
	}
}

func TestParseRevList_Empty(t *testing.T) {
	commits := parseRevList("")

	if len(commits) != 0 {
		t.Errorf("got %d commits, want 0", len(commits))
	}
}

func TestIntegration_CheckRebase(t *testing.T) {
	r := &Runner{Dir: "."}

	// Find the default branch.
	defaultBranch, err := r.DefaultBranch(context.Background())
	if err != nil {
		t.Skipf("cannot determine default branch: %v", err)
	}

	// Get local branches.
	branches, err := r.LocalBranches(context.Background())
	if err != nil {
		t.Fatalf("LocalBranches: %v", err)
	}

	for _, b := range branches {
		if b.Name == defaultBranch {
			continue
		}

		result := r.CheckRebase(context.Background(), defaultBranch, b.Name)
		t.Logf("branch %s: direct=%v squashed=%v steps=%d failedStep=%d conflicts=%v",
			b.Name, result.CanRebase, result.CanRebaseSquashed,
			result.TotalSteps, result.FailedStep, result.Conflicts)
	}
}

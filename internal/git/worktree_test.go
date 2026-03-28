package git

import (
	"context"
	"testing"
)

func TestParseWorktrees_Single(t *testing.T) {
	input := "worktree /home/user/repo\nHEAD abc1234\nbranch refs/heads/main\n"

	wts := parseWorktrees(input)

	if len(wts) != 1 {
		t.Fatalf("got %d worktrees, want 1", len(wts))
	}

	wt := wts[0]
	if wt.Path != "/home/user/repo" {
		t.Errorf("Path = %q, want /home/user/repo", wt.Path)
	}

	if wt.HEAD != "abc1234" {
		t.Errorf("HEAD = %q, want abc1234", wt.HEAD)
	}

	if wt.Branch != "refs/heads/main" {
		t.Errorf("Branch = %q, want refs/heads/main", wt.Branch)
	}

	if wt.BranchShort() != "main" {
		t.Errorf("BranchShort() = %q, want main", wt.BranchShort())
	}

	if wt.Detached || wt.Bare || wt.Prunable {
		t.Error("expected Detached=false, Bare=false, Prunable=false")
	}
}

func TestParseWorktrees_Multiple(t *testing.T) {
	input := "worktree /home/user/repo\n" +
		"HEAD abc1234\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /tmp/linked\n" +
		"HEAD def5678\n" +
		"branch refs/heads/feature\n" +
		"\n" +
		"worktree /tmp/detached\n" +
		"HEAD 999aaaa\n" +
		"detached\n"

	wts := parseWorktrees(input)

	if len(wts) != 3 {
		t.Fatalf("got %d worktrees, want 3", len(wts))
	}

	if wts[0].BranchShort() != "main" {
		t.Errorf("wt[0] branch = %q, want main", wts[0].BranchShort())
	}

	if wts[1].BranchShort() != "feature" {
		t.Errorf("wt[1] branch = %q, want feature", wts[1].BranchShort())
	}

	if !wts[2].Detached {
		t.Error("wt[2] should be detached")
	}

	if wts[2].BranchShort() != "" {
		t.Errorf("detached worktree BranchShort() = %q, want empty", wts[2].BranchShort())
	}
}

func TestParseWorktrees_Prunable(t *testing.T) {
	input := "worktree /home/user/repo\n" +
		"HEAD abc1234\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /tmp/gone\n" +
		"HEAD def5678\n" +
		"branch refs/heads/old\n" +
		"prunable gitdir file points to non-existent location\n"

	wts := parseWorktrees(input)

	if len(wts) != 2 {
		t.Fatalf("got %d worktrees, want 2", len(wts))
	}

	if wts[0].Prunable {
		t.Error("main worktree should not be prunable")
	}

	if !wts[1].Prunable {
		t.Error("stale worktree should be prunable")
	}
}

func TestParseWorktrees_Empty(t *testing.T) {
	wts := parseWorktrees("")

	if len(wts) != 0 {
		t.Errorf("got %d worktrees, want 0", len(wts))
	}
}

func TestIntegration_Worktrees(t *testing.T) {
	r := &Runner{Dir: "."}

	wts, err := r.Worktrees(context.Background())
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}

	if len(wts) == 0 {
		t.Fatal("expected at least 1 worktree (the main one)")
	}

	main := wts[0]
	t.Logf("main worktree: path=%s branch=%s HEAD=%s", main.Path, main.BranchShort(), main.HEAD)

	if main.Path == "" {
		t.Error("main worktree path should not be empty")
	}

	if main.BranchShort() == "" {
		t.Error("main worktree should have a branch")
	}

	for i, wt := range wts[1:] {
		t.Logf("linked worktree[%d]: path=%s branch=%s detached=%v prunable=%v",
			i, wt.Path, wt.BranchShort(), wt.Detached, wt.Prunable)
	}
}

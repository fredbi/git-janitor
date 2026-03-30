package backend_test

import (
	"context"
	"os"
	"os/exec"
	"testing"

	git "github.com/fredbi/git-janitor/internal/git/backend"
)

// Integration tests that run against the actual git-janitor repository.
// These require git to be installed and the test to be run from within the repo.

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	// The test runs from the package directory; find the repo root.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	// Walk up to find .git.
	for {
		if _, statErr := os.Stat(dir + "/.git"); statErr == nil {
			return dir
		}

		parent := dir[:max(0, len(dir)-1)]
		if parent == dir || dir == "/" {
			t.Fatal("could not find repo root")
		}

		dir = parent
	}
}

func TestIntegration_Status(t *testing.T) {
	requireGit(t)

	r := git.NewRunner(repoRoot(t))
	s, err := r.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	// We can't assert specific values, but the branch should be non-empty
	// (unless on detached HEAD, which is unlikely in development).
	if s.OID == "" {
		t.Error("OID should not be empty")
	}

	t.Logf("branch=%s oid=%s upstream=%s ahead=%d behind=%d entries=%d",
		s.Branch, s.OID, s.Upstream, s.Ahead, s.Behind, len(s.Entries))
}

func TestIntegration_Remotes(t *testing.T) {
	requireGit(t)

	r := git.NewRunner(repoRoot(t))
	remotes, err := r.Remotes(context.Background())
	if err != nil {
		t.Fatalf("Remotes: %v", err)
	}

	if len(remotes) == 0 {
		t.Skip("no remotes configured")
	}

	for _, rm := range remotes {
		t.Logf("remote: %s fetch=%s push=%s", rm.Name, rm.FetchURL, rm.PushURL)
	}
}

func TestIntegration_RemoteMap(t *testing.T) {
	requireGit(t)

	r := git.NewRunner(repoRoot(t))
	m, err := r.RemoteMap(context.Background())
	if err != nil {
		t.Fatalf("RemoteMap: %v", err)
	}

	if len(m) == 0 {
		t.Skip("no remotes configured")
	}

	// Should have at least "origin".
	if _, ok := m["origin"]; !ok {
		t.Log("no 'origin' remote found, which is unusual but not an error")
	}
}

func TestIntegration_Branches(t *testing.T) {
	requireGit(t)

	r := git.NewRunner(repoRoot(t))
	branches, err := r.Branches(context.Background())
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}

	if len(branches) == 0 {
		t.Fatal("expected at least one branch")
	}

	var hasCurrent bool
	for _, b := range branches {
		if b.IsCurrent {
			hasCurrent = true
			t.Logf("current branch: %s (%s)", b.Name, b.Hash)
		}
	}

	if !hasCurrent {
		t.Log("no current branch marked (detached HEAD?)")
	}

	t.Logf("total branches: %d", len(branches))
}

func TestIntegration_DefaultBranch(t *testing.T) {
	requireGit(t)

	r := git.NewRunner(repoRoot(t))
	branch, err := r.DefaultBranch(context.Background())
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}

	if branch == "" {
		t.Error("DefaultBranch should not be empty")
	}

	t.Logf("default branch: %s", branch)
}

func TestIntegration_Stashes(t *testing.T) {
	requireGit(t)

	r := git.NewRunner(repoRoot(t))
	stashes, err := r.Stashes(context.Background())
	if err != nil {
		t.Fatalf("Stashes: %v", err)
	}

	t.Logf("stash count: %d", len(stashes))

	for _, s := range stashes {
		t.Logf("  %s on %s: %s", s.Ref, s.Branch, s.Message)
	}
}

package fs

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

// mkRepo creates a directory at path containing a `.git` directory so that
// IsGitDir returns true for it.
func mkRepo(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(path, ".git"), 0o755); err != nil {
		t.Fatalf("mkRepo %s: %v", path, err)
	}
}

// mkLinkedWorktree creates a directory at path containing a `.git` FILE
// (not directory) starting with "gitdir: …", as a real linked worktree
// would have.
func mkLinkedWorktree(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkLinkedWorktree dir %s: %v", path, err)
	}

	if err := os.WriteFile(filepath.Join(path, ".git"),
		[]byte("gitdir: /elsewhere/main/.git/worktrees/x\n"), 0o600); err != nil {
		t.Fatalf("mkLinkedWorktree file: %v", err)
	}
}

// gotKey extracts a Namespace+Name key from each result for stable
// equality checks across filesystems.
func gotKey(repos []models.RepoItem) []string {
	keys := make([]string, 0, len(repos))
	for _, r := range repos {
		k := r.Name
		if r.Namespace != "" {
			k = r.Namespace + "/" + r.Name
		}
		if !r.IsGit {
			k += " (not git)"
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

func TestDiscoverRepos_FlatLayoutBackwardCompat(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "repo-a"))
	mkRepo(t, filepath.Join(root, "repo-b"))
	if err := os.Mkdir(filepath.Join(root, "non-git"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	flat, err := DiscoverRepos(root)
	if err != nil {
		t.Fatalf("DiscoverRepos: %v", err)
	}
	depth1, err := DiscoverReposDepth(root, 1)
	if err != nil {
		t.Fatalf("DiscoverReposDepth(1): %v", err)
	}

	want := []string{"non-git (not git)", "repo-a", "repo-b"}

	if got := gotKey(flat); !equalStrings(got, want) {
		t.Errorf("DiscoverRepos = %v, want %v", got, want)
	}
	if got := gotKey(depth1); !equalStrings(got, want) {
		t.Errorf("DiscoverReposDepth(1) = %v, want %v", got, want)
	}
}

func TestDiscoverReposDepth_NestedLayout(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "myorg", "backend", "api"))
	mkRepo(t, filepath.Join(root, "myorg", "backend", "db"))
	mkRepo(t, filepath.Join(root, "myorg", "web", "site"))
	mkRepo(t, filepath.Join(root, "standalone"))

	repos, err := DiscoverReposDepth(root, 4)
	if err != nil {
		t.Fatalf("DiscoverReposDepth: %v", err)
	}

	want := []string{
		"myorg/backend/api",
		"myorg/backend/db",
		"myorg/web/site",
		"standalone",
	}
	if got := gotKey(repos); !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Spot-check namespace propagation on a nested entry.
	for _, r := range repos {
		if r.Name == "api" && r.Namespace != "myorg/backend" {
			t.Errorf("api Namespace = %q, want %q", r.Namespace, "myorg/backend")
		}
	}
}

func TestDiscoverReposDepth_DepthLimit(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "shallow"))                       // depth 1
	mkRepo(t, filepath.Join(root, "g1", "deep"))                    // depth 2
	mkRepo(t, filepath.Join(root, "g1", "g2", "deeper"))            // depth 3
	mkRepo(t, filepath.Join(root, "g1", "g2", "g3", "deepest"))     // depth 4
	mkRepo(t, filepath.Join(root, "g1", "g2", "g3", "g4", "abyss")) // depth 5

	repos, err := DiscoverReposDepth(root, 3)
	if err != nil {
		t.Fatalf("DiscoverReposDepth: %v", err)
	}

	want := []string{
		"g1/deep",
		"g1/g2/deeper",
		"shallow",
	}
	if got := gotKey(repos); !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDiscoverReposDepth_SkipsHiddenAndNoise(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "good"))
	mkRepo(t, filepath.Join(root, ".hidden", "evil"))
	mkRepo(t, filepath.Join(root, "node_modules", "evil"))
	mkRepo(t, filepath.Join(root, "vendor", "evil"))

	repos, err := DiscoverReposDepth(root, 4)
	if err != nil {
		t.Fatalf("DiscoverReposDepth: %v", err)
	}

	want := []string{"good"}
	if got := gotKey(repos); !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A dot-prefixed directory that is itself a git repo (e.g. a GitHub
// org-defaults ".github" repo) must be listed. Nested content under other
// dot-prefixed directories remains hidden.
func TestDiscoverReposDepth_DotRepoListedButTraversalStillSkipped(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "good"))
	mkRepo(t, filepath.Join(root, ".github")) // legitimate dot-named repo
	mkRepo(t, filepath.Join(root, ".hidden", "evil"))

	repos, err := DiscoverReposDepth(root, 4)
	if err != nil {
		t.Fatalf("DiscoverReposDepth: %v", err)
	}

	want := []string{".github", "good"}
	if got := gotKey(repos); !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDiscoverReposDepth_SkipsLinkedWorktrees(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "main"))
	mkLinkedWorktree(t, filepath.Join(root, "feature-branch"))
	// And nested linked worktree to make sure recursion also skips them.
	mkLinkedWorktree(t, filepath.Join(root, "group", "wt"))
	mkRepo(t, filepath.Join(root, "group", "real"))

	repos, err := DiscoverReposDepth(root, 4)
	if err != nil {
		t.Fatalf("DiscoverReposDepth: %v", err)
	}

	want := []string{"group/real", "main"}
	if got := gotKey(repos); !equalStrings(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDiscoverReposDepth_SymlinkLoop(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "real"))

	// Create a symlink that loops back to root. The walker must not
	// follow it (we filter symlinks via DirEntry.Type()).
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	// If the walker followed symlinks, this call would either deadlock
	// (caught by go test timeout) or stack-overflow.
	repos, err := DiscoverReposDepth(root, -1)
	if err != nil {
		t.Fatalf("DiscoverReposDepth: %v", err)
	}

	for _, r := range repos {
		if r.Name == "loop" {
			t.Errorf("walker followed symlink loop: %+v", r)
		}
	}
}

func TestDiscoverReposDepth_HardCapHonored(t *testing.T) {
	// Build a chain longer than maxRecursionDepth and confirm the walker
	// terminates without panicking. We don't assert a specific repo set
	// here — only that DiscoverReposDepth returns successfully with
	// MaxDepth=-1 (unlimited resolves to maxRecursionDepth).
	root := t.TempDir()
	deep := root
	for range maxRecursionDepth + 5 {
		deep = filepath.Join(deep, "x")
	}
	mkRepo(t, deep)

	if _, err := DiscoverReposDepth(root, -1); err != nil {
		t.Fatalf("DiscoverReposDepth: %v", err)
	}
}

func equalStrings(a, b []string) bool {
	return slices.Equal(a, b)
}

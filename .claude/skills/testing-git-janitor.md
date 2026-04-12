# Testing git-janitor

Project-specific testing knowledge, gotchas, and patterns for writing tests that interact
with git repositories and GitHub APIs.

## When to use

When writing or debugging tests for git-janitor — especially tests that shell out to git,
parse git output, or test the collection/check pipeline.

## Environment variables

Every git command must run with these environment variables:

```go
cmd.Env = append(os.Environ(),
    "LC_ALL=C",              // force English output — French locale produces "en retard de" instead of "behind"
    "GIT_TERMINAL_PROMPT=0", // prevent git from prompting for credentials (hangs the TUI)
)
```

**Where:** set in `git/backend.Runner.run()`. Tests that create their own `exec.Command` for git
must also set these.

## Branch name parsing

Branch names with slashes (`feature/foo`, `chore/lint`) look like remote branches in short form.

- `%(refname:short)` produces both `origin/main` (remote) and `chore/lint` (local) — both have slashes
- **Fix:** use `%(refname)` (full ref) and check for `refs/remotes/` vs `refs/heads/` prefix
- Bare remote ref roots (`origin`, `upstream`) appear in `git branch -a` output as entries with no
  further path component — filter these out

## Fast path vs full path collection

`CollectRepoInfo` is split into two modes to keep navigation responsive:

| Mode | Trigger | Includes | Skips |
|------|---------|----------|-------|
| **Fast** | selecting a repo | Status, Branches (basic), Remotes, Stashes, DefaultBranch, LastCommit, Config | Health, Size, FileStats, Tags, Activity, MergedBranches, CheckMergeable, CheckRebaseable |
| **Full** | Ctrl+R (refresh) | Everything in fast + all skipped fields | Nothing |

**Why:** full collection on large repos (e.g. golang/go at 459MB .git) takes 10+ seconds
(fsck 5s, rev-list 5s, diff 1.4s, cherry per branch). Fast path runs in <200ms.

**Testing implication:** if your check needs data only available in full mode, it will only
produce results after the user presses Ctrl+R. Document this in the check's godoc.

## Repack/GC thresholds

Small repos have high waste ratios due to structural .git overhead (hooks, logs, refs, indexes).
A 200KB .git with 20KB reachable = 10x ratio, but the "waste" is just 180KB of logs/refs.

**Minimum floors for alerting:**
- Pack count ≥ 20
- Pack size ≥ 1MB (for ratio-based alerts)
- Total .git size ≥ 5MB (for waste ratio)
- Absolute waste ≥ 1MB

Tests for gc/repack checks must use repos large enough to exceed these floors, or the
checks will correctly return no alert.

## Empty tree hash

The well-known SHA-1 empty tree hash (`4b825dc...82cf7137`) varies by git version/build.

**Never hardcode it.** Compute dynamically:
```sh
git hash-object -t tree /dev/null
```

This is used for `git diff --numstat` against an empty tree. Hardcoding causes silent failures
on some git builds.

## Fork detection

`DeriveKind` (clone vs fork) scans all remotes for distinct URLs, not just "upstream".

**Why:** typos like `upstram` are common. URL comparison catches forks regardless of remote naming.

**Pattern:** compare normalized URLs across all remotes. Two or more distinct URLs = fork.

## Test repo setup patterns

For integration tests that need a real git repo:

```go
func setupTestRepo(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    run := func(args ...string) {
        cmd := exec.Command("git", args...)
        cmd.Dir = dir
        cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
        if out, err := cmd.CombinedOutput(); err != nil {
            t.Fatalf("git %v: %v\n%s", args, err, out)
        }
    }
    run("init", "-b", "main")
    run("config", "user.email", "test@test.com")
    run("config", "user.name", "Test")
    // ... create commits, branches as needed
    return dir
}
```

**Key:** always set `user.email` and `user.name` in test repos — CI environments have no
global git config, and commits fail without them.

## GitHub API testing

The GitHub client (`github/backend.Client`) has a TTL cache. In tests:
- Use `FetchOptions{ForceRefresh: true}` to bypass the cache
- The client checks `Available()` (token present) — tests without `GH_TOKEN` skip GitHub checks
- Rate-limit state is per-client instance — each test should use a fresh client

## What's missing (testing roadmap)

Areas that need more test coverage:
- Multiple worktrees (check behavior when repo has active worktrees)
- CI-adapted tests (blank environment, no locale, no user git config)
- Edge cases: repos with thousands of branches, very large .git dirs, broken refs
- The two-wave async flow (git fast → GitHub second wave → merged alert re-evaluation)

# Plan: GitHub Checks Integration

> Last revised: 2026-03-29
> Move to `.claude/plans/github-checks-plan.md` on exit.

## Context

git-janitor has 23 git checks and 9 git actions working end-to-end (check → alert → action → execution).
The next family is **GitHub API checks** — read-only checks that surface GitHub-specific information
(archived repos, security alerts, unresponded issues, pending PRs) and enrich the Facts tab with
GitHub metadata.

The engine already has typed dispatch infrastructure for `CheckKindGitHub` (stubs in
`check_github.go`, `action_github.go`, `checks/github/register.go`). The `EvaluateRepo` method
currently skips non-git checks — we need a parallel `EvaluateGitHub` method.

## Key Decisions

1. **SDK**: `github.com/google/go-github/v68` (latest v6x)
2. **Auth**: `GITHUB_TOKEN` env, `GH_TOKEN` fallback. No token = silently skip all GitHub features
3. **Config**: Global `GitHub.Enabled` (default true) + per-root `GitHub` override.
   Allows disabling GitHub for specific roots (e.g. gitlab-hosted roots).
   Individual checks use existing `rules.checks` enable/disable mechanism
4. **Data fetching**: Async (background `tea.Cmd`) after git data arrives.
   Only when `SCM == "github"` AND token available AND config enabled for this root.
   Fast path stays fast — GitHub data arrives as a second wave
5. **Caching**: In-memory TTL cache (5min default, 10min when rate-limited).
   Ctrl+R bypasses cache
6. **Phase 1 actions**: None. All GitHub checks are informational-only.
   "Open in browser" is the only soft suggestion (no API writes)
7. **GitHubRepoData replacement**: The empty `engine.GitHubRepoData` placeholder becomes
   `github.RepoData` from `internal/github/`. Engine imports it like it imports `git.RepoInfo`

## Package Structure

```
internal/github/
  doc.go           — update existing
  repodata.go      — RepoData struct (implements engine.RepoInfo)
  client.go        — Client wrapper (token, rate-limit, go-github)
  collect.go       — CollectRepoData(ctx, client, owner, repo) → *RepoData
  parse.go         — ExtractOwnerRepo(remoteURL) → (owner, repo, err)
  cache.go         — TTL cache keyed by "owner/repo"
  parse_test.go    — URL parsing tests
```

## RepoData Type (`internal/github/repodata.go`)

```go
type RepoData struct {
    Owner, Repo, FullName string
    Description, HTMLURL  string
    IsFork, IsArchived, IsPrivate bool
    DefaultBranch string
    Topics        []string
    License       string   // SPDX or ""

    OpenIssues, OpenPRs int
    StarCount, ForkCount int

    VulnerabilityAlerts int  // -1 = unknown/no access

    ParentFullName string // "" if not a fork
    CreatedAt, UpdatedAt, PushedAt time.Time

    Err error // non-nil if API call failed
}

func (*RepoData) IsRepoInfo() {} // satisfies engine.RepoInfo
```

**API cost per repo**:
- **Cheap path** (initial batch): 1 call (`repos.Get`) — most fields
- **Full path** (with expensive checks): +1 (`pulls.List per_page=1`) for PR count,
  +1 (`issues.ListByRepo` filtered) for unresponded issues,
  +1 (GraphQL or REST) for vulnerability alerts
- Total: 1–4 API calls depending on which checks are enabled

## Check Catalog

### Initial batch (cheap — data from `repos.Get` only)

| Check Name | Severity | Default | What it detects |
|---|---|---|---|
| `github-repo-archived` | High | enabled | repo is archived on GitHub (read-only, no push) |
| `github-default-branch-mismatch` | Low | enabled | GitHub default branch ≠ local default branch |
| `github-description-missing` | Low | enabled | no description set on GitHub |
| `github-visibility-private` | Info | enabled | informational: repo is private |
| `github-repo-fork-parent` | Info | enabled | informational: fork parent identification |

### Expensive batch (extra API calls, opt-in)

| Check Name | Severity | Default | Extra API cost | What it detects |
|---|---|---|---|---|
| `github-issues-unresponded` | Medium | disabled | +1 call (paginated) | open issues with no comments |
| `github-prs-pending-review` | Medium | disabled | +1 call (paginated) | open PRs with no reviews |
| `github-security-alerts` | High | disabled | +1 call (GraphQL) | vulnerability alerts (needs token scope) |

Expensive checks disabled by default. Users opt-in per-root via `rules.checks`.

## Config Changes

### `internal/config/config.go`

```go
type Config struct {
    Roots    []LocalRoot
    Defaults struct {
        RootConfig *RootConfig
        Rules      *RulesConfig
    }
    GitHub GitHubConfig  // NEW — global default
}

type GitHubConfig struct {
    Enabled bool // default true
}
```

Per-root override in `RootConfig`:

```go
type RootConfig struct {
    ScheduleInterval time.Duration
    Rules            *RootRulesOverride
    GitHub           *GitHubConfig  // NEW — nil = inherit global default
}
```

New helper:

```go
func (c *Config) GitHubEnabled(rootIndex int) bool
```

Returns per-root override if set, otherwise global default.

### `internal/config/default_config.yaml`

```yaml
github:
  enabled: true
```

Add to checks list:
```yaml
    # GitHub (API-based, require GITHUB_TOKEN)
    - name: github-repo-archived
    - name: github-default-branch-mismatch
    - name: github-description-missing
    - name: github-visibility-private
    - name: github-repo-fork-parent
    - name: github-default-branch # used to cross-check git HEAD / heuristic
    # Expensive / opt-in (uncomment or enable per-root):
    # - name: github-issues-unresponded
    # - name: github-prs-pending-review
    # - name: github-security-alerts
```

Additional github information retrieval checks planned for phase 3:
```yaml
* github-protected-branches
* github-ci-workflows-status
* github-contributors
* github-latest-release # <- bonus: signed release? immutable release?
* github-security-settings-check # <- check against template for security settings
* github-push-rules-check # <- similar check against template
```

### Config wizard changes (UX)

The new github-specific fields in the config are configurable from the UX (config wizard, command /config)

## Engine Changes

### `internal/engine/check_github.go`

- Replace `GitHubRepoData struct{}` with import of `github.RepoData`
- Update `GitHubCheck.Evaluate` signature: `Evaluate(*github.RepoData)`
- Update `githubCheckEvaluator` interface accordingly

### `internal/engine/engine.go`

Add `EvaluateGitHub` method (parallel to `EvaluateRepo`):

```go
func (e *Engine) EvaluateGitHub(ctx context.Context, data *github.RepoData,
    enabledChecks []string) []Alert
```

Same pattern as `EvaluateRepo`: iterate `CheckKindGitHub` checks, filter by enabled, collect alerts, sort.

## UX Integration

### New message type (`internal/ux/types/types.go`)

```go
type GitHubInfoMsg struct {
    RepoPath string
    Data     *github.RepoData
}
```

### model.go flow

1. **Init**: `github.NewClient()` once at startup → `m.GitHubClient`
2. **After handleRepoInfo**: If `info.SCM == "github"` && client available &&
   `cfg.GitHubEnabled(rootIndex)`: extract owner/repo from origin URL, fire async `fetchGitHubCmd`
3. **handleGitHubInfo**: Store `m.LastGitHubData`, call `m.Right.SetGitHubData(data, enabledChecks)`
4. **Ctrl+R**: Also refetch GitHub data (bypass cache)

### infos.go

New method `SetGitHubData(data *github.RepoData, enabledChecks []string)`:
- Pass data to Facts panel for GitHub sub-section
- Call `engine.EvaluateGitHub` → get GitHub alerts
- Merge with existing git alerts (append + re-sort by severity)
- Update Alerts panel

### Facts tab — GitHub sub-section

When GitHub data is available, render after git facts:

```
  ── GitHub ──────────────
  Visibility:  public
  Fork:        owner/parent-repo
  Description: A tool for...
  Stars: 42  Forks: 2 Issues: 5  PRs: 2
  License:     Apache-2.0
  Archived:    no
```

NOTE: I assume that stars and forks are part of the generally available information for any repo
without extra API call.

## Error Handling

| Condition | Behavior |
|---|---|
| No token | `Client.Available() == false`. Skip all. Facts: "GitHub: no token" (dim) |
| Repo not GitHub-hosted | Skip. No GitHub section in Facts |
| Config disabled for root | Skip. No GitHub section in Facts |
| API 401/403 | `RepoData.Err` set. Facts: "GitHub: access denied" |
| API 404 | Same as 403 |
| Rate limited | Cache extends TTL. Status bar: "GitHub: rate limited" |
| Network error | `RepoData.Err` set. Facts: "GitHub: unavailable". Git data unaffected |

## Rate Limiting Strategy

- Authenticated: 5000 req/hour. 1–4 calls/repo
- Cache: 5min TTL default. Extended to 10min when `Rate.Remaining < 100`
- Hard stop when `Rate.Remaining < 20`: skip API calls, show warning
- Ctrl+R bypasses cache but still respects hard stop

## Implementation Steps

### Step 1: GitHub client + URL parsing (foundation, no UX)

| File | Action |
|------|--------|
| `go.mod` | Add `google/go-github/v68` |
| `internal/github/parse.go` | `ExtractOwnerRepo()` |
| `internal/github/parse_test.go` | URL parsing tests (SSH, HTTPS, edge cases) |
| `internal/github/repodata.go` | `RepoData` struct |
| `internal/github/client.go` | Client wrapper with token resolution |
| `internal/github/collect.go` | `CollectRepoData()` — cheap path only |
| `internal/github/cache.go` | TTL cache |

### Step 2: Engine wiring

| File | Action |
|------|--------|
| `internal/engine/check_github.go` | Replace placeholder with `github.RepoData` |
| `internal/engine/engine.go` | Add `EvaluateGitHub()` |
| `internal/config/config.go` | Add `GitHubConfig` (global + per-root), `GitHubEnabled()` |
| `internal/config/default_config.yaml` | Add github section + check names |

### Step 3: Initial batch of checks (vertical slice)

| File | Action |
|------|--------|
| `internal/checks/github/repo.go` | archived, description-missing, visibility, fork-parent |
| `internal/checks/github/branches.go` | default-branch-mismatch |
| `internal/checks/github/register.go` | Wire all 5 checks |
| `internal/checks/github/*_test.go` | Unit tests with synthetic RepoData |

### Step 4: UX wiring

| File | Action |
|------|--------|
| `internal/ux/types/types.go` | Add `GitHubInfoMsg` |
| `internal/ux/model.go` | GitHub client init, async fetch, handler |
| `internal/ux/panels/infos/infos.go` | `SetGitHubData()`, alert merging |
| `internal/ux/panels/infos/tab-facts/facts.go` | GitHub sub-section |

### Step 5: Acceptance testing (initial batch)

- Test on real repos: fredbi/*, go-openapi/*, a private repo, an archived repo, a fork
- Verify: no-token graceful degradation, rate limiting, cache, Facts display, alerts
- Test per-root disable on a gitlab root

### Step 6: Expensive checks

| File | Action |
|------|--------|
| `internal/github/collect.go` | Add `CollectExpensiveData()` — issues, PRs, vulns |
| `internal/checks/github/issues.go` | `github-issues-unresponded` |
| `internal/checks/github/prs.go` | `github-prs-pending-review` |
| `internal/checks/github/security.go` | `github-security-alerts` |
| `internal/checks/github/register.go` | Wire 3 additional checks |
| Unit tests | Synthetic RepoData with issue/PR/vuln data |

### Step 7: Acceptance testing (expensive batch)

- Enable expensive checks per-root on repos with known issues/PRs
- Verify pagination, rate-limit behavior, correct counting

## Verification

After each step:
1. `go build ./...` — clean build
2. `go test ./...` — all tests pass
3. `golangci-lint run` — no new lint issues

Milestone checks:
- After Step 1: `TestExtractOwnerRepo` passes, client resolves token
- After Step 3: unit tests pass with synthetic RepoData
- After Step 4: manual TUI test — GitHub facts + alerts appear for GitHub repos
- After Step 5: tested on real repos with various conditions
- After Step 7: expensive checks working on opt-in repos

## Critical Files

- `internal/engine/check_github.go` — replace placeholder, update base type
- `internal/engine/engine.go:95` — `EvaluateRepo` skips GitHub; new `EvaluateGitHub` parallels it
- `internal/engine/setup/setup.go` — already calls `githubchecks.RegisterAll` (becomes effective)
- `internal/ux/model.go` — GitHub client init + async fetch trigger + handler
- `internal/ux/panels/infos/infos.go` — `SetGitHubData` + alert merging
- `internal/git/inspect.go:177` — `DeriveSCM()` already detects GitHub repos
- `internal/config/config.go` — `GitHubConfig` (global + per-root)

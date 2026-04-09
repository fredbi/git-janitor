> [!NOTE]
> Last revision: 2026-04-09

# git-janitor project

## Summary

git-janitor is an interactive TUI tool for tending to local git repositories:
discover repos across configured roots, run checks (git + GitHub), surface alerts,
and execute fix actions — all from a single keyboard-driven interface.

The tool is **already highly useful in interactive mode**: walking repos and
triggering actions on a single key delivers massive productivity gains when
managing dozens or hundreds of git clones.

## Companion documents

- [git-janitor-plan.md](git-janitor-plan.md) — architecture and implementation plan
- [git-checks-catalog.md](git-checks-catalog.md) — git check/action specifications
- [github-checks-plan.md](github-checks-plan.md) — GitHub checks integration plan
- [github-checks-catalog.md](github-checks-catalog.md) — GitHub check specifications

## Current state

### What's done (Phase 1 — complete)

**Engine & pipeline:**
- Domain pipeline: Check → Alert → ActionSuggestion → Assignment → Result → History
- Typed dispatch per provider (GitCheck, GitHubCheck) via interface assertions
- Check and Action registries with insertion-ordered iteration
- `ActionSuggestion.Params` parallel slice for parameterized actions
- Persistence layer: bbolt KV store (cache + action history)
- Queued/throttled command runner

**Git checks (28 checks):**
- Branch hygiene: lagging, merged-not-deleted, gone-upstream, no-upstream, diverged, not-mergeable, empty
- Remote branch: merged-not-deleted, diverged
- Remote hygiene: no-origin, misnamed-upstream, exposed credentials
- Health: gc-advised, repack-advised, fsck-errors
- Activity: stale, inactive-dirty, stale-dirty, inactive-nondefault, stale-stash
- Worktree: dirty
- Tags: local-only, remote-only
- Config: no-email, unsigned
- File stats: large-files, binary
- Traits: shallow, submodules, LFS

**Git actions (12 actions):**
- Branch: delete, update, rebase, push, delete-remote, rebase-remote
- Maintenance: run-gc, run-gc-aggressive
- Tags: push-tag, fetch-tags
- Remote: rename-remote
- Clone: delete-local-clone
- Browser: open-in-browser

**GitHub checks (12 checks):**
- Repo: archived, description-missing, visibility-private, fork-parent
- Branches: default-branch-mismatch, branch-protection
- Security: security-alerts (3 APIs), security-not-enabled (admin distinction)
- Workflow: workflow-failures
- Issues & PRs: open issues, open PRs, pending PRs

**GitHub actions (3 actions):**
- set-repo-description (API write)
- delete-head-on-merge (API write)
- disable-workflows (API write)

**Config:**
- YAML config with defaults + per-root overrides
- `github.enabled` + `github.securityAlerts` (global + per-root)
- Config wizard with GitHub toggle fields
- `EnabledChecks()`, `IsActionAuto()`, `GitHubEnabled()`, `GitHubSecurityAlerts()`

**UX:**
- Left panel: tabbed roots with repo list
- Right panel: Facts, Branches, Stashes, Alerts, Actions, Activity tabs
- Facts tab: git metadata + GitHub sub-section with per-scanner security breakdown
- Alerts panel: severity-sorted cards with fix indicators, `c` to copy URL
- Actions panel: multi-line cards, Y/N confirmation, subject picker (Ctrl+P)
- Branch/stash detail panels with Enter
- Recent activity tab wired to bbolt history
- Clipboard: OSC 52 + xclip/xsel/wl-copy fallback
- Async GitHub fetch (second wave after git fast path)
- Help popup, theme cycling, command input

## Roadmap

### Phase 2: Consolidation (usability, trust, performance)

Goal: make the interactive tool rock-solid in all-terrain conditions.

1. **Alert rules wizard** — usability++. Configure checks/actions without editing YAML.
2. **Complete remaining acceptance tests** — trust++:
   - Multiple worktrees
   - Edge cases from TODO.md
3. **Robust task scheduling** — mostly done (queued/throttled runner),
   needs stress testing for fast navigation patterns.
4. **Consolidated alerts panel** — the "fleet dashboard" UX. View all alerts
   across all repos, grouped by severity/type. Navigate into a group → pick repo → act.
   This is the "mop 100 repos in 5 minutes" workflow.
   Lives on the left panel alongside root tabs.
5. **UX polish:**
   - Mini-form overlay for per-subject parameter editing (branch name, commit message)
   - Direct branch/stash actions from detail panels (D=delete, R=rebase)
   - Dependabot PR suggestions (@dependabot rebase for stale PRs)
   - `...more...` indicator accuracy for paginated GitHub data

### Phase 3: New features (impact)

1. 🔍 **External action protocol** (stdin/stdout, JSON context) — the big multiplier.
   Not just custom actions: _intelligent_ actions via AI agents.
   Candidates: conflict resolution, CI failure diagnosis, PR review.
   See TODO.md "AI features" section for detailed design of prompt-tool backend.
2. 📝 **Additional git checks:**
   - Unreachable-bloat + deep-clean action
   - Track-only clones → impose shallow
   - Fork: upstream sync (push origin to upstream)
   - Signing gotcha (gpgsign=true but no signingkey)
3. 📝 **Additional GitHub features:**
   - Gists, keys, old CI artifacts
   - Issue/PR detail views

### Phase 4: Quality & documentation

- ♥️ Relint (golangci-lint v2, `default: all` posture)
- ♥️ Set up CI (shared workflows for test, coverage, scanners, release via goreleaser)
- ♥️ Upgrade deps (bubble/v2, go-openapi/testify/v2)
- ♥️ Full SPDX headers, mockery mocks
- ♥️ Adapt tests for CI (blank environment)
- ♥️ User documentation (hugo doc-site)

### Phase 5: Semi-autonomous janitor

Thin layer over Phase 2 infrastructure — another engine implementation
with priority queue, rate limiter, human-confirmed batch mode.

### Beyond

- Pluggable actions (WASM / go plugins)
- Headless mode (TUI connects to daemon)
- AI agent integrations as first-class external actions

### Design backlog (unscheduled)

- Shallow clone management (unshallow, want-shallow, auto-clean binaries)
- Submodule sync (lagging remote, scan as independent repo, UI rendering)
- Stale .git/modules cleanup (orphaned submodule data)
- Reflog/notes cleanup (GC for refs, reflog, old git notes)
- Tags push parameterization (upstream-only, origin-only, all-remotes)
- Binary file exceptions (configurable regexp)
- Merge/rebase strategy selection
- Theme extraction as reusable bubble component
- Root path auto-completion, auto-discover roots
- Self-update

## Achievements

### Phase 2 consolidation — in progress (2026-04-09) ⭐⭐

- bbolt persistence layer: cache + action history
- Queued/throttled command runner (git + GitHub)
- Recent activity tab wired to persistent history
- Branch/stash detail panels
- Workflow failures check + disable-workflows action
- Issues/PRs checks, branch protection check
- Delete-head-on-merge action for forks
- Remote branch checks (merged-not-deleted, diverged)
- Remote credentials exposure check
- Inactive-dirty, stale-dirty, stale-stash, inactive-nondefault checks
- Branch-empty check (no own commits, safe to delete)
- Bug fixes: branch-merged-not-deleted false positives on empty/dirty current branches

### GitHub checks Phase 1 (2026-03-29) ⭐⭐⭐

- 7 initial GitHub checks + acceptance tested
- GitHub client with token resolution, rate-limit awareness, TTL cache
- Config: global + per-root GitHub settings
- Facts tab GitHub sub-section with per-scanner security breakdown
- Actions: delete-local-clone, open-in-browser, set-repo-description
- Clipboard: OSC 52 + xclip/xsel/wl-copy fallback
- Async data flow: git fast path → GitHub second wave → merged alerts

### Phase 1 — git checks complete (2026-03-29) ⭐⭐⭐

- 23 git checks + 9 actions, full acceptance testing
- Major bugs found and fixed during testing:
  French locale, branch name slashes, empty-tree hash, git auth hanging,
  push to wrong remote, panel refresh on root tabs, large repo performance,
  repack threshold calibration
- Engine foundation with typed dispatch, registries, history
- Full UX pipeline: alerts → actions → confirmation → execution → refresh

# git-janitor Roadmap

> Last revision: 2026-04-12

## Vision Statement

git-janitor is an interactive TUI for tending to local git repositories at scale.
The tool is already highly productive in interactive mode: walking repos and triggering
actions on a single key delivers massive time savings when managing dozens or hundreds
of git clones.

The roadmap progresses from polishing the interactive experience (Phase 2) through
new feature domains (Phase 3) toward AI-assisted and semi-autonomous operation (Phase 4+).

## Phase 1 — Foundation (complete) ✅

  ✅ **Engine & pipeline**: Check → Alert → ActionSuggestion → Result → History
  ✅ **28 git checks**: branch hygiene, remote, health, activity, worktree, tags, config, file stats, traits
  ✅ **12 git actions**: branch ops, maintenance, tags, remote, clone, browser
  ✅ **12 GitHub checks**: repo, branches, security, workflows, issues/PRs
  ✅ **3 GitHub actions**: set-description, delete-head-on-merge, disable-workflows
  ✅ **YAML config**: defaults + per-root overrides, merge logic, config wizard
  ✅ **Persistence**: bbolt KV store (cache + action history)
  ✅ **UX**: tabbed panels, alerts, actions with confirmation, branch/stash details, clipboard, async GitHub, help, themes
  ✅ **Quick actions**: Ctrl+K popup, init-commands, pre-commands, worktree support, branch subject

## Phase 2 — Consolidation (usability, trust, polish)

  Goal: make the interactive tool rock-solid in all-terrain conditions.

### Bugs & polish

  📝 **`...more...` indicator accuracy** for paginated GitHub data
  📝 **git gc: prune unreachable refs** + informative-only when < 10MB
  ✅ **After branch deletion: fetch with `--prune`** so remotes update
  📝 **Activity tab**: needs UX refactoring (pagination done but layout needs work)
  📝 **check upstream rebase shows even when no refresh, whereas local rebase don't

### UX improvements

  📝 **Foldable/expandable tree view** for repo hierarchy in left panel
  📝 **Consolidated all-alerts panel** — fleet dashboard: all repos grouped by severity on left panel. The "mop 100 repos in 5 minutes" workflow.
  📝 **Alert rules wizard** — configure checks/actions from TUI without editing YAML
  📝 **Direct branch/stash actions from detail panels** (D=delete, R=rebase)
  📝 **Mini-form overlay** for per-subject parameter editing — partially done, needs generalization to multi-subject actions
  📝 **Issue/PR detail views** — partially done, needs more detail content
  📝 **Theme in config** — persist selected theme across restarts
  📝 **Auto-discover roots** — scan common locations for git repos
  📝 **Prevent root cycles / duplicates** in config wizard

### Quick actions

  📝 **`/quick-actions` config wizard** with live testing
  📝 **Config reload check** — verify quick actions update when config changes mid-session
  ⛔ ~Auto-detect preferred terminal emulator~ — too brittle across emulators; `bash --init-file` is terminal-agnostic

### Testing

  📝 **Test case with multiple worktrees**
  📝 **Adapt tests for CI** — blank environment (no locale, no git config)

### Quality

  📝 **Use go-openapi/testify** test framework
  📝 **Full SPDX headers everywhere**
  📝 **CI & release** — goreleaser, binary artifacts, shared workflows

## Phase 3 — New features (impact)

### Git checks & actions

  📝 **Fork-upstream-lagging / fork-sync-upstream** — push origin commits to upstream
  📝 **Fork: rebase/merge/delete on remote branches** — partially done, ongoing for conflict cases
  📝 **Detect "track-only clones"** → impose shallow clone (needs config categorization)
  📝 **Unreachable-bloat + deep-clean action** for git gc
  📝 **Signing gotcha check** — gpgsign=true but no signingkey (needs implementation check)
  📝 **fork-upstream-lagging / fork-sync-upstream** : push origin commit to upstream

### GitHub features

  📝 **Dependabot PRs** — suggest `@dependabot rebase` for stale PRs with successful CI
  📝 **Gists** browsing/management
  📝 **Old & large CI artifacts** cleanup (maybe)
  ⛔ ~SSH keys management~ — out of scope (dedicated tool needed)

### Worktrees

  📝 **Worktrees as a first-class subject** — dedicated tab to browse worktrees with status, branch, delete action

### Infrastructure

  📝 **Runner throttling / generic queue** — FIFO with debounce, per-provider parallelism limits
  📝 **Clear history / older history** (e.g. > 90d)
  📝 **Rule-config wizard** — edit checks/actions config from TUI

### Dependency upgrades

  📝 **bubbles/v2** — blocked on upstream release
  ✅ ~Other deps upgraded~ (testify/v2, golangci-lint v2, etc.)

## Phase 4 — AI & automation

  🔨 **AI agent backend** — in progress (current WIP branch):
    - Manual actions panel (user-triggered actions outside check→alert flow)
    - Suggestion actions with prompt runner
    - Prompt-tool backend: new runner producing prompts for AI agents (conflict resolution, CI failure diagnosis)
  📝 **Check "material" stash or dirty** — AI-assisted cleanup (later)
  📝 **User documentation** — hugo doc-site

## Phase 5 — Semi-autonomous janitor

  📝 **Semi-autonomous engine** — priority queue, rate limiter, human-confirmed batch mode
  📝 **Shallow clone management** — unshallow, auto-clean binaries
  📝 **Submodule sync** — lagging remote, scan as independent repo
  📝 **Stale .git/modules cleanup**
  📝 **Reflog/notes cleanup**
  📝 **Binary file exceptions** — configurable regexp
  📝 **Merge/rebase strategy selection**

## Beyond (maybe / future)

  🧪 **External action protocol** — stdin/stdout JSON for external scripts and AI agents
  🧪 **Pluggable actions** — WASM / go plugins
  🧪 **Headless mode** — TUI connects to daemon
  🧪 **Tags push parameterization** — upstream-only, origin-only, all-remotes
  🧪 **Self-update** mechanism
  🧪 **Theme extraction** as reusable bubble component

## Legend

  ✅ Done
  📝 Planned (TODO)
  🔨 In progress
  ♥️ Nice-to-have / chore
  🧪 Maybe / experimental / future
  ⛔ Won't do (with rationale)

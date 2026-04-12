# Architecture rules (git-janitor)

These rules preserve the dependency structure and patterns established during the
April 2025 refactoring. Violating them reintroduces the tangled dependencies and
type-assertion chaos the refactoring eliminated.

## Dependency injection via `ifaces.Engineer`

`ifaces.Engineer` is the **sole mediator** between the UX layer and backend providers.

- UX packages (`internal/ux/...`) may import: `ifaces`, `models`, `config`, `engine` (constructor only), `registry`.
- UX packages must **never** import `git/backend`, `github/backend`, or any provider sub-package.
- All git/GitHub operations flow through `Engineer` methods: `Evaluate`, `Execute`, `Collect`, `CollectDetails`, `Refresh`, `QuickActionsFor`, `ExecuteQuickAction`.
- New capabilities must be added to the `Engineer` interface first, then implemented in `engine.Interactive`.

## Provider insulation

`internal/git/` and `internal/github/` are independent provider trees.

- They **never import each other**.
- Each has `backend/`, `checks/`, `actions/` sub-packages with the same structure.
- Only `internal/engine/` imports backends. This is where runner creation and provider dispatch live.
- Adding a new provider (e.g. GitLab) follows the same structure: `internal/gitlab/{backend,checks,actions}/`.

## Registry pattern

`registry.Registry[T Registrable]` is used for all extensible collections:

- **Checks:** `Registry[ifaces.Check]`
- **Actions:** `Registry[ifaces.Action]`
- **Themes:** `Registry[uxtypes.Theme]`
- **Quick actions:** `Registry[*quickactions.QuickAction]`

Registries are built at startup from producer functions (`iter.Seq[T]`) and injected via options.
Items must implement `Name() string` (the `Registrable` constraint). Names are unique within a registry — duplicates panic at construction.

Quick actions use composite keys (`{rootIndex}/{name}`) to allow per-root overrides of the same display name.

## Shared data model

All domain types live in `internal/models/`. No logic — pure data + enums + sort helpers.

- `RepoInfo` — the central data bag passed to checks and displayed by panels.
- `Branch`, `Stash`, `Remote`, `Tag` — git object models.
- `Alert`, `ActionSuggestion`, `ActionSubject` — check output → action input chain.
- `SubjectKind` — enum shared by checks, actions, and quick actions.
- `PlatformInfo` — GitHub API data (or future GitLab).

Providers populate `RepoInfo`; the engine caches it; the UX displays it. No type assertions at boundaries.

## UX patterns

### Theme propagation

Theme is a value type (`uxtypes.Theme`) stored as a field, not a global. Propagated via `New(theme)` constructors and `SetTheme(theme)` methods. The top-level `Model` owns the canonical copy and calls `setTheme()` to push changes to all sub-components.

### Centralized key bindings

All key bindings are declared in `ux/key/bindings.go` as `Binding` constants. Panels use `key.MsgBinding(msg)` to convert `tea.KeyMsg` → `Binding`, then switch on the constant. Never match on raw key strings outside `key/`.

### Panel base struct

Scrollable panels embed `panels.Base` (from `ux/panels/base.go`) for cursor, offset, sizing, and navigation key handling. Use `NavigateKey()`, `ClampScroll()`, `VisibleRange()` — don't reimplement scroll logic.

### Message-driven communication

Sub-components communicate with the top-level `Model` via bubbletea messages defined in `ux/types/types.go`. Sub-components never import `Model` — they return `tea.Cmd` closures that emit messages.

### Two-wave async collection

Data collection follows a two-wave pattern to keep navigation responsive:

1. **Wave 1 (git fast path):** when the user selects a repo, `Engine.Collect()` runs
   a fast git collection (status, branches, remotes, stashes — no health/filestat/merge checks).
   The result is sent as `RepoInfoMsg` → panels update immediately.

2. **Wave 2 (GitHub second wave):** after wave 1 completes, `triggerGitHubFetch()` fires
   an async `GitHubInfoMsg` with platform metadata. On arrival, **all checks are re-evaluated**
   (git + GitHub combined) so alerts reflect the complete picture.

This means:
- Checks must tolerate partial data (no `Platform` field in wave 1).
- GitHub checks skip silently when `info.Platform == nil`.
- Adding a new data source (e.g. GitLab) follows the same wave pattern: fast local data
  first, platform enrichment second, full re-evaluation on arrival.
- `Ctrl+R` (refresh) triggers a full git collection + force-refresh of platform data.

### Overlay rendering priority

In `Model.View()`, overlays are checked in priority order. Centered overlays (Help, Detail, Wizard) replace the frame entirely. Anchored overlays (QuickActions) are spliced into the frame. Only one overlay is active at a time.

## Config merge contract

`config.LoadDefault()` merges embedded defaults with the user config file. New entries (checks, actions, quick-actions) added to `default_config.yaml` are automatically picked up by existing users — the merge appends missing entries by name.

The same merge-by-name pattern applies at the root level for quick actions: `Config.QuickActionsForRoot(idx)` merges global defaults with per-root overrides.

## Engine reload contract

`Engine.Reload(cfg)` must rebuild all config-derived state (quick-actions registry, enabled checks, etc.) so that wizard saves and config changes take effect immediately without restarting.

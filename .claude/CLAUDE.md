# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.
For detailed patterns, see the **skills/** and **rules/** directories.

## Project Overview

**git-janitor** is a terminal UI (TUI) application that monitors and maintains local git
repositories. It scans configured root directories, displays repo status, and provides
actions to keep repositories in good shape (fetch, rebase, push, stash management, etc.).

- Module: `github.com/fredbi/git-janitor`
- Go version: 1.25.0
- License: Apache-2.0
- TUI framework: [bubbletea](https://github.com/charmbracelet/bubbletea) + bubbles + lipgloss

## Package Structure

### Top level

| Path | Purpose |
|------|---------|
| `cmd/git-janitor/` | Entry point: loads config, builds registries, wires engine + UX, runs TUI |
| `internal/` | All application logic (see below) |
| `docs/` | Documentation (STYLE.md, etc.) |

### Core (`internal/`)

| Package | Purpose |
|---------|---------|
| `config/` | YAML-based configuration: roots, rules, quick-actions. Embedded defaults + user overlay + merge. |
| `models/` | Shared domain types: `RepoInfo`, `Branch`, `Alert`, `ActionSuggestion`, `SubjectKind`, enums. No logic. |
| `ifaces/` | Interfaces: `Engineer`, `Check`, `Action`, `SelfDescribed`. The contract between UX and providers. |
| `ifaces/mocks/` | Generated mocks (mockery). Regenerate: `mockery --config internal/.mockery.yml` from `internal/`. |
| `engine/` | `Interactive` engine: orchestrates checks, actions, collection, caching, history. Implements `ifaces.Engineer`. |
| `registry/` | Generic `Registry[T Registrable]`: insert-order, name-indexed, used for checks, actions, themes, quick-actions. |

### Providers (`internal/git/`, `internal/github/`)

Each provider has the same sub-package layout:

| Sub-package | Purpose |
|-------------|---------|
| `backend/` | Runner (git CLI) or Client (GitHub API). Low-level operations. |
| `checks/` | Check implementations. Each implements `ifaces.Check`. |
| `actions/` | Action implementations. Each implements `ifaces.Action`. |
| `all_checks.go` / `all_actions.go` | Producer functions returning `iter.Seq` for registry construction. |

**Providers never import each other.** The engine is the sole consumer of backends.

### Quick Actions (`internal/quickactions/`)

User-configured shell commands launched via Ctrl+K. Config-driven (not built-in).
Registry keys are `{rootIndex}/{name}`. Supports pre-commands (synchronous setup),
init-commands (run inside spawned shell via `bash --init-file`), and placeholder
substitution (`{{repo}}`, `{{workdir}}`, `{{branch}}`, `{{worktree}}`, `{{init-file}}`).

### Storage (`internal/store/`)

| Package | Purpose |
|---------|---------|
| `store/` | `Store` interface (key-value, bucket-scoped) |
| `store/bolt/` | BoltDB implementation for caching `RepoInfo` and action history |

### UX (`internal/ux/`)

| Package | Purpose |
|---------|---------|
| `ux/` | Top-level `Model` (bubbletea): owns all panels, handles key dispatch, layout, overlays |
| `ux/key/` | Centralized key bindings (`key.Binding` enum, `MsgBinding()` helper) |
| `ux/gadgets/` | Reusable UI components: `DetailPopup`, `QuickActionsPopup`, `PathAutocomplete`, clipboard |
| `ux/panels/repos/` | Left panel: tabbed repo list (one tab per root), filter, group headers |
| `ux/panels/infos/` | Right panel: tabbed info display (Facts, Branches, Alerts, Actions, Activity, Stashes, Recent) |
| `ux/panels/infos/tab-*/` | Individual tab implementations |
| `ux/panels/` | Shared `Base` struct for scrollable, cursor-driven panels |
| `ux/commands/` | Command input bar + command dispatch |
| `ux/commands/help/` | Help popup with contextual help per panel/tab |
| `ux/commands/config-wizard/` | In-app wizard for adding/editing root directories |
| `ux/commands/scan/` | Background root scanning |
| `ux/statusbar/` | Bottom status bar (message + progress spinner) |
| `ux/themes/` | Color theme definitions |
| `ux/types/` | Shared UX types: messages (`RepoInfoMsg`, `ExecuteActionMsg`, etc.), `Theme` struct |

### Other

| Package | Purpose |
|---------|---------|
| `fs/` | Filesystem utilities: repo discovery walker, home expansion, skip-dir rules |
| `history/` | Action history types |

## Build & Test

```sh
go build ./...
go test ./...
golangci-lint run --new-from-rev master
```

Regenerate mocks after changing `ifaces.Engineer`, `store.Store`, or `registry.Registrable`:
```sh
cd internal && mockery --config .mockery.yml
```

## Key Navigation

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle focus: Repos → Right pane → Input |
| `Ctrl+A` / `←` / `→` | Cycle tabs within focused panel |
| `j`/`k` or `↑`/`↓` | Navigate items |
| `g` / `G` | Jump to top / bottom |
| `Enter` | Perform action / show details |
| `/` | Jump to command input |
| `Ctrl+K` | Quick actions popup (context-dependent subject) |
| `Ctrl+R` | Fetch and refresh selected repo |
| `Ctrl+H` | Contextual help |
| `Ctrl+D` | Show status bar details |
| `Esc` / `q` | Close popup / clear filter |
| `Ctrl+C` / `Ctrl+Q` | Quit |

## Configuration

Config is YAML-based (`~/.config/git-janitor/config.yaml`). Embedded defaults are
merged with user config on load — new built-in checks, actions, and quick-actions
are picked up automatically.

Key sections:
- `roots:` — directories to scan (path, name, interval, maxDepth, per-root overrides)
- `defaults.rules:` — enabled checks and actions with auto/confirmation settings
- `quick-actions:` — user-defined shell commands (subject, name, command, pre-commands, init-commands)
- `github:` — GitHub API integration (enabled, securityAlerts)

## Further reading

- **Adding checks/actions:** `.claude/skills/new-check-action.md`
- **Adding/extending quick actions:** `.claude/skills/quick-actions.md`
- **Adding UX features:** `.claude/skills/ux-gadget.md`
- **Testing git-janitor:** `.claude/skills/testing-git-janitor.md`
- **Architecture rules:** `.claude/rules/architecture.md`

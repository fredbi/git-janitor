# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**git-janitor** is a terminal UI (TUI) application that monitors and maintains local git
repositories. It scans configured root directories, displays repo status, and provides
actions to keep repositories in good shape (fetch, rebase, push, stash management, etc.).

- Module: `github.com/fredbi/git-janitor`
- Go version: 1.25.0
- License: Apache-2.0
- TUI framework: [bubbletea](https://github.com/charmbracelet/bubbletea) + bubbles + lipgloss

## Package Structure

| Package | Purpose |
|---------|---------|
| `.` (root) | Package-level doc only |
| `internal/git/` | Git operations: branch, remote, status, stash, fetch parsing |
| `internal/state/` | Bubbletea model, panels (repos, alerts, actions, recent), config wizard, scanner |
| `internal/config/` | YAML-based configuration (roots, scan intervals) |
| `internal/fs/` | Filesystem utilities |
| `pkg/actions/` | Planned: local and remote git actions (fetch, stash, rebase, merge, sync-fork, etc.) |
| `pkg/explorer/` | Planned: repository explorer |
| `pkg/store/sqlite/` | Planned: SQLite-backed persistent store |
| `pkg/ui/` | Planned: UI components and events |

## Architecture

### TUI Layout

- **Left panel (repos)**: Tabbed panel, one tab per configured root directory. Each tab lists discovered repos.
- **Right panel**: Context-dependent (alerts, actions, recent activity).
- **Status bar**: Bottom bar with mode/status info.
- **Config wizard**: In-app wizard to add/edit root directories (path, name, scan interval).
- **Help popup**: `?` key shows keybinding reference.

### Configuration

Config is YAML-based (`internal/config/`). Each "root" has:
- `Path` — directory to scan for git repos
- `Name` — display name (defaults to `filepath.Base(path)`)
- `Interval` — scan frequency

### Scanner

The scanner (`internal/state/scanner.go`) walks configured roots, discovers git repos,
and returns results grouped by root index for populating the tabbed panel.

## Build & Test

```sh
go build ./...
go test ./...
```

## Key Navigation

- `Tab`/`Shift-Tab`: cycle panels
- `Ctrl+A` / arrow keys: cycle tabs within a panel
- `j`/`k` or arrows: navigate items
- `Space`: toggle selection
- `Enter`: perform action
- `?`: help popup
- `q`: quit

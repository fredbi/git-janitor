# Adding or extending quick actions

Step-by-step guide for adding new quick actions or extending the quick-actions system.

## When to use

When you need to add a new user-launchable shell command to the Ctrl+K popup — either as a
built-in default or to extend the framework (new placeholders, new subjects, new execution phases).

**Quick actions are not `ifaces.Action`.** They are config-driven shell commands in
`internal/quickactions/`, not built-in check/action pairs. For adding checks and actions,
see the `new-check-action` skill.

## How quick actions work

1. User presses **Ctrl+K** → Model determines subject from focus context (repo, branch, ...)
2. Engine returns registered quick actions for that `(rootIndex, subject)` pair
3. Popup shows eligible actions; user picks one with Enter
4. `Run()` executes pre-commands → generates init script → spawns terminal

## Config structure

```yaml
quick-actions:
  - subject: repo              # SubjectKind string: repo, branch, stash, ...
    name: open-in-terminal     # unique key (per root scope)
    description: Open repo in a terminal
    pre-commands:              # optional: run synchronously BEFORE terminal spawn
      - 'git worktree add "{{worktree}}" "{{branch}}"'
    init-commands:             # optional: run INSIDE the spawned shell
      - git status
    command:                   # the terminal emulator command
      - gnome-terminal
      - '--'
      - bash
      - '--init-file'
      - '{{init-file}}'
```

**Per-root overrides:** A root can declare quick actions with the same name as globals — they
merge by name (root wins). Registry keys are `{rootIndex}/{name}`.

## Available placeholders

| Placeholder | Source | Available when |
|-------------|--------|----------------|
| `{{repo}}` | `RepoItem.Name` (leaf dir name) | Always |
| `{{workdir}}` | `RepoItem.Path` (full path) | Always |
| `{{subject}}` | Selected item name | Always (defaults to repo path) |
| `{{branch}}` | Selected branch name | Subject = branch |
| `{{worktree}}` | `{workdir}/.git-janitor-worktrees/{sanitized-branch}` | Subject = branch |
| `{{init-file}}` | Auto-generated temp script path | When init-commands present |

Placeholders are substituted in: command args, pre-commands, init-commands.

## Execution phases

### Pre-commands (`pre-commands:`)
- Run **synchronously** via `bash -c` in `{{workdir}}`
- If any exits non-zero → quick action aborted, error shown in status bar
- Use for setup that must succeed before the terminal opens (e.g. `git worktree add`)
- **File:** `internal/quickactions/precommands.go`

### Init script (`init-commands:`)
- Written to a temp file, passed via `bash --init-file`
- Standard preamble (always included): source `~/.bashrc`, set terminal title via PS1 patching, `cd` into workdir
- User commands appended after preamble
- **File:** `internal/quickactions/initscript.go`

### Worktree redirect
After pre-commands, if `{{worktree}}` exists as a directory, `{{workdir}}` is automatically
redirected to it — so the init script's `cd` and the terminal's cwd both land in the worktree.

## Step-by-step: Adding a new default quick action

### 1. Add the YAML entry

**File:** `internal/config/default_config.yaml`

Add under `quick-actions:`. The merge logic in `LoadDefault()` will auto-append it to
existing user configs.

### 2. Add new placeholders (if needed)

**File:** `internal/ux/ux-model.go` → `runQuickAction()`

The `params` map is built here. Add new keys for your subject context:
```go
params["mykey"] = computedValue
```

### 3. Add new subject context (if needed)

**File:** `internal/ux/ux-model.go` → `quickActionsSubject()`

Map focus state to `models.SubjectKind`:
```go
case infos.TabMyNewTab:
    return models.SubjectMyKind, true
```

Also update `quickActionsAnchor()` for popup positioning and `runQuickAction()` for
subject-specific param population.

### 4. Expose panel selection (if needed)

If the new subject needs a selected item from a panel (like `SelectedBranch()`), add:
- A `SelectedX()` method on the tab panel (`tab-*/`)
- A forwarding method on `infos.Panel`
- Use it in `runQuickAction()` to populate params

### 5. Update help text

**File:** `internal/ux/commands/help/help.go`

Add Ctrl+K documentation to the relevant contextual help entry.

### 6. Test

```sh
go build ./...
go test ./internal/config/... ./internal/quickactions/...
golangci-lint run --new-from-rev master
```

Manual: select a matching item, press Ctrl+K, verify popup shows the new action,
press Enter, verify the terminal opens correctly.

## Step-by-step: Extending the framework

### Adding a new execution phase

Follow the pre-commands pattern:
1. Add field to `config.QuickActionConfig`
2. Add field to `quickactions.QuickAction` and `quickactions.Params`
3. Wire in `quickactions.Run()` at the right point in the lifecycle
4. Update registry builder to pass the new field
5. Update tests

### Adding support for a new terminal emulator

No code changes needed — users configure their terminal in YAML. The `bash --init-file`
mechanism is terminal-agnostic. Only the `command:` array changes:

```yaml
# wezterm example
command:
  - wezterm
  - start
  - '--cwd'
  - '{{workdir}}'
  - '--'
  - bash
  - '--init-file'
  - '{{init-file}}'
```

## Design philosophy

- **Config-driven, not built-in:** Quick actions are YAML, not Go code. Users own their setup.
- **No precondition filtering:** All eligible actions are shown. Let the command fail visibly.
- **Terminal-agnostic:** The init-file mechanism works with any emulator that can spawn `bash`.
- **Deterministic worktrees:** `{repo}/.git-janitor-worktrees/{sanitized-branch}` — reusable, no orphans.

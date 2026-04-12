# Adding a new Check + Action pair to git-janitor

Step-by-step guide for implementing a new check with an associated repair action.

## When to use

When you need to add a new automated detection (check) with a suggested fix (action) to git-janitor. This applies to both git CLI checks and GitHub API checks.

**Not for quick actions.** Quick actions (`internal/quickactions/`) are user-configured shell commands declared in YAML — they don't implement `ifaces.Action` and are not registered via `all_actions.go`. See the `quick-actions` skill for their design.

## Prerequisites

- Understand whether this is a **git check** (`internal/git/checks/`) or a **GitHub check** (`internal/github/checks/`)
- Understand the **SubjectKind** the action operates on (Branch, Stash, Remote, Repo, etc.) — see `internal/models/enums.go`
- Know which **runner methods** are needed — git runner (`internal/git/backend/`) or GitHub client (`internal/github/backend/`)

## Step-by-step

### 1. Add git commands (if needed)

**File:** `internal/git/backend/git_commands.go`

Add command builder functions. These return `[]string` slices for `Runner.run()`:

```go
func cmdMyNewCommand(arg string) []string {
    return []string{"my-command", "--flag", arg}
}
```

### 2. Add runner methods (if needed)

**File:** `internal/git/backend/actions.go` (for git actions) or `internal/github/backend/github_runner.go` (for GitHub)

Git runner pattern:
```go
func (r *Runner) MyAction(ctx context.Context, arg string) models.ActionResult {
    _, err := r.run(ctx, cmdMyNewCommand(arg)...)
    if err != nil {
        return models.ActionResult{Message: fmt.Sprintf("my-action failed: %v", err)}
    }
    return models.ActionResult{OK: true, Message: "done"}
}
```

GitHub client pattern:
```go
func (c *Client) MyGitHubAction(ctx context.Context, owner, repo string) error {
    _, resp, err := c.gh.SomeService.SomeMethod(ctx, owner, repo, &gogithub.Options{})
    c.updateRate(resp)
    return err
}
```

### 3. Create the action

**File:** `internal/git/actions/my_action.go` or `internal/github/actions/my_action.go`

```go
type MyAction struct {
    gitAction // or githubAction for GitHub actions
}

func NewMyAction() MyAction {
    return MyAction{
        gitAction: gitAction{
            Describer: models.NewDescriber(
                "my-action",           // registered name (used in config + suggestions)
                "description of action", // shown in /actions listing
            ),
        },
    }
}

// Override defaults as needed:
func (MyAction) ApplyTo() models.SubjectKind { return models.SubjectBranch }
func (MyAction) Destructive() bool           { return true }  // requires Y/N confirmation
func (MyAction) ParamPrompt() string         { return "" }    // non-empty = show text input

func (a MyAction) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
    runner, err := runnerCtx(ctx)
    if err != nil {
        return models.Result{}, err
    }
    result := runner.MyAction(ctx, params[0])
    return result.ToResult(), nil
}
```

**Key patterns:**
- `params` comes from `ActionSubject.Params` (via `executePerSubject`) or subject names (direct execution)
- For per-subject actions with params, the engine prepends `Subject` to `Params`: `params = [subject, params...]`
- Use `result.ToResult()` to convert `ActionResult` → `Result` (includes CommandLog)
- The runner's `StartLogging()` is called by the engine — command log is automatic

### 4. Create the check

**File:** `internal/git/checks/my_check.go` or `internal/github/checks/my_check.go`

```go
type MyCheck struct {
    gitCheck // or githubCheck
}

func NewMyCheck() MyCheck {
    return MyCheck{
        gitCheck: gitCheck{
            Describer: models.NewDescriber(
                "my-check",
                "description shown in /checks listing",
            ),
        },
    }
}

func (c MyCheck) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
    return c.evaluate(info)
}

func (c MyCheck) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
    // Filter subjects that match the condition.
    subjects := filterBranches(info, func(b models.Branch) bool {
        return /* condition */
    })

    if len(subjects) == 0 {
        return noAlert(c.Name())
    }

    return singleAlert(models.Alert{
        CheckName:   c.Name(),
        Severity:    models.SeverityLow, // None, Info, Low, Medium, High, Critical
        Summary:     fmt.Sprintf("%d issue(s) found", len(subjects)),
        Detail:      subjectsDetail(subjects),
        Suggestions: []models.ActionSuggestion{{
            ActionName:  "my-action",
            SubjectKind: models.SubjectBranch,
            Subjects:    subjects,
        }},
    }), nil
}
```

**Helpers available in checks packages:**
- `filterBranches(info, func)` — filter branches by predicate
- `filterTags(info, func)` — filter tags by predicate
- `simpleSubject(names...)` — create `[]ActionSubject` from names
- `branchSuggestion(action, subjects)` — shorthand for branch suggestions
- `noAlert(name)` — return SeverityNone (check passed)
- `singleAlert(alert)` — return one alert as `iter.Seq`
- `subjectsDetail(subjects)` — comma-joined subject names
- `forkPlatform(info)` — get the PlatformInfo for the fork side (GitHub checks)

**For GitHub checks that need fork data:**
```go
func (c MyCheck) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
    fork := forkPlatform(info) // checks both Platform and UpstreamPlatform
    if fork == nil {
        return noAlert(c.Name())
    }
    return c.evaluate(fork)
}
```

**For checks that emit multiple alerts** (e.g. rebasable + stuck):
```go
return func(yield func(models.Alert) bool) {
    if len(rebasable) > 0 {
        if !yield(models.Alert{/* rebasable alert */}) { return }
    }
    if len(stuck) > 0 {
        if !yield(models.Alert{/* stuck alert */}) { return }
    }
}, nil
```

### 5. Register the check and action

**Checks:** `internal/git/all_checks.go` or `internal/github/all_checks.go`
```go
checks.NewMyCheck(),
```

**Actions:** `internal/git/all_actions.go` or `internal/github/all_actions.go`
```go
actions.NewMyAction(),
```

### 6. Add to default config

**File:** `internal/config/default_config.yaml`

Under `checks:`:
```yaml
      - name: my-check
```

Under `actions:`:
```yaml
      - name: my-action
        auto: false  # true = no confirmation needed
```

**Note:** The config merge (`LoadDefault`) automatically adds new default checks/actions to existing user configs. Users don't need to update their config file.

### 7. Build and test

```sh
go build ./...
go test ./...
golangci-lint run --new-from-rev master
```

**Manual testing:** select a repo that matches the check condition, verify the alert appears in the Alerts tab, press Enter to see suggestions, execute the action.

## Common patterns

### Action with params from check suggestion

The check provides params in `ActionSubject.Params`. The engine's `executePerSubject` prepends the subject name:

```
Check creates: Subject: "feature", Params: ["main"]
Engine passes: params = ["feature", "main"]
Action reads:  params[0] = "feature" (old), params[1] = "main" (new)
```

### Fork-aware checks

Use `forkPlatform(info)` to find the fork regardless of which remote (origin or upstream) it's on. Check `HasAdminAccess` before suggesting admin-level actions.

### Remote branch checks

- Use `strings.HasPrefix(b.Name, models.RemoteUpstream+"/")` to filter upstream branches
- Use `b.Merged`, `b.AheadOnly`, `b.RebaseCheck`, `b.MergeCheck` for pre-validated state
- These fields are populated during full collection (`Ctrl+R`) for upstream remote branches

### Collection prerequisites

If your check needs data that isn't collected in the fast path, it will only work after `Ctrl+R` (full refresh). Data collected in fast vs full:

- **Fast** (`CollectFast`): Status, Branches (no merge/rebase checks), Remotes, Stashes, DefaultBranch, LastCommit, Config
- **Full** (Ctrl+R): Everything in fast + Health, Size, FileStats, Tags, Activity, MergedBranches, CheckMergeable, CheckRebaseable, MarkRemoteAheadOnly

### Severity guidelines

- **Critical**: data loss risk, corruption
- **High**: security issue (credentials in URL), needs immediate attention
- **Medium**: branch protection missing, merged branches to clean up
- **Low**: housekeeping (stale branches, diverged remotes, inactive repos)
- **Info**: informational only
- **None**: check passed, no issue

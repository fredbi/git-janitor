# Project Plan: Orchestration Layer for git-janitor

## Vision

### The problem to solve

I am the maintainer of dozens of public repos on github.
These may either belong to the organizations I am supporting (go-openapi, go-swagger) or personal repositories.

At my job, I am the caretaker / janitor of about 100-150 gitlab repos (gitlab entreprise).

In addition, as a naturally prolific developer, I am keen on forking & contributing things to whatever
project I like, I need or want to support at some point in time.

My problem is: even with a git GUI (e.g. gitkraken), I just can't cope with the administrative burden
to manage all my clones (on multiple dev computers...), git configurations and github settings.

### How the janitor helps

We build a janitor to tend to our git repositories (local or remote) and alleviate much
of the housekeeping tasks, often involving complex git syntax.

* Phase 1: tending to repos manually but from the TUI is already a great time-saver. All actions are
  triggered by the user from the TUI and executed synchronously. All configuration can be carried out
  interactively from the TUI: the user may play with the YAML file, but is never compelled to do so.

* Phase 2: the git-janitor is open and schedules checks, actions and assignments in the background,
  prioritizing tasks so as not to be intrusive with resource intensive periods of activity by the
  developer, and being gentle on github API rate limits.
  New TUI interactions are needed to show the progress status of these automatic actions.
  When phase 2 is complete, I just have to take a look from time to time to persistent alerts that need
  special attention.
  New persistence layer (e.g. local embedded KV store or similar light db) is needed to track alerts, scheduled
  actions etc. Configuration is not enough for that.

* Phase 3: the git-janitor becomes extensible and allows the user to configure external actions (scripts, commands)
  that take the alert as input.
  This may be anything (e.g. a bash script). The primary use case is intended to run AI-capable agents with an
  accurate description of what's wrong.

  Examples:
  * git alert: "branch X is not mergeable" -> AI-agent trained to resolve merge conflicts intelligently
  * github alert: "CI job failed" -> AI-agent trained to retrieve CI logs and analyze the failure

* Phase 4: cover edge cases / extended use-cases
  The following use-cases are deliberately ignored for now, but we have a few repos that may justify some
  degree of support in the future:
  * fork with multiple upstreams (source upstream, fork from source - our origin -, personal fork from this fork - our upstream )
  * repos using different ssh keys (for now assume all ssh is uniformly and transparently resolved, assuming
     sshd provides all the authentication material transparently)
  * nested roots (for now roots are explored assuming none is containing another root)
  * support for git notes (related to another endeavor to introduce git notes to automatically produce high-quality
    release notes)


#### What git-janitor is

git-janitor is a local development tool. It is installed on my development machines
and acts locally.

It is a fast, responsive TUI written in go. It uses git command line as it we may safely assume
it is installed locally.

git-janitor discovers my repos from local clones. There is a configuration that drives this.

There are 2 main use-cases for local repos:
* clone (I am cloning a repo for either working directly on it - e.g. personal project -
  or simply to observe it source code without contributing (1 remote, or 2 identical remotes)
* fork (2 different remotes) : this is intended to contribute, and we may want to maintain the fork
  up-to-date with its upstream (but not necessarily, e.g. we fork out with breaking changes)

For each repo, git-janitor knows about its remote setup, branches, tags, stashes and git config.
It may determine repos or branches that are stale or active.

Its primary use-case is to maintain branch and fork hygiene (updating, pulling, rebasing etc).
It knows which branches are mergeable/rebasable and which are not.

Other indicators tell us about various git health metrics that may require special cleanup actions
(e.g. git gc, unattended presence of binary files or large files).

Eventually, the janitor may remind me about stale repositories or branches, with little or no activity,
or no release in a while.

git-janitor runs deterministic, predictable checks and actions based on configuration.

The janitor knows its tenants well and repos may be configured differently while sharing some common defaults.

Built-in capabilities include support for git (now) and gihub interactions (soon).
A mechanism to inject custom checks and actions allows the user to handle bespoke operations or to
extend what we can do _from_ the janitor.

#### What git-janitor is not

git-janitor is not a git UI like gitkraken (GUI) or lazygit (TUI): it doesn't allow the user
to interact with the contents of the repos in a granular way such as browsing files or checking
for diffs.

git-janitor is not intended to run autonomously as a bot or in a CI environment. This is a purely local tool.

The janitor is not equipped with AI capabilities. However, it might trigger AI-capable external actions (see below § Janitorial assignments).

I don't have plans to use git-janitor as a traditional CLI tool (e.g. with flags, etc) to operate in non-interactive mode. At least not yet.

### Janitorial concepts

* Check: a simple question asked to a repository that may or may not raise an alert

  Examples (git):
  * "are there local branches lagging behind their remote counterpart?"
  * "are there local branches that don't have any remote counterpart?"

  Examples (github):
  * "are there new un-responded issues on this repo?""
  * "are there security alerts detected?""

* Alert: the outcome of a check.
  An alert is a self-documented object: its description gives a clear idea of what's going on.
  Alerts have a severity. SeverityNone means that the alert may be safely ignored.
  The alert suggests a fix/mitigation action.

  Examples (git), from alerts raised by the above checks:
  * "these local branches are lagging behind their remote counterpart" -> suggestion: "update local branch"
  * "these local branches don't have any remote counterpart" -> suggestion: "push branch to upstream"

### Janitorial assignments

Suggested actions are scheduled as assignments.

In pure UX more (phase 1), an assignment is simply the synchronous execution of an action selected by the user
on the UX.

In semi-autonomous mode (phase 2), assignments are scheduled from actions taking into account current workload,
repo prioritization (e.g. most active repos processed first), smoothing git & github API interactions over time.

*Ideally*, we should be able to configure actions as really *chains-of-actions*, but it may be a bit cumbersome
to configure.

Example: rebase THEN push could be either a chain of 2 unitary actions, or a single combined action.
For phases 1 and 2 this distinction is not super-important. But for phase 3, it might come in handy to
prepare a chain-of-actions interleaving one or several external actions. Not sure yet on this topic.

---

## What we've built so far

git-janitor has a working POC with two solid pillars:
- **internal/git/** (5,140 LOC): comprehensive repo inspection API and actions
- **internal/ux/** (3,100 LOC): working TUI with tabs, filter, themes, config wizard

The missing piece: nothing connects the rich `git.RepoInfo` data to the Alerts, Actions, and Recent tabs.
They show hardcoded sample data. There's no evaluation layer, no action execution from the UI, no audit trail.

**Goal:** Build the orchestration layer that turns repo inspection data into actionable alerts,
lets the user execute suggested fixes, and tracks what happened.

### Layered Architecture

```
ux          → consumes alerts + actions, drives user interaction
engine      → orchestrates: config rules → check dispatch → alert collection → action execution
checks/     → registry of named checks (git, github, custom)
actions/    → registry of named actions (git, github, custom)
git         → CLI wrapper: inspection (RepoInfo) + mutating operations
github      → API wrapper: inspection (RepoData) + operations (future)
config      → YAML config: roots, rules (which checks + actions per root), defaults
store       → persistent state (Phase 2: KV store for alerts, assignments, collected data)
```

---

## Domain Model

### Pipeline

```
Check → Alert → ActionSuggestion → Assignment → Result → History
```

### Core Types (all live in `internal/engine/`)

```go
// SubjectKind categorizes what a check/action operates on.
type SubjectKind uint8

const (
    SubjectNone    SubjectKind = iota
    SubjectRepo
    SubjectRemote
    SubjectBranch
    SubjectStash
    SubjectTag
)

// Severity levels for alerts.
type Severity uint8

const (
    SeverityNone   Severity = iota  // check ran, nothing wrong (zero value)
    SeverityInfo                     // informational, no action needed
    SeverityLow                      // minor housekeeping
    SeverityMedium                   // should address soon
    SeverityHigh                     // needs attention now
)

// CheckKind identifies the provider of a check.
type CheckKind uint8

const (
    CheckKindGit    CheckKind = iota
    CheckKindGitHub
    CheckKindGitLab
    CheckKindCustom
)

// ActionKind mirrors CheckKind for actions.
type ActionKind uint8

const (
    ActionKindGit    ActionKind = iota
    ActionKindGitHub
    ActionKindCustom
)
```

### Alert

One alert per SubjectKind per check invocation. If a check finds 5 lagging branches,
that's 1 alert with 5 subject instances. The UX shows one row; the user drills in
to select/deselect individual subjects before executing.

```go
type Alert struct {
    CheckName   string              // which check produced this
    Severity    Severity            // SeverityNone = "check passed, nothing wrong"
    Summary     string              // one-line description
    Detail      string              // longer explanation (useful for custom/AI checks)
    Suggestions []ActionSuggestion  // zero or more suggested fixes
}

type ActionSuggestion struct {
    ActionName  string      // key in ActionRegistry
    SubjectKind SubjectKind // what kind of thing: branch, tag, repo...
    Subjects    []string    // specific instances: ["feature/old-1", "feature/old-2"]
}
```

Alert zero value (`Severity == SeverityNone`) means "check ran, nothing wrong."
The Alerts tab filters these out by default.

### Check Interface

Composition pattern: base `Check` interface for registry/config, typed concrete structs per provider.

```go
type SelfDescribed interface {
    Name() string
    Description() string
}

type Check interface {
    isCheck()               // sealed marker
    SelfDescribed
    Kind() CheckKind
}

// Base struct for embedding (provides Name/Description).
type describer struct {
    name        string
    description string
}

func (d describer) Name() string        { return d.name }
func (d describer) Description() string { return d.description }
```

Provider-specific concrete types with typed Evaluate methods:

```go
// GitCheck is the base for all git checks.
type GitCheck struct {
    describer
}

func (GitCheck) isCheck()            {}
func (GitCheck) Kind() CheckKind     { return CheckKindGit }
func (GitCheck) Evaluate(_ *git.RepoInfo) (iter.Seq[Alert], error) {
    return nil, errors.New("not implemented")
}

// GitHubCheck is the base for all GitHub checks.
type GitHubCheck struct {
    describer
}

func (GitHubCheck) isCheck()            {}
func (GitHubCheck) Kind() CheckKind     { return CheckKindGitHub }
func (GitHubCheck) Evaluate(_ *github.RepoData) (iter.Seq[Alert], error) {
    return nil, errors.New("not implemented")
}
```

Concrete checks embed the base and override Evaluate:

```go
type GitLocalBranchLagging struct {
    GitCheck
}

func (c GitLocalBranchLagging) Evaluate(info *git.RepoInfo) (iter.Seq[Alert], error) {
    // inspect info.Branches[].Behind, produce alert with subjects
    ...
}
```

### Action Interface

Mirrors the check pattern:

```go
type Action interface {
    isAction()              // sealed marker
    SelfDescribed
    Kind() ActionKind
    ApplyTo() SubjectKind   // what kind of subject this action operates on
    Destructive() bool      // needs user confirmation in Phase 1
}

type GitAction struct {
    describer
}

func (GitAction) isAction()              {}
func (GitAction) Kind() ActionKind       { return ActionKindGit }
func (GitAction) Destructive() bool      { return false }
func (GitAction) ApplyTo() SubjectKind   { return SubjectNone }
func (GitAction) Execute(_ context.Context, _ *git.Runner, _ *git.RepoInfo, _ []string) (Result, error) {
    return Result{}, errors.New("not implemented")
}

type Result struct {
    OK      bool
    Message string
}
```

Concrete actions embed the base and override:

```go
type GitActionUpdateBranch struct {
    GitAction
}

func (GitActionUpdateBranch) ApplyTo() SubjectKind { return SubjectBranch }
func (a GitActionUpdateBranch) Execute(ctx context.Context, r *git.Runner, info *git.RepoInfo, subjects []string) (Result, error) {
    // subjects = branch names to update
    ...
}
```

### Registries

Single flat registry per concern (checks, actions). Uniqueness enforced by map key.

```go
type CheckRegistry struct {
    checks map[string]Check  // keyed by Name()
}

func (r *CheckRegistry) Register(c Check)
func (r *CheckRegistry) Get(name string) (Check, bool)
func (r *CheckRegistry) All() iter.Seq2[string, Check]

type ActionRegistry struct {
    actions map[string]Action  // keyed by Name()
}

func (r *ActionRegistry) Register(a Action)
func (r *ActionRegistry) Get(name string) (Action, bool)
func (r *ActionRegistry) All() iter.Seq2[string, Action]
```

### Engine (stub for Phase 1, full scheduler for Phase 2)

The engine is the orchestrator. It owns the Evaluate type-switch dispatcher and
performs sanity checks (action SubjectKind must match suggestion SubjectKind).

```go
type Engine struct {
    Checks  *CheckRegistry
    Actions *ActionRegistry
    History *History
}

func New() *Engine  // registers all built-in checks and actions

// Evaluate dispatches to the right Evaluate method based on check Kind.
// The type-switch lives here, not in the checks package.
func (e *Engine) Evaluate(check Check, input RepoInfo) (iter.Seq[Alert], error)

// EvaluateRepo runs all enabled checks for a repo, collects alerts.
// enabledChecks comes from config rules.
func (e *Engine) EvaluateRepo(ctx context.Context, info *git.RepoInfo, enabledChecks []string) []Alert

// Execute runs an action, validating SubjectKind match.
func (e *Engine) Execute(ctx context.Context, runner *git.Runner, info *git.RepoInfo, suggestion ActionSuggestion) (Result, error)
```

`RepoInfo` is an interface so the dispatcher can accept both `*git.RepoInfo` and `*github.RepoData`:

```go
type RepoInfo interface {
    IsRepoInfo()  // exported marker (concrete types live in other packages)
}
```

### Assignment (Phase 1: thin wrapper, Phase 2: adds scheduling)

```go
type Assignment struct {
    Suggestion  ActionSuggestion
    RepoPath    string
    // Phase 2 fields:
    // State      AssignmentState  // pending, running, done, failed
    // Priority   int
    // ScheduledAt time.Time
}
```

### History (in-memory, Phase 2: persisted to KV store)

```go
type HistoryEntry struct {
    Timestamp  time.Time
    RepoPath   string
    ActionName string
    Subjects   []string
    Result     Result
}

type History struct {
    entries []HistoryEntry
    max     int  // ring buffer capacity
}

func (h *History) Append(entry HistoryEntry)
func (h *History) Entries() []HistoryEntry  // newest first
```

---

## Configuration Design

Single YAML file, new `rules` section alongside existing `roots` and `defaults`.

```yaml
defaults:
  schedule_interval: 5m
  rules:
    checks:
      - name: branch-merged-not-deleted
      - name: branch-gone-upstream
      - name: branch-lagging
      - name: health-gc-advised
      - name: activity-stale
        params:
          threshold_days: 180
    actions:
      - name: delete-branch
        auto: false     # requires user confirmation
      - name: update-branch
        auto: true      # can be auto-executed
      - name: run-gc
        auto: true

roots:
  - path: /home/fred/src/github.com/go-openapi
    name: go-openapi
    rules:
      checks:
        disable:
          - config-unsigned     # don't care about signing here
  - path: /home/fred/src/github.com/personal
    name: personal
```

Per-root overrides: `disable` list removes checks from the defaults.
Per-repo overrides: deferred (per-root is sufficient for now as a "repo group").
The config wizard will show available checks/actions from the registry with descriptions.

---

## Checks Catalog (git, Phase 1)

| Check Name | Evaluates | SubjectKind | Severity | Suggested Action |
|---|---|---|---|---|
| `branch-merged-not-deleted` | `Branches[].Merged && !IsRemote` | Branch | medium | `delete-branch` |
| `branch-gone-upstream` | `Branches[].Gone` | Branch | medium | `delete-branch` |
| `branch-lagging` | `Branches[].Behind > 0` | Branch | low | `update-branch` |
| `branch-no-upstream` | `Branches[].Upstream == ""` (local only) | Branch | low | — |
| `branch-diverged` | `Ahead > 0 && Behind > 0` | Branch | medium | `rebase-branch` |
| `dirty-worktree` | `Status.IsDirty()` | Repo | high | — (user must act) |
| `health-fsck-errors` | `Health.FSCKErrors` | Repo | high | — (manual fix) |
| `health-gc-advised` | `Health.GCAdvised` | Repo | low | `run-gc` |
| `size-repack-advised` | `Size.RepackAdvised` | Repo | low | `run-gc-aggressive` |
| `activity-stale` | `Activity.Staleness == "stale"` | Repo | low | — |
| `activity-dormant` | `Activity.Staleness == "dormant"` | Repo | medium | — |
| `config-no-email` | `Config.UserEmail.Value == ""` | Repo | medium | — |
| `config-unsigned` | `Config.CommitSign.Value != "true"` | Repo | info | — |
| `tags-local-only` | `Tags[].LocalOnly` | Tag | low | — |
| `tags-remote-only` | `Tags[].RemoteOnly` | Tag | low | — |
| `filestats-large-files` | `FileStats.LargeFiles` | Repo | low | — |
| `filestats-binary` | `FileStats.BinaryFiles` | Repo | info | — |
| `traits-shallow` | `IsShallow` | Repo | info | — |
| `traits-submodules` | `HasSubmodules` | Repo | info | — |
| `traits-lfs` | `HasLFS` | Repo | info | — |

---

## Actions Catalog (git, Phase 1)

| Action Name | ApplyTo | Destructive | git.Runner method |
|---|---|---|---|
| `delete-branch` | Branch | yes | `Runner.DeleteBranch` (new) |
| `update-branch` | Branch | no | `Runner.UpdateBranch` |
| `rebase-branch` | Branch | no | `Runner.RebaseBranch` |
| `merge-into` | Branch | no | `Runner.MergeInto` |
| `run-gc` | Repo | no | `Runner.Compact` |
| `run-gc-aggressive` | Repo | yes | `Runner.CompactAggressive` |

---

## Package Layout

```
internal/
  engine/                 ← shared domain types + orchestrator
    doc.go
    types.go              ← Severity, SubjectKind, CheckKind, ActionKind, Alert,
                             ActionSuggestion, Result, Assignment, RepoInfo interface,
                             SelfDescribed, Check interface, Action interface
    check_git.go          ← GitCheck base struct
    check_github.go       ← GitHubCheck base struct (stub for now)
    action_git.go         ← GitAction base struct
    action_github.go      ← GitHubAction base struct (stub for now)
    registry.go           ← CheckRegistry, ActionRegistry
    engine.go             ← Engine struct, Evaluate dispatcher, EvaluateRepo, Execute
    history.go            ← HistoryEntry, History ring buffer

  checks/                 ← concrete check implementations
    git/                  ← built-in git checks
      branches.go         ← GitLocalBranchLagging, GitBranchMergedNotDeleted, etc.
      health.go           ← GitHealthFSCK, GitHealthGCAdvised
      size.go             ← GitSizeRepackAdvised
      activity.go         ← GitActivityStale, GitActivityDormant
      config.go           ← GitConfigNoEmail, GitConfigUnsigned
      tags.go             ← GitTagsLocalOnly, GitTagsRemoteOnly
      filestats.go        ← GitLargeFiles, GitBinaryFiles
      traits.go           ← GitShallow, GitSubmodules, GitLFS
      register.go         ← RegisterAll(registry) — registers all git checks
    github/               ← slot for future GitHub checks
      register.go

  actions/                ← concrete action implementations
    git/                  ← built-in git actions
      branch.go           ← GitActionDeleteBranch, GitActionUpdateBranch, GitActionRebaseBranch
      maintenance.go      ← GitActionRunGC, GitActionRunGCAggressive
      register.go         ← RegisterAll(registry) — registers all git actions
    github/               ← slot for future GitHub actions
      register.go

  git/                    ← unchanged, the git CLI wrapper
  config/                 ← gains rules section in Config struct
  store/                  ← Phase 2: KV persistence (bolt/badger)
  ux/                     ← consumes engine.Alert + engine.ActionSuggestion
```

---

## Implementation Plan (within product Phase 1)

### Step 1: Engine types + registries (no UX, no checks yet)

| File | Content |
|------|---------|
| `internal/engine/doc.go` | Package documentation |
| `internal/engine/types.go` | All shared types: Severity, SubjectKind, CheckKind, ActionKind, Alert, ActionSuggestion, Result, Assignment, RepoInfo interface, SelfDescribed, Check, Action |
| `internal/engine/check_git.go` | GitCheck base struct with default Evaluate |
| `internal/engine/check_github.go` | GitHubCheck base struct (stub) |
| `internal/engine/action_git.go` | GitAction base struct with default Execute |
| `internal/engine/action_github.go` | GitHubAction base struct (stub) |
| `internal/engine/registry.go` | CheckRegistry + ActionRegistry |
| `internal/engine/engine.go` | Engine struct, Evaluate dispatcher, EvaluateRepo, Execute with SubjectKind validation |
| `internal/engine/history.go` | HistoryEntry + History ring buffer |
| `internal/engine/engine_test.go` | Tests: registry, dispatch, SubjectKind validation |

**Depends on:** `internal/git` types only (for `git.RepoInfo`).

### Step 2: Built-in git checks

| File | Content |
|------|---------|
| `internal/checks/git/branches.go` | 5 branch checks |
| `internal/checks/git/health.go` | fsck + gc-advised |
| `internal/checks/git/size.go` | repack-advised |
| `internal/checks/git/activity.go` | stale + dormant |
| `internal/checks/git/config.go` | no-email + unsigned |
| `internal/checks/git/tags.go` | local-only + remote-only |
| `internal/checks/git/filestats.go` | large files + binary |
| `internal/checks/git/traits.go` | shallow + submodules + LFS |
| `internal/checks/git/register.go` | RegisterAll(registry) |
| `internal/checks/git/*_test.go` | Tests with synthetic RepoInfo values |

**Depends on:** Step 1.

### Step 3: Built-in git actions

| File | Content |
|------|---------|
| `internal/actions/git/branch.go` | delete, update, rebase, merge-into |
| `internal/actions/git/maintenance.go` | gc, gc-aggressive |
| `internal/actions/git/register.go` | RegisterAll(registry) |
| `internal/git/actions.go` | Add `Runner.DeleteBranch()` |
| `internal/git/git_commands.go` | Add `cmdDeleteBranch()` |

**Depends on:** Step 1.

### Step 4: Configuration

| File | Content |
|------|---------|
| `internal/config/config.go` | Add Rules section (checks + actions config, per-root overrides) |

**Depends on:** Step 1 (needs check/action names).

### Step 5: Wire alerts into UX

| File | Change |
|------|--------|
| `internal/ux/panels/infos/infos.go` | Add `*engine.Engine` to Panel; call `EvaluateRepo` in `SetRepoInfo()`; push alerts to Alerts panel |
| `internal/ux/panels/infos/tab-alerts/alerts.go` | Add `SetAlerts([]engine.Alert)`; remove hardcoded data; map engine.Severity → UX display |

**Depends on:** Steps 2, 4.

### Step 6: Wire actions into UX

| File | Change |
|------|--------|
| `internal/ux/types/types.go` | Add `ExecuteActionMsg`, `ActionResultMsg` |
| `internal/ux/panels/infos/tab-actions/actions.go` | `SetSuggestions()` from alerts; Enter → `ExecuteActionMsg`; subject selection UI; remove hardcoded data |
| `internal/ux/panels/infos/infos.go` | Push suggestions to Actions panel |
| `internal/ux/model.go` | Handle `ExecuteActionMsg` (background tea.Cmd); handle `ActionResultMsg` (status bar + refresh) |

**Depends on:** Steps 3, 5.

### Step 7: Wire recent activity

| File | Change |
|------|--------|
| `internal/ux/panels/infos/tab-recent/recent.go` | `SetHistory()`; remove hardcoded data |
| `internal/ux/model.go` | Hold `*engine.History`; append on `ActionResultMsg`; push to Recent panel |

**Depends on:** Step 6.

### Step 8: Facts tab enrichment (lower priority)

| File | Change |
|------|--------|
| `internal/ux/panels/infos/tab-facts/facts.go` | Display Activity, Health, Size, Tags, FileStats, Config data from RepoInfo |

**Independent:** Can be done anytime.

### Step 9: Cleanup

| File | Change |
|------|--------|
| `internal/alerts/` | Remove entire package (superseded by engine + checks) |
| `.claude/CLAUDE.md` | Update package table |

---

## Deferred (product Phases 2-4)

- **Persistence** (`internal/store/`): KV store (bolt/badger) for History, alerts, assignments.
  Transition from in-memory should be straightforward (same data structures).
- **Background scheduler** (engine becomes full-fledged): prioritized task queue,
  workload-aware scheduling, rate limiting for API calls.
- **GitHub checks and actions** (`checks/github/`, `actions/github/`): PR status, CI, issues,
  security alerts. Using native Go SDK (google/go-github or similar).
- **Custom/external checks and actions** (Phase 3): shell command execution, stdin/stdout protocol
  for passing alert context to external tools (AI agents, scripts).
- **Confirmation dialogs**: modal popup for destructive actions before execution.
- **Batch subject selection**: select/deselect individual subjects within a suggestion.
- **Config wizard for rules**: interactive setup of which checks/actions are enabled per root.
- **Chain-of-actions**: action composition (rebase THEN push). Data model doesn't prevent it.
- **GitLab support**: parallel to GitHub, for workplace use.

---

## Verification

After each step:

1. `go build ./...` — clean build
2. `go test ./...` — all tests pass
3. `golangci-lint run` — no new lint issues

Milestone checks:
- After Step 2: unit tests pass with synthetic `git.RepoInfo` (no git binary needed)
- After Step 5: manual TUI test — select a repo, Alerts tab shows real alerts
- After Step 6: manual TUI test — Actions tab shows suggestions, Enter executes, status bar updates
- After Step 7: manual TUI test — Recent tab shows action history

---

## Critical Files

- `internal/git/repo_info.go` — the `RepoInfo` struct that checks consume
- `internal/engine/types.go` — all shared domain types
- `internal/engine/engine.go` — the Evaluate dispatcher + Execute with validation
- `internal/engine/registry.go` — check and action registries
- `internal/ux/panels/infos/infos.go:85` — `SetRepoInfo()`, the UX integration point
- `internal/ux/panels/infos/tab-alerts/alerts.go` — hardcoded data to replace
- `internal/ux/panels/infos/tab-actions/actions.go` — hardcoded data to replace
- `internal/ux/panels/infos/tab-recent/recent.go` — hardcoded data to replace
- `internal/ux/model.go` — message routing for action execution
- `internal/config/config.go` — rules configuration section

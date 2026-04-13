# GitLab Provider Plan

> Last revision: 2026-04-13
>
> Status: plan finalized, implementation not started

## Overview

Add a GitLab provider with the same structure as the existing GitHub provider.
The existing codebase already anticipates this: `CheckKindGitLab`, `SCMGitLab`,
`gitlabHostRe` in SCM detection, and `PlatformInfo` documented as covering
"GitHub, GitLab, Gitea, etc."

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Base URL resolution | Auto-derive from remote URL, with optional config override | Most enterprise instances work with `https://<host>` derived from the remote. Config override for edge cases (API on different port/path). |
| Subject naming for MRs/Pipelines | Reuse `SubjectPullRequests` / `SubjectWorkflowRuns` | Simpler — no enum changes, TUI adjusts display labels based on `info.SCM`. |
| PlatformInfo additions | Add `ProjectID int` and `WebURL string` to shared struct | `ProjectID` is GitLab-specific (zero for GitHub). `WebURL` stores the instance base URL, useful for both providers. |
| GitLab SDK | `gitlab.com/gitlab-org/api/client-go` (official successor to `xanzy/go-gitlab`) | Used in the reference example code. Current version: v0.137.0. |
| Token env vars | `GITLAB_TOKEN` (primary), `GL_TOKEN` (fallback) | Mirrors GitHub convention. `glab` CLI uses `GITLAB_TOKEN`. |
| Enterprise/self-hosted | Fully supported via auto-derived or configured base URL | Host regex `gitlab\.(\w+)` already detects most instances. For hosts without "gitlab" in the name, per-root `scm` config could be added later. |

---

## Phase 1 — Core Kernel

### 1A: Enum & Model Changes

**Modify `internal/models/enums.go`:**

- Add `RunnerKindGitLab` after `RunnerKindGitHub` (value = 2)
- Insert `ActionKindGitLab` between `ActionKindGitHub` and `ActionKindCustom`
  (currently `ActionKindCustom` = 2; becomes 3)

**Modify `internal/models/platform.go`:**

- Add `ProjectID int` in the Identity section (GitLab integer ID; zero for GitHub)
- Add `WebURL string` (base URL of the instance, e.g. `https://gitlab.ca.cib`)

### 1B: Config Changes

**Modify `internal/config/config.go`:**

Add new type:
```go
type GitLabConfig struct {
    Enabled        bool
    BaseURL        string `mapstructure:"baseURL,omitempty"`
    SecurityAlerts *bool  `mapstructure:"securityAlerts,omitempty"`
}
```

Add `GitLab GitLabConfig` to `Config` struct.
Add `GitLab *GitLabConfig` to `RootConfig` (per-root override).

Add helper methods:
- `GitLabEnabled(rootIndex int) bool`
- `GitLabSecurityAlerts(rootIndex int) bool`
- `GitLabBaseURL(rootIndex int) string`

**Modify `internal/config/default_config.yaml`:**

Add after `github:` section:
```yaml
gitlab:
  enabled: true
  securityAlerts: false  # requires GitLab Ultimate, off by default
```

Add GitLab checks and actions to the `defaults.rules` lists.

### 1C: `internal/gitlab/backend/` — New Package

| File | Purpose |
|------|---------|
| `gitlab_runner.go` | `Client` + `Runner` structs. Token from `GITLAB_TOKEN` / `GL_TOKEN`. `NewClient(baseURL string)` using `gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))`. `Available() bool`. Rate-limit tracking. |
| `collect.go` | `collectRepoInfo(ctx, client, projectPath, fetchSecurity) *PlatformInfo`. Maps `Projects.GetProject()` response to `PlatformInfo`. Checks `ProtectedBranches.ListProtectedBranches` for default branch protection. |
| `cache.go` | In-memory TTL cache keyed by project path. Same pattern as GitHub cache. |
| `parse.go` | `ExtractProjectPath(rawURL) (string, error)` — handles nested GitLab paths (`group/subgroup/project`). `ExtractBaseURL(rawURL) (string, error)` — extracts `https://hostname` from remote URL. |
| `parse_test.go` | Table-driven tests for SSH, HTTPS, `ssh://` URLs with nested paths. |
| `issues.go` | `ListIssues`, `GetIssueDetail`. Maps `gitlab.Issue` → `models.Issue`. |
| `merge_requests.go` | `ListMergeRequests`, `GetMergeRequestDetail`. Maps `gitlab.BasicMergeRequest` → `models.PullRequest`. |
| `pipelines.go` | `ListPipelines`, `GetPipelineDetail`. Maps `gitlab.PipelineInfo` → `models.WorkflowRun`. |

**Key difference from GitHub**: GitLab project paths are N-segment (`group/subgroup/project`),
not 2-segment (`owner/repo`). The SDK accepts URL-encoded paths or integer IDs.

**Field mapping: `Projects.GetProject()` → `PlatformInfo`:**

| GitLab SDK field | PlatformInfo field |
|---|---|
| `project.ID` | `ProjectID` |
| `project.PathWithNamespace` | `FullName` |
| `project.Namespace.FullPath` | `Owner` |
| `project.Path` | `Repo` |
| `project.WebURL` | `HTMLURL` |
| `project.Description` | `Description` |
| `project.DefaultBranch` | `DefaultBranch` |
| `project.Topics` | `Topics` |
| `project.ForkedFromProject != nil` | `IsFork`, `ParentFullName` |
| `project.Archived` | `IsArchived` |
| `project.Visibility == "private"` | `IsPrivate` |
| `project.Permissions` (>= Developer/30) | `HasPushAccess` |
| `project.Permissions` (>= Maintainer/40) | `HasAdminAccess` |
| `project.OpenIssuesCount` | `OpenIssues` |
| MR list per_page=1, read X-Total header | `OpenPRs` |
| `project.StarCount` | `StarCount` |
| `project.ForksCount` | `ForkCount` |
| `project.CreatedAt` | `CreatedAt` |
| `project.LastActivityAt` | `UpdatedAt` |
| `project.RemoveSourceBranchAfterMerge` | `DeleteBranchOnMerge` (1/0) |
| Protected branches check | `DefaultBranchProtected` (1/0) |
| N/A | `ActionsEnabled` = -1 (no equivalent) |
| Derived from remote URL | `WebURL` |

### 1D: `internal/gitlab/checks/` — New Package (7 checks)

All checks embed a `gitlabCheck` base struct (same pattern as `githubCheck`):
returns `CheckKindGitLab`, embeds `models.Describer`.

| File | Check | Name | Severity | Logic |
|------|-------|------|----------|-------|
| `check.go` | `gitlabCheck` base | — | — | Base struct |
| `repo.go` | `RepoArchived` | `gitlab-repo-archived` | Medium | `data.IsArchived` |
| `repo.go` | `DescriptionMissing` | `gitlab-description-missing` | Low | `data.Description == ""` |
| `repo.go` | `VisibilityPrivate` | `gitlab-visibility-private` | Info | `data.IsPrivate` |
| `repo.go` | `RepoForkParent` | `gitlab-repo-fork-parent` | Info | `data.IsFork` |
| `branches.go` | `DefaultBranchMismatch` | `gitlab-default-branch-mismatch` | Low | `data.DefaultBranch != data.LocalDefaultBranch` |
| `branch_protection.go` | `BranchProtectionMissing` | `gitlab-branch-protection-missing` | Medium | `data.DefaultBranchProtected == 0` |
| `delete_branch_on_merge.go` | `DeleteBranchOnMergeMissing` | `gitlab-delete-branch-on-merge` | Low | `data.DeleteBranchOnMerge == 0` |
| `helpers.go` | Shared alert builders | — | — | — |

Test files mirror GitHub pattern: call internal `evaluate(*PlatformInfo)` directly.

### 1E: `internal/gitlab/actions/` — New Package (4 actions)

| File | Action | Name | Subject | Logic |
|------|--------|------|---------|-------|
| `action.go` | `gitlabAction` base | — | — | Returns `ActionKindGitLab` |
| `context.go` | Runner extraction | — | — | `runnerFromContext(ctx)` |
| `repo.go` | `SetProjectDescription` | `gitlab-set-project-description` | SubjectRepo | `Projects.EditProject(id, ...)` |
| `browser.go` | `OpenInBrowser` | `gitlab-open-in-browser` | SubjectRepo | Opens `HTMLURL` |
| `branch_protection.go` | `EnableBranchProtection` | `gitlab-enable-branch-protection` | SubjectRepo | `ProtectedBranches.ProtectRepositoryBranches()` |
| `delete_branch_on_merge.go` | `EnableDeleteBranchOnMerge` | `gitlab-enable-delete-branch-on-merge` | SubjectRepo | `Projects.EditProject(id, &EditProjectOptions{RemoveSourceBranchAfterMerge: Ptr(true)})` |

### 1F: `internal/gitlab/` — Registration

| File | Contents |
|------|----------|
| `all_checks.go` | `func AllChecks() iter.Seq[ifaces.Check]` — yields all 7 checks |
| `all_actions.go` | `func AllActions() iter.Seq[ifaces.Action]` — yields all 4 actions |

### 1G: Engine Integration

**Modify `internal/engine/engine_interactive.go`:**

1. Import `gitlabbackend "github.com/fredbi/git-janitor/internal/gitlab/backend"`

2. Add field to `Interactive`:
   ```go
   gitlabClients map[string]*gitlabbackend.Client // keyed by base URL
   ```
   (Map because we may talk to multiple GitLab instances.)

3. `Evaluate()` — add `CheckKindGitLab` to skip condition:
   ```go
   if check.Kind() == models.CheckKindGitLab && info.Platform == nil && info.UpstreamPlatform == nil {
       continue
   }
   ```

4. `collectPlatform()` — add `case models.SCMGitLab:` branch:
   - Check `e.cfg.GitLabEnabled(info.RootIndex)`
   - Extract project path via `gitlabbackend.ExtractProjectPath(originURL)`
   - Auto-derive base URL via `gitlabbackend.ExtractBaseURL(originURL)`, override with `e.cfg.GitLabBaseURL(info.RootIndex)` if non-empty
   - Call `e.getGitLabClient(baseURL).Fetch(ctx, projectPath, opts)`
   - Populate `info.Platform`

5. `ProviderEnabled()` — add `case "gitlab":` checking config + token env vars.

6. `withRunnerForAction()` — add `case models.ActionKindGitLab:` creating a `gitlabbackend.Runner`.

7. Add `getGitLabClient(baseURL string) *gitlabbackend.Client` — lazy-init, keyed by base URL.

**Modify `internal/engine/activity.go`:**

Add SCM dispatch: when `info.SCM == models.SCMGitLab`, route `SubjectPullRequests`
to `collectMergeRequestList()` and `SubjectWorkflowRuns` to `collectPipelineList()`.
Phase 1 can stub these out; Phase 2 provides the real implementation.

### 1H: Registration & Dependency

**Modify `cmd/git-janitor/main.go`:**
```go
import "github.com/fredbi/git-janitor/internal/gitlab"

checks := registry.New[ifaces.Check](
    registry.With(git.AllChecks(), github.AllChecks(), gitlab.AllChecks()),
)
actions := registry.New[ifaces.Action](
    registry.With(git.AllActions(), github.AllActions(), gitlab.AllActions()),
)
```

**Modify `go.mod`:**
```
require gitlab.com/gitlab-org/api/client-go v0.137.0
```

---

## Phase 2 — Activity & Security (deferred)

### 2A: Activity — MR listing, Pipeline listing, Issue listing
Implement `ListIssues`, `ListMergeRequests`, `ListPipelines` in the backend package.

### 2B: Engine activity dispatch
Wire `collectMergeRequestList()`, `collectPipelineList()`, `collectIssueList()` for GitLab
in `internal/engine/activity.go`.

### 2C: Security checks
- `gitlab-security-alerts` — query `ProjectVulnerabilities.ListProjectVulnerabilities()` (requires GitLab Ultimate)
- `gitlab-security-not-enabled` — check if vulnerability scanning is configured
- Map vulnerability counts to `DependabotAlerts` / `CodeScanningAlerts` / `SecretScanningAlerts` or add GitLab-specific fields

---

## GitHub ↔ GitLab Feature Mapping

| GitHub Feature | GitLab Equivalent | Status |
|---|---|---|
| `Repositories.Get` → repo metadata | `Projects.GetProject` | Phase 1 |
| Open issue count | `Project.OpenIssuesCount` (native, no PRs mixed in) | Phase 1 |
| Open PR count (per_page=1 trick) | MR list per_page=1, read `X-Total` header | Phase 1 |
| Branch protection | `ProtectedBranches.ListProtectedBranches` | Phase 1 |
| Delete branch on merge | `Project.RemoveSourceBranchAfterMerge` | Phase 1 |
| Fork parent | `Project.ForkedFromProject` | Phase 1 |
| Permissions (admin/push) | `Project.Permissions` access levels | Phase 1 |
| Set description | `Projects.EditProject` | Phase 1 |
| Enable branch protection | `ProtectedBranches.ProtectRepositoryBranches` | Phase 1 |
| Dependabot/code scanning/secret scanning | Vulnerability API (GitLab Ultimate) | Phase 2 |
| Issues list | `Issues.ListProjectIssues` | Phase 2 |
| Pull requests list | `MergeRequests.ListProjectMergeRequests` | Phase 2 |
| Workflow runs | `Pipelines.ListProjectPipelines` | Phase 2 |
| Actions enabled (fork toggle) | No equivalent (CI is file-driven) | N/A |

---

## File Summary

### New files (~25)

| Path | Phase |
|------|-------|
| `internal/gitlab/backend/gitlab_runner.go` | 1C |
| `internal/gitlab/backend/collect.go` | 1C |
| `internal/gitlab/backend/cache.go` | 1C |
| `internal/gitlab/backend/parse.go` | 1C |
| `internal/gitlab/backend/parse_test.go` | 1C |
| `internal/gitlab/backend/issues.go` | 1C (stub) / 2A (impl) |
| `internal/gitlab/backend/merge_requests.go` | 1C (stub) / 2A (impl) |
| `internal/gitlab/backend/pipelines.go` | 1C (stub) / 2A (impl) |
| `internal/gitlab/checks/check.go` | 1D |
| `internal/gitlab/checks/repo.go` | 1D |
| `internal/gitlab/checks/repo_test.go` | 1D |
| `internal/gitlab/checks/branches.go` | 1D |
| `internal/gitlab/checks/branches_test.go` | 1D |
| `internal/gitlab/checks/branch_protection.go` | 1D |
| `internal/gitlab/checks/branch_protection_test.go` | 1D |
| `internal/gitlab/checks/delete_branch_on_merge.go` | 1D |
| `internal/gitlab/checks/delete_branch_on_merge_test.go` | 1D |
| `internal/gitlab/checks/helpers.go` | 1D |
| `internal/gitlab/actions/action.go` | 1E |
| `internal/gitlab/actions/context.go` | 1E |
| `internal/gitlab/actions/repo.go` | 1E |
| `internal/gitlab/actions/browser.go` | 1E |
| `internal/gitlab/actions/branch_protection.go` | 1E |
| `internal/gitlab/actions/delete_branch_on_merge.go` | 1E |
| `internal/gitlab/all_checks.go` | 1F |
| `internal/gitlab/all_actions.go` | 1F |

### Modified files (~7)

| Path | Phase | Change |
|------|-------|--------|
| `internal/models/enums.go` | 1A | `RunnerKindGitLab`, `ActionKindGitLab` |
| `internal/models/platform.go` | 1A | `ProjectID`, `WebURL` fields |
| `internal/config/config.go` | 1B | `GitLabConfig`, helpers |
| `internal/config/default_config.yaml` | 1B | `gitlab:` section, checks, actions |
| `internal/engine/engine_interactive.go` | 1G | GitLab branches in collectPlatform, ProviderEnabled, withRunnerForAction |
| `internal/engine/activity.go` | 1G | SCM dispatch for activity collection |
| `cmd/git-janitor/main.go` | 1H | Register `gitlab.AllChecks()`, `gitlab.AllActions()` |
| `go.mod` | 1H | Add `gitlab.com/gitlab-org/api/client-go` |

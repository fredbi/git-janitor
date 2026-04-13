# GitHub vs GitLab: API & Architecture Differences

> Last revision: 2026-04-13
>
> Reference analysis for the GitLab provider implementation.
> Covers SDK differences, API model differences, and architectural
> implications for git-janitor.

## SDK & Client

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| SDK package | `github.com/google/go-github/v72/github` | `gitlab.com/gitlab-org/api/client-go` (successor to `xanzy/go-gitlab`) |
| Client creation | `gogithub.NewClient(nil).WithAuthToken(token)` | `gitlab.NewClient(token, gitlab.WithBaseURL(url))` |
| Base URL | Hardcoded `https://api.github.com` — no config needed | **Must be configured** per-instance. Self-hosted is the norm, not the exception. |
| Auth token env var | `GITHUB_TOKEN` / `GH_TOKEN` | `GITLAB_TOKEN` / `GL_TOKEN` (convention from `glab` CLI) |
| OAuth2 flow | Not used in git-janitor | SDK has built-in `gitlaboauth2` sub-package with browser callback flow. Not needed for git-janitor Phase 1. |
| Response type | `*github.Response` (embeds `*http.Response`) | `*gitlab.Response` (embeds `*http.Response`) |
| Pointer helpers | `github.String()`, `github.Bool()`, etc. (deprecated in favor of direct `&val`) | `gitlab.Ptr(val)` — generic helper |

## Authentication

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| Token type | Personal Access Token (classic or fine-grained) | Personal Access Token, Project Access Token, Group Access Token, Deploy Token |
| Token scope discovery | `X-OAuth-Scopes` response header | Not available via headers; scopes are defined at creation time |
| Scope validation | git-janitor reads `X-OAuth-Scopes` header to report token capabilities | Would need to call `PersonalAccessTokens.GetSinglePersonalAccessToken()` (requires `read_api` scope) or just try operations and handle 403 |

## Rate Limiting

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| Headers | `X-RateLimit-Remaining`, `X-RateLimit-Limit`, `X-RateLimit-Reset` | `RateLimit-Remaining`, `RateLimit-Limit`, `RateLimit-Reset` (different header names) |
| SDK exposure | `resp.Rate.Remaining`, `resp.Rate.Limit`, `resp.Rate.Reset.Time` | `resp.Header.Get("RateLimit-Remaining")` — must parse manually from response headers |
| Default limits | 5000/hour (authenticated) | Varies by instance and tier; typically 600/min for authenticated users on gitlab.com |
| Strategy in git-janitor | Caution at <100, hard stop at <20 | Same strategy can be applied; thresholds may need adjustment for per-minute windows |

## Project/Repository Identity

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| Identifier format | `owner/repo` (always 2 segments) | `group/subgroup/.../project` (N segments, arbitrarily nested) |
| API identifier | String: `owner`, `repo` as separate params | Integer `projectID` **or** URL-encoded path string `group%2Fsubgroup%2Fproject` |
| URL parsing | Strip host + `.git`, split on `/`, take first 2 segments | Strip host + `.git`, take **entire remaining path** |
| Owner concept | GitHub user or organization (1 segment) | GitLab namespace/group (can be nested: `group/subgroup`) |
| Integer ID | Not commonly used in API (string-based) | `project.ID` — integer, heavily used. More reliable than path (paths can be renamed). |

### Implication for git-janitor

The `ExtractOwnerRepo(url) (owner, repo string, error)` pattern doesn't work for GitLab.
Need `ExtractProjectPath(url) (string, error)` returning the full path. The `Owner` and
`Repo` fields in `PlatformInfo` can be populated from `project.Namespace.FullPath` and
`project.Path` respectively, but the API calls use the full path or integer ID.

## Repository Metadata

| Field | GitHub (`Repositories.Get`) | GitLab (`Projects.GetProject`) | Notes |
|-------|---------------------------|-------------------------------|-------|
| Name | `repo.GetName()` | `project.Name` | — |
| Full path | `repo.GetFullName()` ("owner/repo") | `project.PathWithNamespace` ("group/sub/proj") | Different segment depth |
| Web URL | `repo.GetHTMLURL()` | `project.WebURL` | — |
| Description | `repo.GetDescription()` | `project.Description` | — |
| Default branch | `repo.GetDefaultBranch()` | `project.DefaultBranch` | — |
| Topics/tags | `repo.Topics` ([]string) | `project.Topics` ([]string) | — |
| License | `repo.GetLicense().GetSPDXID()` | Not directly on project; need `Projects.GetProjectLicenses()` or check for LICENSE file | Extra API call or skip |
| Fork status | `repo.GetFork()` | `project.ForkedFromProject != nil` | — |
| Fork parent | `repo.GetParent().GetFullName()` | `project.ForkedFromProject.PathWithNamespace` | — |
| Archived | `repo.GetArchived()` | `project.Archived` | — |
| Visibility | `repo.GetPrivate()` (bool: public/private) | `project.Visibility` (string: "public"/"internal"/"private") | GitLab has 3 levels; "internal" is unique to GitLab |
| Permissions | `repo.GetPermissions()` map[string]bool (admin, push, pull) | `project.Permissions` with `ProjectAccess`/`GroupAccess` structs containing `AccessLevel` (int: 10-50) | Different model (see below) |
| Open issues | `repo.GetOpenIssuesCount()` (**includes PRs**) | `project.OpenIssuesCount` (**excludes MRs**) | GitHub lumps PRs into issue count; GitLab does not |
| Stars | `repo.GetStargazersCount()` | `project.StarCount` | — |
| Forks | `repo.GetForksCount()` | `project.ForksCount` | — |
| Created | `repo.GetCreatedAt().Time` | `*project.CreatedAt` | GitLab uses `*time.Time` pointers |
| Updated | `repo.GetUpdatedAt().Time` | `*project.LastActivityAt` | Different semantics: GitHub is metadata update, GitLab is any activity |
| Pushed | `repo.GetPushedAt().Time` | No direct equivalent | GitLab tracks `LastActivityAt` instead |
| Delete branch on merge | `repo.GetDeleteBranchOnMerge()` (repo-level setting) | `project.RemoveSourceBranchAfterMerge` (project-level default) | Semantically equivalent; GitLab also has per-MR override |

## Permissions Model

| GitHub | GitLab |
|--------|--------|
| Boolean map: `{admin: true, push: true, pull: true}` | Integer access levels: Guest=10, Reporter=20, Developer=30, Maintainer=40, Owner=50 |
| `perms["admin"]` → `HasAdminAccess` | `accessLevel >= 40` (Maintainer) → `HasAdminAccess` |
| `perms["push"]` → `HasPushAccess` | `accessLevel >= 30` (Developer) → `HasPushAccess` |

GitLab permissions can come from two sources:
- `project.Permissions.ProjectAccess` (direct project membership)
- `project.Permissions.GroupAccess` (inherited from group)

Must check both and take the higher access level.

## Branch Protection

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| Check if protected | `Repositories.GetBranchProtection(owner, repo, branch)` — 404 = not protected | `ProtectedBranches.ListProtectedBranches(projectID)` — check if default branch appears in list |
| Enable protection | `Repositories.UpdateBranchProtection(owner, repo, branch, request)` | `ProtectedBranches.ProtectRepositoryBranches(projectID, opts)` |
| Protection model | Enforcement rules: required reviews, status checks, enforce admins | Access levels: who can push, who can merge, who can unprotect. Default: Maintainers push, Developers+Maintainers merge. |
| Default protection | None — must be explicitly set | GitLab auto-protects the default branch at project creation (Maintainers can push, Developers can merge) |

### Implication

The "branch protection missing" check has different semantics:
- GitHub: 404 from the protection API = not protected → alert
- GitLab: default branch is typically already protected at creation; the check should verify
  it's in the protected list (rare to be missing, but possible if someone removed it)

## Pull Requests vs Merge Requests

| Aspect | GitHub (Pull Requests) | GitLab (Merge Requests) |
|--------|----------------------|------------------------|
| API service | `PullRequests` | `MergeRequests` |
| List | `PullRequests.List(owner, repo, opts)` | `MergeRequests.ListProjectMergeRequests(projectID, opts)` |
| Get detail | `PullRequests.Get(owner, repo, number)` | `MergeRequests.GetMergeRequest(projectID, mrIID, opts)` |
| Identifier | `number` (int, sequential per repo) | `iid` (int, sequential per project) — distinct from global `id` |
| State values | "open", "closed" | "opened", "closed", "merged", "locked" |
| Draft detection | `pr.GetDraft()` (bool) | `mr.Draft` (bool) or title prefix `Draft:` / `WIP:` |
| Mergeable | `pr.GetMergeable()` (nullable bool, requires detail call) | `mr.MergeStatus` (string: "can_be_merged", "cannot_be_merged", etc.) |
| Review state | `PullRequestReviews` service (separate API) | Approval rules on the MR itself; `mr.Approvals` or `MergeRequestApprovals` service |
| Diff stats | `pr.GetAdditions()`, `pr.GetDeletions()`, `pr.GetChangedFiles()` | `mr.Changes` (requires `GetMergeRequest` with `with_stats` option) |

### Mapping to `models.PullRequest`

The existing `models.PullRequest` struct works for both:
- `Number` → GitHub PR number / GitLab MR IID
- `Title`, `State`, `Author`, `Branch`, `Base`, `Draft` — direct mapping
- `State`: normalize GitLab "opened" → "open", "merged" remains as-is

## Workflow Runs vs Pipelines

| Aspect | GitHub (Actions Workflow Runs) | GitLab (CI/CD Pipelines) |
|--------|-------------------------------|-------------------------|
| API service | `Actions` | `Pipelines` |
| List | `Actions.ListWorkflowRunsByRepo(owner, repo, opts)` | `Pipelines.ListProjectPipelines(projectID, opts)` |
| Get detail | `Actions.GetWorkflowRunByID(owner, repo, runID)` | `Pipelines.GetPipeline(projectID, pipelineID)` |
| Identifier | `run.GetID()` (int64) | `pipeline.ID` (int) |
| Name | `run.GetName()` (workflow name) | `pipeline.Ref` + source (no workflow name — pipelines are monolithic) |
| Status | `run.GetStatus()`: "queued", "in_progress", "completed" | `pipeline.Status`: "created", "waiting_for_resource", "preparing", "pending", "running", "success", "failed", "canceled", "skipped", "manual", "scheduled" |
| Conclusion | `run.GetConclusion()`: "success", "failure", "cancelled", etc. | Folded into `Status` — no separate conclusion field |
| Branch | `run.GetHeadBranch()` | `pipeline.Ref` |
| Event/trigger | `run.GetEvent()`: "push", "pull_request", etc. | `pipeline.Source`: "push", "web", "trigger", "schedule", "api", "merge_request_event", etc. |
| Duration | Computed: `UpdatedAt - CreatedAt` | `pipeline.Duration` (seconds, directly available) |
| Run number | `run.GetRunNumber()` | No direct equivalent — use `pipeline.ID` |
| Re-run attempts | `run.GetRunAttempt()` | `pipeline.RetryCount` or use `Pipelines.RetryPipelineBuild` |

### Mapping to `models.WorkflowRun`

- `ID` → pipeline ID
- `Name` → `pipeline.Ref` (branch name as proxy, or construct from source)
- `Status` → map GitLab statuses to a normalized set
- `Conclusion` → derive from status: "success"→"success", "failed"→"failure", "canceled"→"cancelled"
- `Branch` → `pipeline.Ref`
- `Event` → `pipeline.Source`
- `Duration` → directly available

## Issues

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| API service | `Issues` | `Issues` |
| List | `Issues.ListByRepo(owner, repo, opts)` | `Issues.ListProjectIssues(projectID, opts)` |
| Includes PRs? | **Yes** — GitHub issues include PRs (must filter with `pr == nil`) | **No** — issues and MRs are separate |
| Identifier | `issue.GetNumber()` | `issue.IID` (project-scoped) |
| State values | "open", "closed" | "opened", "closed" |
| Labels | `[]Label` with name + color | `gitlab.Labels` ([]string) — just names |
| Assignees | `[]*User` | `[]*BasicUser` |

## Security Alerts

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| Dependabot alerts | `Dependabot.ListRepoAlerts(owner, repo, opts)` | No direct equivalent. Dependency scanning produces vulnerability reports. |
| Code scanning alerts | `CodeScanning.ListAlertsForRepo(owner, repo, opts)` | SAST/DAST results via `ProjectVulnerabilities.ListProjectVulnerabilities(projectID, opts)` |
| Secret scanning alerts | `SecretScanning.ListAlertsForRepo(owner, repo, opts)` | Secret detection results also via vulnerability API |
| Availability | Free tier (Dependabot); GHAS for code/secret scanning | **GitLab Ultimate only** for most vulnerability features |
| Alert model | Separate types per scanner | Unified `Vulnerability` type with `scanner` and `severity` fields |
| Counting trick | per_page=1, read `LastPage` from response | per_page=1, read `X-Total` header |

### Implication

Security features in GitLab are tier-gated (Ultimate). The default config should have
`securityAlerts: false` for GitLab. When enabled, use a single vulnerability API to
populate all three alert count fields, filtered by scanner type.

## Actions/CI Toggle

| GitHub | GitLab |
|--------|--------|
| `Repositories.GetActionsPermissions(owner, repo)` — returns enabled/disabled | No equivalent — CI runs if `.gitlab-ci.yml` exists |
| `Repositories.EditActionsPermissions(owner, repo, perms)` — disable Actions | No direct toggle. Can disable CI via project settings (`builds_access_level`) |
| Used in git-janitor for forks: "CI on fork is wasteful" check | Not applicable — GitLab forks don't inherit CI config by default |

### Implication

The `github-fork-actions-enabled` check has **no GitLab equivalent**. GitLab forks start
with no CI unless the user explicitly adds `.gitlab-ci.yml`. The `ActionsEnabled` field
stays as -1 (not applicable) for GitLab repos.

## Pagination

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| Total count | `resp.LastPage` (when per_page=1, LastPage = total items) | `resp.Header.Get("X-Total")` (explicit total count header) |
| Next page | `resp.NextPage` (0 = no more pages) | `resp.NextPage` (0 = no more pages) — same pattern |
| Page param | `ListOptions{Page: n, PerPage: n}` | `ListOptions{Page: n, PerPage: n}` — same struct |

GitLab actually provides the total count directly via the `X-Total` header, which is
more reliable than GitHub's `LastPage` inference trick.

## Error Handling

| Aspect | GitHub | GitLab |
|--------|--------|--------|
| 404 meaning | Resource not found OR insufficient permissions | Resource not found OR insufficient permissions (same) |
| 403 meaning | Rate limited or forbidden | Forbidden (rate limit uses 429) |
| Error type | `*github.ErrorResponse` | `*gitlab.ErrorResponse` |
| Status code access | `resp.StatusCode` | `resp.StatusCode` |

## Delete Branch on Merge

| GitHub | GitLab |
|--------|--------|
| Repo-level setting: `DeleteBranchOnMerge` | Project-level default: `RemoveSourceBranchAfterMerge` |
| Applies to all PRs automatically when enabled | Default for new MRs; can be overridden per-MR |
| Toggle: `Repositories.Edit(owner, repo, &Repository{DeleteBranchOnMerge: &true})` | Toggle: `Projects.EditProject(id, &EditProjectOptions{RemoveSourceBranchAfterMerge: Ptr(true)})` |

Semantically equivalent for git-janitor's purposes.

## Summary of Key Architectural Differences

1. **Multi-host**: GitHub is (almost) always `github.com`. GitLab is commonly self-hosted.
   The client must accept a configurable base URL, and git-janitor may talk to multiple
   GitLab instances simultaneously (keyed by base URL).

2. **Nested paths**: GitLab project paths are arbitrarily deep. URL parsing must handle this.

3. **Integer IDs**: GitLab uses integer project IDs as the primary API identifier. Store
   `ProjectID` in `PlatformInfo` to avoid re-resolving paths on every action call.

4. **Visibility ternary**: GitLab has `public`/`internal`/`private` (3 levels vs GitHub's 2).
   The `IsPrivate` bool covers `private`; `internal` could be surfaced as an info-level alert.

5. **No Actions toggle**: CI on GitLab is file-driven; no "disable Actions" equivalent.

6. **Security is tier-gated**: Vulnerability API requires GitLab Ultimate. Default to off.

7. **MRs vs PRs**: Cosmetic difference (terminology) but mechanically very similar.
   State value "opened" vs "open" needs normalization.

8. **Pipelines vs Workflow Runs**: GitLab pipelines are simpler (no named workflows,
   duration is directly available). Status and conclusion are merged into one field.

9. **Auto-protected default branch**: GitLab protects the default branch by default;
   the "protection missing" check will rarely fire but should still exist for edge cases.

10. **Issue counting**: GitLab separates issues from MRs natively, so no need for the
    per_page=1 trick to count PRs and subtract from issue count.

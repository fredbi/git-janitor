# GitHub Checks Catalog

> Last revised: 2026-03-29
> Companion to [github-checks-plan.md](github-checks-plan.md)

## Check Specifications

### Initial Batch (cheap — single `repos.Get` call)

---

#### `github-repo-archived`

| Field | Value |
|-------|-------|
| Severity | High |
| Default | enabled |
| Input | `RepoData.IsArchived` |
| Description | Repo is archived on GitHub — read-only, no push allowed |
| Suggestion | None (informational) |
| Detail | "Repository {FullName} is archived on GitHub. It is read-only: push, branch, and tag operations will fail." |

---

#### `github-default-branch-mismatch`

| Field | Value |
|-------|-------|
| Severity | Low |
| Default | enabled |
| Input | `RepoData.DefaultBranch`, `git.RepoInfo.DefaultBranch` (from `LastRepoInfo`) |
| Description | GitHub default branch differs from local default branch |
| Suggestion | None (informational, user decides which is correct) |
| Detail | "GitHub default branch is '{ghDefault}' but local HEAD points to '{localDefault}'." |

**Note**: This check needs access to both GitHub data and git data. The evaluate method receives
`*github.RepoData`. We need to pass the local default branch into `RepoData` at collection time
(from the git.RepoInfo that was already collected), or store it as a field on `RepoData`.
Design choice: add `LocalDefaultBranch string` to `RepoData`, populated when the infos panel
calls `SetGitHubData` (it already has `LastRepoInfo`).

---

#### `github-description-missing`

| Field | Value |
|-------|-------|
| Severity | Low |
| Default | enabled |
| Input | `RepoData.Description` |
| Description | No description set on GitHub |
| Suggestion | None (informational) |
| Detail | "Repository {FullName} has no description on GitHub." |

---

#### `github-visibility-private`

| Field | Value |
|-------|-------|
| Severity | Info |
| Default | enabled |
| Input | `RepoData.IsPrivate` |
| Description | Informational: repo is private |
| Suggestion | None |
| Detail | "Repository {FullName} is private." |

---

#### `github-repo-fork-parent`

| Field | Value |
|-------|-------|
| Severity | Info |
| Default | enabled |
| Input | `RepoData.IsFork`, `RepoData.ParentFullName` |
| Description | Informational: identifies fork parent |
| Suggestion | None |
| Detail | "Repository {FullName} is a fork of {ParentFullName}." |
| No-alert | When `IsFork == false` |

---

### Expensive Batch (extra API calls, opt-in)

---

#### `github-issues-unresponded`

| Field | Value |
|-------|-------|
| Severity | Medium |
| Default | disabled |
| Input | `RepoData.UnrespondedIssues` (int, from paginated API call) |
| Extra API | `issues.ListByRepo` with `state=open`, `sort=updated`, filtered for 0 comments |
| Description | Open issues with no comments — may need triage |
| Suggestion | None (informational) |
| Detail | "{N} open issue(s) have no comments and may need triage." |
| Threshold | Alert when N > 0 |
| Rate-limit | Single paginated call. Cap at 100 issues scanned (first page only). |

**RepoData additions**: `UnrespondedIssues int` (populated only when this check is enabled).

---

#### `github-prs-pending-review`

| Field | Value |
|-------|-------|
| Severity | Medium |
| Default | disabled |
| Input | `RepoData.PendingReviewPRs` (int, from paginated API call) |
| Extra API | `pulls.List` with `state=open`, filter for 0 reviews |
| Description | Open PRs with no reviews — may need attention |
| Suggestion | None (informational) |
| Detail | "{N} open PR(s) have no reviews." |
| Threshold | Alert when N > 0 |
| Rate-limit | Single paginated call. Cap at 100 PRs scanned (first page only). |

**RepoData additions**: `PendingReviewPRs int` (populated only when this check is enabled).

---

#### `github-security-alerts`

| Field | Value |
|-------|-------|
| Severity | High |
| Default | disabled |
| Input | `RepoData.VulnerabilityAlerts` (int) |
| Extra API | GraphQL `repository.vulnerabilityAlerts(states: OPEN)` or REST Dependabot alerts |
| Description | Open vulnerability alerts on the repository |
| Suggestion | None (informational — user should review on GitHub) |
| Detail | "{N} open vulnerability alert(s). Review at {HTMLURL}/security/dependabot." |
| Threshold | Alert when N > 0. N == -1 means "no access" (skip) |
| Token scope | Requires `security_events` scope or repo admin access |

**Note**: If the token lacks the required scope, the API returns 403. Set
`VulnerabilityAlerts = -1` and skip the check silently.

---

## Phase 3 Checks (planned, not yet specified)

These will be specified in a future revision when Phase 1 GitHub checks are validated.

| Check Name | Category | Description |
|---|---|---|
| `github-protected-branches` | Settings | Branch protection rules audit |
| `github-ci-workflows-status` | CI | GitHub Actions workflow run status |
| `github-contributors` | Info | Active contributor count and recency |
| `github-latest-release` | Release | Latest release info, signed/immutable checks |
| `github-security-settings-check` | Security | Audit against a security settings template |
| `github-push-rules-check` | Settings | Push rules audit against template |

---

## Actions

Phase 1: **No GitHub actions**. All checks are informational-only.

Future candidates:
- `open-in-browser` — open repo/PR/issue in default browser (Phase 2)
- `sync-fork` — sync fork with upstream via GitHub API (Phase 2)
- `enable-vulnerability-alerts` — enable Dependabot via API (Phase 3)
- `create-branch-protection` — apply standard branch protection (Phase 3)

---

## RepoData Field Summary

| Field | Source | Cheap | Expensive |
|-------|--------|-------|-----------|
| Owner, Repo, FullName | `repos.Get` | yes | — |
| Description, HTMLURL | `repos.Get` | yes | — |
| IsFork, IsArchived, IsPrivate | `repos.Get` | yes | — |
| DefaultBranch | `repos.Get` | yes | — |
| Topics, License | `repos.Get` | yes | — |
| OpenIssues (includes PRs) | `repos.Get` | yes | — |
| StarCount, ForkCount | `repos.Get` | yes | — |
| ParentFullName | `repos.Get` (Parent field) | yes | — |
| CreatedAt, UpdatedAt, PushedAt | `repos.Get` | yes | — |
| OpenPRs | `pulls.List per_page=1` | yes (cheap extra) | — |
| LocalDefaultBranch | from `git.RepoInfo` | injected | — |
| UnrespondedIssues | `issues.ListByRepo` paginated | — | yes |
| PendingReviewPRs | `pulls.List` + reviews check | — | yes |
| VulnerabilityAlerts | GraphQL or Dependabot REST | — | yes |

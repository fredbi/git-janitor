> [!NOTE]
> Last revision: 2026-03-29

# Git Checks & Actions Catalog

## Summary

Complete catalog of all git checks and their paired actions for Phase 1.
Each check evaluates `git.RepoInfo` fields and produces alerts with optional action suggestions.
This document serves as the specification for subagent-driven implementation.

## Preliminary actions before batch implementation

* `SeverityCritical` has been added to `engine/types.go`
* `health-fsck-errors` must be updated to use `SeverityCritical`
* `branch-no-upstream` must be updated to suggest `push-branch` action
* `branch-diverged` must be updated to only suggest rebase when RebaseCheck confirms feasibility
* New check `branch-not-mergeable` must be added
* New action `push-branch` must be implemented

---

## Current state

### Implemented checks (8) — corrections needed

| Check | File | Severity | Paired action | Correction needed |
|---|---|---|---|---|
| `health-gc-advised` | health.go | low | `run-gc` | — |
| `size-repack-advised` | health.go | low | `run-gc-aggressive` | — |
| `health-fsck-errors` | health.go | ~~high~~ **critical** | — (manual) | update severity to SeverityCritical |
| `branch-lagging` | branches.go | low | `update-branch` | — |
| `branch-merged-not-deleted` | branches.go | medium | `delete-branch` | local delete only (remote is Phase 4) |
| `branch-gone-upstream` | branches.go | medium | `delete-branch` | — (already local delete) |
| `branch-no-upstream` | branches.go | low | ~~—~~ **`push-branch`** | add push-branch suggestion |
| `branch-diverged` | branches.go | medium | `rebase-branch` | only suggest when rebasable/squash-rebasable |

### New check to add alongside corrections

| Check | File | Severity | Paired action | Notes |
|---|---|---|---|---|
| `branch-not-mergeable` | branches.go | medium | — (manual) | fires when MergeCheck.Clean == false AND RebaseCheck.CanRebase == false AND RebaseCheck.CanRebaseSquashed == false |

### Implemented actions (2)

| Action | File | Destructive | Auto | Runner method |
|---|---|---|---|---|
| `run-gc` | maintenance.go | no | yes | `Runner.Compact` |
| `run-gc-aggressive` | maintenance.go | yes | no | `Runner.CompactAggressive` |

---

## Checks to implement

### 1. Dirty worktree

| Field | Value |
|---|---|
| **Check name** | `dirty-worktree` |
| **File** | `checks/git/worktree.go` (new) |
| **Description** | detects uncommitted changes in the working tree |
| **RepoInfo fields** | `Status.IsDirty()` |
| **Severity** | high |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational, user must commit/stash themselves) |
| **Detail** | count of staged, unstaged, and untracked entries |

### 2. Activity — stale

| Field | Value |
|---|---|
| **Check name** | `activity-stale` |
| **File** | `checks/git/activity.go` (new) |
| **Description** | detects repositories with no merged commits in the last 360 days |
| **RepoInfo fields** | `Activity.Staleness == "stale"` |
| **Severity** | low |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |
| **Detail** | last commit date, commit counts (7d/30d/90d/360d) |

### 3. Activity — dormant

| Field | Value |
|---|---|
| **Check name** | `activity-dormant` |
| **File** | `checks/git/activity.go` (same file) |
| **Description** | detects repositories with no merged commits in over 360 days |
| **RepoInfo fields** | `Activity.Staleness == "dormant"` |
| **Severity** | low |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |
| **Detail** | last commit date |

### 4. Config — no email

| Field | Value |
|---|---|
| **Check name** | `config-no-email` |
| **File** | `checks/git/config.go` (new) |
| **Description** | detects repositories with no user.email configured |
| **RepoInfo fields** | `Config.UserEmail.Value == ""` |
| **Severity** | medium |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |
| **Detail** | reports which scope (global/local/unset) the email is missing from |

### 5. Config — unsigned commits

| Field | Value |
|---|---|
| **Check name** | `config-unsigned` |
| **File** | `checks/git/config.go` (same file) |
| **Description** | detects repositories where commit signing is not enabled |
| **RepoInfo fields** | `Config.CommitSign.Value != "true"` |
| **Severity** | info |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |
| **Detail** | reports current commit.gpgsign value and scope |

### 6. Tags — local only

| Field | Value |
|---|---|
| **Check name** | `tags-local-only` |
| **File** | `checks/git/tags.go` (new) |
| **Description** | detects tags that exist locally but not on the remote |
| **RepoInfo fields** | `Tags[].LocalOnly` |
| **Severity** | low |
| **SubjectKind** | Tag |
| **Suggested action** | `push-tag` |
| **Detail** | lists the local-only tag names |
| **Note** | one suggestion with all local-only tags as subjects; action pushes one by one |
| **Default** | **disabled** in config (sensitive — needs parameterization before auto-use) |

### 7. Tags — remote only

| Field | Value |
|---|---|
| **Check name** | `tags-remote-only` |
| **File** | `checks/git/tags.go` (same file) |
| **Description** | detects tags that exist on the remote but not locally |
| **RepoInfo fields** | `Tags[].RemoteOnly` |
| **Severity** | low |
| **SubjectKind** | Repo |
| **Suggested action** | `fetch-tags` |
| **Detail** | lists the remote-only tag names |
| **Note** | repo-level action: `git fetch --all --tags` fetches everything |

### 8. File stats — large files

| Field | Value |
|---|---|
| **Check name** | `filestats-large-files` |
| **File** | `checks/git/filestats.go` (new) |
| **Description** | detects files in HEAD exceeding the size threshold |
| **RepoInfo fields** | `FileStats.LargeFiles` (len > 0) |
| **Severity** | low |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |
| **Detail** | lists file paths and sizes |

### 9. File stats — binary files

| Field | Value |
|---|---|
| **Check name** | `filestats-binary` |
| **File** | `checks/git/filestats.go` (same file) |
| **Description** | detects binary files tracked in HEAD |
| **RepoInfo fields** | `FileStats.BinaryFiles` (len > 0) |
| **Severity** | info |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |
| **Detail** | lists binary file paths |

### 10. Traits — shallow clone

| Field | Value |
|---|---|
| **Check name** | `traits-shallow` |
| **File** | `checks/git/traits.go` (new) |
| **Description** | detects shallow clones (incomplete history) |
| **RepoInfo fields** | `IsShallow` |
| **Severity** | info |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |

### 11. Traits — submodules

| Field | Value |
|---|---|
| **Check name** | `traits-submodules` |
| **File** | `checks/git/traits.go` (same file) |
| **Description** | detects repositories using git submodules |
| **RepoInfo fields** | `HasSubmodules` |
| **Severity** | info |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |

### 12. Traits — LFS

| Field | Value |
|---|---|
| **Check name** | `traits-lfs` |
| **File** | `checks/git/traits.go` (same file) |
| **Description** | detects repositories using Git LFS |
| **RepoInfo fields** | `HasLFS` |
| **Severity** | info |
| **SubjectKind** | Repo |
| **Suggested action** | — (informational) |

---

## Actions to implement

### 1. delete-branch

| Field | Value |
|---|---|
| **Action name** | `delete-branch` |
| **File** | `actions/git/branch.go` (new) |
| **Description** | delete a local branch |
| **ApplyTo** | Branch |
| **Destructive** | yes |
| **Auto** | false |
| **Runner method** | **NEW: `Runner.DeleteBranch(ctx, name string)`** |
| **Git command** | `git branch -D <name>` |
| **Guards** | refuse if branch is current (IsCurrent); trust check's Merged flag for safety |
| **Note** | uses `-D` because squash-merged branches aren't recognized by `-d` |

### 2. update-branch

| Field | Value |
|---|---|
| **Action name** | `update-branch` |
| **File** | `actions/git/branch.go` (same file) |
| **Description** | fast-forward a local branch from its upstream |
| **ApplyTo** | Branch |
| **Destructive** | no |
| **Auto** | true |
| **Runner method** | `Runner.UpdateBranch` (exists) |
| **Note** | looks up `git.Branch` from `RepoInfo.Branches` by name |

### 3. rebase-branch

| Field | Value |
|---|---|
| **Action name** | `rebase-branch` |
| **File** | `actions/git/branch.go` (same file) |
| **Description** | rebase a local branch onto the default branch |
| **ApplyTo** | Branch |
| **Destructive** | no |
| **Auto** | false |
| **Runner method** | `Runner.RebaseBranch` (exists) |
| **Note** | looks up Branch struct + uses DefaultBranch from RepoInfo |

### 4. push-branch

| Field | Value |
|---|---|
| **Action name** | `push-branch` |
| **File** | `actions/git/branch.go` (same file) |
| **Description** | push a local branch to origin and set upstream tracking |
| **ApplyTo** | Branch |
| **Destructive** | no |
| **Auto** | false |
| **Runner method** | **NEW: `Runner.PushBranch(ctx, name string)`** |
| **Git command** | `git push -u origin <branch>` |

### 5. push-tag

| Field | Value |
|---|---|
| **Action name** | `push-tag` |
| **File** | `actions/git/tag.go` (new) |
| **Description** | push a local tag to the origin remote |
| **ApplyTo** | Tag |
| **Destructive** | no |
| **Auto** | false |
| **Runner method** | **NEW: `Runner.PushTag(ctx, name string)`** |
| **Git command** | `git push origin <tag>` |

### 6. fetch-tags

| Field | Value |
|---|---|
| **Action name** | `fetch-tags` |
| **File** | `actions/git/tag.go` (same file) |
| **Description** | fetch all tags from all remotes |
| **ApplyTo** | Repo |
| **Destructive** | no |
| **Auto** | true |
| **Runner method** | `Runner.FetchAllTags` (exists) |

---

## New Runner methods needed

### `Runner.DeleteBranch`

```
git branch -D <name>
```

Guards: refuse if name matches current branch.
Add to `internal/git/actions.go`. Returns `ActionResult`.
Add `cmdDeleteBranch(name)` to `git_commands.go`.

### `Runner.PushBranch`

```
git push -u origin <branch>
```

Add to `internal/git/actions.go`. Returns `ActionResult`.
Add `cmdPushBranchUpstream(remote, branch)` to `git_commands.go`.

### `Runner.PushTag`

```
git push origin <tag>
```

Add to `internal/git/actions.go`. Returns `ActionResult`.
Add `cmdPushTag(remote, tag)` to `git_commands.go`.

---

## Config defaults update

After all checks and actions are implemented, `default_config.yaml` should list:

```yaml
defaults:
  rules:
    checks:
      # Branch hygiene
      - name: branch-merged-not-deleted
      - name: branch-gone-upstream
      - name: branch-lagging
      - name: branch-no-upstream
      - name: branch-diverged
      - name: branch-not-mergeable
      # Worktree
      - name: dirty-worktree
      # Health & maintenance
      - name: health-gc-advised
      - name: size-repack-advised
      - name: health-fsck-errors
      # Activity
      - name: activity-stale
      - name: activity-dormant
      # Config audit
      - name: config-no-email
      - name: config-unsigned
      # Tags (tags-local-only disabled by default — sensitive)
      # - name: tags-local-only
      - name: tags-remote-only
      # File stats
      - name: filestats-large-files
      - name: filestats-binary
      # Traits
      - name: traits-shallow
      - name: traits-submodules
      - name: traits-lfs
    actions:
      # Branch actions
      - name: delete-branch
        auto: false
      - name: update-branch
        auto: true
      - name: rebase-branch
        auto: false
      - name: push-branch
        auto: false
      # Maintenance
      - name: run-gc
        auto: true
      - name: run-gc-aggressive
        auto: false
      # Tags
      - name: push-tag
        auto: false
      - name: fetch-tags
        auto: true
```

---

## Implementation order

### Batch 0: Fix existing checks + new Runner methods

1. Update `health-fsck-errors` to `SeverityCritical`
2. Update `branch-no-upstream` to suggest `push-branch`
3. Update `branch-diverged` to gate on RebaseCheck feasibility
4. Add `branch-not-mergeable` check to branches.go
5. Add `Runner.DeleteBranch`, `Runner.PushBranch`, `Runner.PushTag` + git_commands.go entries

### Batch 1: Actions (reference new + existing Runner methods)

6. Action `delete-branch` (with IsCurrent guard)
7. Action `update-branch` (Branch lookup from RepoInfo)
8. Action `rebase-branch` (Branch lookup + DefaultBranch)
9. Action `push-branch`
10. Action `push-tag`
11. Action `fetch-tags`
12. Register all in `actions/git/register.go`

### Batch 2: Checks with actions

13. Check `tags-local-only` → suggests `push-tag`
14. Check `tags-remote-only` → suggests `fetch-tags`

### Batch 3: Informational checks (parallelize freely)

15. Check `dirty-worktree`
16. Check `activity-stale` + `activity-dormant`
17. Check `config-no-email` + `config-unsigned`
18. Check `filestats-large-files` + `filestats-binary`
19. Check `traits-shallow` + `traits-submodules` + `traits-lfs`

### Batch 4: Config + final wiring

20. Update `default_config.yaml` with all checks and actions
21. Update `git-check-authoring.md` skill with new actions table

---

## Summary table (all 21 checks, 8 actions)

### Checks

| # | Check | Severity | Action | Status |
|---|---|---|---|---|
| 1 | `health-gc-advised` | low | `run-gc` | ✅ done |
| 2 | `size-repack-advised` | low | `run-gc-aggressive` | ✅ done |
| 3 | `health-fsck-errors` | critical | — | ⚠️ fix severity |
| 4 | `branch-lagging` | low | `update-branch` | ✅ check done, action needed |
| 5 | `branch-merged-not-deleted` | medium | `delete-branch` | ✅ check done, action needed |
| 6 | `branch-gone-upstream` | medium | `delete-branch` | ✅ check done, action needed |
| 7 | `branch-no-upstream` | low | `push-branch` | ⚠️ fix: add suggestion |
| 8 | `branch-diverged` | medium | `rebase-branch` | ⚠️ fix: gate on feasibility |
| 9 | `branch-not-mergeable` | medium | — | 📝 new check needed |
| 10 | `dirty-worktree` | high | — | 📝 check needed |
| 11 | `activity-stale` | low | — | 📝 check needed |
| 12 | `activity-dormant` | low | — | 📝 check needed |
| 13 | `config-no-email` | medium | — | 📝 check needed |
| 14 | `config-unsigned` | info | — | 📝 check needed |
| 15 | `tags-local-only` | low | `push-tag` | 📝 check needed (disabled by default) |
| 16 | `tags-remote-only` | low | `fetch-tags` | 📝 check needed |
| 17 | `filestats-large-files` | low | — | 📝 check needed |
| 18 | `filestats-binary` | info | — | 📝 check needed |
| 19 | `traits-shallow` | info | — | 📝 check needed |
| 20 | `traits-submodules` | info | — | 📝 check needed |
| 21 | `traits-lfs` | info | — | 📝 check needed |

### Actions

| # | Action | ApplyTo | Destructive | Auto | Status |
|---|---|---|---|---|---|
| 1 | `run-gc` | Repo | no | yes | ✅ done |
| 2 | `run-gc-aggressive` | Repo | yes | no | ✅ done |
| 3 | `delete-branch` | Branch | yes | no | 📝 needed |
| 4 | `update-branch` | Branch | no | yes | 📝 needed |
| 5 | `rebase-branch` | Branch | no | no | 📝 needed |
| 6 | `push-branch` | Branch | no | no | 📝 needed |
| 7 | `push-tag` | Tag | no | no | 📝 needed |
| 8 | `fetch-tags` | Repo | no | yes | 📝 needed |

---

## Deferred to later phases

### Phase 2 — missing features noted during review

**Repo health:**
- Check for stale `.git/modules` (orphaned submodule data)
- Cleanup old git notes

**Unreachable object bloat (new check + action):**
- New check: `health-unreachable-bloat` — detects repos where `.git` is much larger than reachable
  objects AND the cause is unreachable objects held by reflogs (not structural overhead).
  Distinct from `size-repack-advised` which focuses on pack fragmentation.
  - Detection: high waste ratio (`.git` vs `rev-list --disk-usage --all`) where `.git` > 5 MB
  - Severity: low (the space is reclaimable but no data loss risk)
  - Suggested action: `deep-clean` (new)
- New action: `deep-clean` — chain of: `git reflog expire --expire-unreachable=now --all`
  then `git gc --prune=now`. This is a single combined action (first chain-of-actions use case).
  - Destructive: yes (loses reflog history — can't undo past rebases/amends)
  - Auto: false
  - Runner: new `Runner.DeepClean(ctx)` method
- Rationale: `git gc --aggressive` doesn't help when bloat comes from reflog-held objects.
  Common on repos with heavy rebase workflows. Validated on go-openapi repos where waste
  ratios of 30x-126x persist after aggressive gc.
  go-swagger is another example: after massive branch/file pruning, unreachable objects
  from deleted history stay in packs for 2 weeks (gc.pruneExpire default).
- The current `size-repack-advised` check should be narrowed to only fire on genuine pack
  fragmentation (too many packs, large loose objects) and NOT on waste ratio — waste ratio
  should be handled by this new check instead.

**Repack threshold calibration (done 2026-03-29):**
- Packs threshold raised from 5 to 20 (routine fetches create small packs)
- Loose/packed ratio now requires packSize ≥ 1 MB (meaningless on tiny repos)
- Waste ratio now requires .git ≥ 5 MB and absolute waste ≥ 1 MB
- Validated against 16 go-openapi repos: small repos (< 5 MB .git) no longer trigger

**Branch hygiene — merge/rebase strategy selection:**
- Single commit since branch-out → rebase
- Multiple commits, each mergeable → rebase
- Multiple commits, overall mergeable but intermediary conflicts → squash-rebase (destructive)
- Configurable: `merge-strategy: [merge-parent|rebase]`
- Check: branch-out commit lagging HEAD of parent + branch is rebasable → suggest rebase onto parent

**Forgotten stashes:**
- Stashes older than N days (parameterizable)

**Forgotten dirty worktree:**
- Dirty state unchanged for 7+ days (parameterizable: `stale-dirty-days: 7`)
- Action: create branch, commit dirty files, push upstream
- Configurable commit title (default: "experimental: temporary work")

**Config audit — additional:**
- Check for missing user.name (similar to no-email)
- Action: auto-configure user.name/email from config as local/global
- Check: gpgsign enabled but signingkey not defined

**Tags — push parameterization:**
- Push target options: upstream-only, origin-only, all-remotes
- Check for unsigned tags (similar to unsigned commits)

**File stats — binary exceptions:**
- Configurable regexp for tolerated binary files (e.g. `*.png`, `*.ico`)

### Phase 4 — edge cases

**Remote branch deletion:**
- `delete-remote-branch` action: `git push origin --delete <branch>`
- Default remotes: upstream (not origin) — parameterizable
- Most repos already have GitHub's "delete head branch after merge" setting

**Shallow clone management:**
- Action: unshallow clone + refetch tags
- Inverse option: want-shallow with configurable fetch-depth
- Auto-clean: branch out, delete binaries not in exceptions, commit and push

**Submodules:**
- Check submodule lagging its remote / action: submodule sync
- Scan submodule as independent repo — figure out UI rendering

**Binary auto-clean:**
- In worktree, branch out, delete binary files not in exceptions, commit and push
- Configurable commit title

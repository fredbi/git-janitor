// Package git is a frontend for running git CLI commands against local repositories.
//
// It requires git (>= 2.38 for merge-tree checks) to be installed on the local host.
// All operations are executed via the git CLI — no libgit2 or go-git dependency.
//
// # Runner
//
// [Runner] is the entry point. Create one with [NewRunner] for a given repository directory.
// All methods on Runner accept a context for timeout and cancellation.
// The default command timeout is 30 seconds.
//
// # Repository inspection
//
// [Runner.Status] parses git status --porcelain=v2 into a [Status] struct:
// current branch, HEAD OID, upstream tracking ref, ahead/behind counts,
// and a list of changed/untracked/ignored [StatusEntry] items.
//
// [CollectRepoInfo] gathers a complete [RepoInfo] snapshot for a repository:
// status, branches, remotes, stashes, default branch, last commit time,
// SCM provider, repository kind, merge/rebase feasibility per branch.
//
// # Branches
//
// [Runner.Branches] lists all local and remote branches with metadata parsed from
// git branch -a --format. Each [Branch] includes:
//
//   - Name, Hash, IsCurrent, IsRemote
//   - Upstream tracking ref (if configured) with [Branch.HasUpstream]
//   - Ahead/Behind counts relative to the upstream
//   - Gone flag (upstream deleted from remote)
//   - Merged flag (reachable from or fully applied to the default branch)
//   - MergeCheck and RebaseCheck results (see below)
//
// [Runner.LocalBranches] and [Runner.RemoteBranches] filter by locality.
// [Runner.DefaultBranch] detects the default branch (origin/HEAD, then main/master, then current).
//
// # Merge detection
//
// [Runner.MergedBranches] uses git branch --merged to find branches whose tips
// are reachable from a target branch.
//
// [Runner.IsFullyApplied] uses git cherry to compare patch IDs, detecting branches
// whose changes were incorporated via squash-merge or rebase (not just ancestry).
//
// [MarkMerged] combines both strategies: first the fast reachability check,
// then the patch-id fallback for remaining local branches.
//
// # Merge feasibility
//
// [Runner.CanMerge] performs a dry-run merge using git merge-tree --write-tree.
// The merge is computed entirely in memory — no worktree or index changes.
// Returns a [MergeCheck] with Clean status and any conflicting file paths.
//
// [Runner.CheckMergeable] runs CanMerge for all unmerged local branches,
// populating [Branch.MergeCheck].
//
// # Rebase feasibility
//
// [Runner.CheckRebase] performs a two-strategy dry-run rebase analysis:
//
//   - Direct rebase: replays each commit one by one onto the target using
//     git merge-tree --merge-base and git commit-tree to build a synthetic
//     commit chain. This catches per-commit conflicts.
//   - Squash-first fallback: if direct rebase fails, checks whether squashing
//     all commits first would allow a clean rebase (equivalent to merge-tree).
//
// All operations use plumbing commands only. No refs, worktree, or index are
// modified. Synthetic commit objects are unreferenced and garbage-collected.
//
// Returns a [RebaseCheck] with results for both strategies, the failing step,
// and conflicting file paths.
//
// [Runner.CheckRebaseable] runs CheckRebase for all unmerged local branches,
// populating [Branch.RebaseCheck].
//
// # Tags
//
// [Runner.Tags] lists all tags with metadata parsed from git tag -l --format.
// Each [Tag] includes:
//
//   - Name, Hash, TargetHash (dereferenced commit), Date, Message
//   - Annotated flag (objecttype == "tag") vs lightweight (objecttype == "commit")
//   - Signed flag (GPG/SSH signature present)
//   - Semver parsing: IsSemver, HasVPrefix, IsPrerelease, SemverMajor/Minor/Patch/Prerelease
//   - OnDefaultBranch (tagged commit is reachable from default branch)
//   - LocalOnly (exists locally but not on origin) / RemoteOnly (on origin but not fetched)
//
// Semver matching: "v1.2.3", "1.2.3", "v1.2.3-beta.1", "v1.2.3+build" are all recognized.
//
// [CompareSemver] orders tags by version: major, minor, patch, then prerelease < release.
//
// [DeriveTagSummary] computes summary fields from a tag list:
// last tag date (any tag), last semver tag (by version ordering), and its date.
// These are included in [RepoInfo] by [CollectRepoInfo].
//
// # Remotes
//
// [Runner.Remotes] parses git remote -v into [Remote] structs (name, fetch URL, push URL).
// [Runner.RemoteMap] returns a convenience map of remote name to fetch URL.
//
// # SCM and repository kind
//
// [DeriveSCM] inspects the origin remote URL to classify the hosting platform:
// "github", "gitlab", or "other". Uses regexp matching on the hostname.
//
// [DeriveKind] compares origin and upstream remote URLs to classify the repository:
// "clone" (single remote or same URL), "fork" (different URLs), or "not-git".
//
// URL comparison normalizes across SSH and HTTPS schemes via [NormalizeURL].
// [ExtractHost] and [OriginFetchURL] are exported helpers.
//
// # Configuration
//
// [Runner.Config] queries a curated set of git config values and returns a [RepoConfig].
// Each entry is a [ConfigEntry] with the effective value and its [ConfigScope]
// (system, global, local, worktree, or unset). The IsLocal flag indicates whether
// the value is defined in the repo's own config rather than inherited.
//
// Queried keys: user.email, user.name, user.signingkey, commit.gpgsign, tag.gpgsign.
//
// Uses git config --show-scope --get (requires git >= 2.26).
// The config is included in [RepoInfo] by [CollectRepoInfo].
//
// # File stats
//
// [Runner.FileStats] collects information about large and binary files in a [FileStats]:
//
//   - LargeFiles: files in HEAD exceeding a configurable threshold (default 1 MB),
//     found via git ls-tree -r -l HEAD.
//   - LargeBlobs: the largest blob objects across all history (including deleted files),
//     found via git rev-list --objects --all piped to git cat-file --batch-check.
//     This catches binaries that were committed and later removed but still occupy
//     space in the pack.
//   - BinaryFiles: files in HEAD that git considers binary, detected via
//     git diff --numstat against the empty tree (binary files show as "-\t-\tpath").
//
// Options: [WithLargeThreshold] and [WithTopBlobs] configure the query.
// The file stats are included in [RepoInfo] by [CollectRepoInfo].
//
// # Repository size
//
// [Runner.Size] collects size metrics in a [RepoSize]:
//
//   - GitDirBytes: total .git directory size on disk (filesystem walk).
//   - ReachableBytes: size of all reachable objects (git rev-list --disk-usage --all).
//   - RepackAdvised + RepackReasons: advisory based on pack file count,
//     loose-to-packed size ratio, absolute .git size, and .git-to-reachable bloat ratio.
//
// Both measurements are fast (< 50ms on 100MB repos). The size report is included
// in [RepoInfo] by [CollectRepoInfo].
//
// # Repository traits
//
// [Runner.IsShallow] detects shallow clones via git rev-parse --is-shallow-repository.
//
// [Runner.HasSubmodules] checks for the presence of a .gitmodules file.
//
// [Runner.HasLFS] scans .gitattributes for "filter=lfs" entries, detecting Git LFS
// usage without requiring git-lfs to be installed.
//
// All three are included in [RepoInfo] by [CollectRepoInfo].
//
// # Health
//
// [Runner.Health] performs a repository health check and returns a [HealthReport]:
//
//   - Integrity: runs git fsck --connectivity-only to detect corruption
//     (missing objects, broken links). Dangling objects are excluded as benign.
//   - GC diagnostics: runs git count-objects -v to gather loose object counts,
//     pack file statistics, prune-packable duplicates, and garbage files.
//   - GC advisory: evaluates whether git gc would be beneficial based on:
//     loose object count vs gc.auto threshold, prune-packable duplicates,
//     pack file proliferation, and garbage files.
//
// The health report is included in [RepoInfo] by [CollectRepoInfo].
//
// # Worktrees
//
// [Runner.Worktrees] lists all worktrees (main + linked) via git worktree list --porcelain.
// Each [Worktree] includes path, HEAD hash, branch, and flags for detached/bare/prunable state.
// [Worktree.BranchShort] strips the refs/heads/ prefix for display.
//
// Worktrees are included in [RepoInfo] by [CollectRepoInfo].
//
// The rebase action ([Runner.RebaseBranch]) uses temporary worktrees for non-checked-out
// branches to avoid disturbing the user's main checkout.
//
// # Stashes
//
// [Runner.Stashes] parses git stash list into [Stash] structs (ref, branch, message).
//
// # Fetch
//
// [Runner.Fetch] fetches a single remote.
// [Runner.FetchAll] fetches all remotes.
// [Runner.FetchAllTags] fetches all remotes including tags.
//
// # Refresh
//
// [RefreshRepo] combines FetchAllTags + CollectRepoInfo into a single operation,
// returning an up-to-date [RepoInfo] snapshot.
//
// # Last commit
//
// [Runner.LastCommitTime] returns the author date of the most recent commit on HEAD.
//
// # Actions
//
// The following methods perform actual git operations (not just checks).
// All actions that modify the repository guard against a dirty worktree.
//
// [Runner.UpdateBranch] fast-forwards a local branch from its upstream remote.
// For the current branch, it uses git pull --ff-only (after checking for a clean worktree).
// For non-checked-out branches, it uses git fetch <remote> <branch>:<branch> to update
// the ref without switching — no worktree guard needed since the worktree is untouched.
// Only fast-forward updates are attempted — diverged branches fail safely.
//
// [Runner.RebaseBranch] rebases a branch onto a target (typically the default branch).
// For the current branch, it runs git rebase directly (requires clean worktree).
// For non-checked-out branches, it creates a temporary git worktree, performs the
// rebase there, and removes the worktree — the user's main checkout is never disturbed.
// On failure, the rebase is aborted and the temp worktree is cleaned up.
//
// [Runner.MergeInto] merges a source branch into the current branch.
// Typically used to merge the default branch into the current working branch.
// Requires a clean worktree. On failure, the merge is aborted.
//
// [Runner.RebaseBranchRemote] rebases a branch onto a target and pushes the result
// to the remote. The entire operation runs in a temporary worktree — the user's
// checkout is untouched. The push uses --force-with-lease for safety (fails if the
// remote was updated by someone else since the last fetch).
//
// [Runner.MergeIntoRemote] merges a source branch into a target branch and pushes.
// Also runs in a temporary worktree. Typically used to merge the default branch
// into a feature branch and push the updated feature branch.
//
// [Runner.Compact] runs git gc for standard garbage collection: repacks objects,
// prunes unreachable objects, expires old reflog entries, and updates the commit-graph.
// The timeout is extended to 5 minutes.
//
// [Runner.CompactAggressive] runs git gc --aggressive for deeper optimization with
// higher compression settings. Use when [RepoSize.RepackAdvised] is true.
// The timeout is extended to 10 minutes.
//
// All actions return an [ActionResult] indicating success or failure with a message.
// Actions that touch the current branch guard against a dirty worktree.
package git

// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// RepoInfo holds the consolidated data for a single repository,
// combining git-derived data and optional hosting-platform metadata.
type RepoInfo struct {
	// RootIndex identifies which configured root this repo belongs to.
	// -1 means unknown/unset. The engine uses this to look up per-root
	// config (enabled checks, GitHub settings, etc.).
	RootIndex int

	// Core git data.
	Path           string
	IsGit          bool
	Status         Status
	Branches       []Branch
	Remotes        []Remote
	Stashes        []Stash
	DefaultBranch  string
	SCM            RepoSCM  // github, gitlab, other, no-scm
	Kind           RepoKind // clone, fork, not-git
	LastCommit        time.Time
	LastCommitMessage string    // subject line of the most recent commit on HEAD
	LastLocalUpdate   time.Time // most recent local activity: last commit if clean, newest dirty file mtime if dirty
	CommitCount       int       // total number of commits reachable from HEAD; 0 when unavailable or shallow
	FirstCommit       time.Time // author date of the earliest commit reachable from HEAD; zero when unavailable or shallow
	Worktrees      []Worktree
	IsShallow      bool
	HasSubmodules  bool
	HasLFS         bool
	Tags           []Tag
	LastTagDate    time.Time // most recent tag date (any tag)
	LastSemverTag  string    // latest semver tag by version ordering
	LastSemverDate time.Time // date of LastSemverTag

	// Cache metadata.
	CollectedAt  time.Time    // when this info was last collected (for cache TTL)
	CollectLevel CollectLevel // fast or full (for cache validity)

	// Git diagnostics (optional, nil when not collected).
	Health    *HealthReport
	Size      *RepoSize
	Config    *RepoConfig
	FileStats *FileStats
	Activity  *Activity

	// Platform metadata (optional, nil when not collected).
	// Platform is collected from the origin remote.
	Platform *PlatformInfo

	// UpstreamPlatform is collected from the upstream remote (if present).
	// Used by fork-aware checks that need data about the user's fork.
	UpstreamPlatform *PlatformInfo

	// UpstreamDefaultBehindLocal is true when the upstream remote's default
	// branch is strictly behind the local default branch.
	// Only populated for fork-kind repos that have an upstream remote.
	UpstreamDefaultBehindLocal bool

	// UpstreamDefaultBehindOrigin is true when the upstream remote's default
	// branch is strictly behind the origin remote's default branch.
	// Only populated for fork-kind repos that have both remotes.
	UpstreamDefaultBehindOrigin bool

	// Errors.
	Err      error // fatal collection error
	FetchErr error // non-fatal fetch failure (local data still valid)
}

// NewRepoInfo creates a minimal RepoInfo seeded with a path.
// Use this as input to [ifaces.Engineer.Collect] when no prior data exists.
func NewRepoInfo(pth string) *RepoInfo {
	return &RepoInfo{
		RootIndex: -1,
		Path:      pth,
	}
}

// NewRepoInfoForRoot creates a minimal RepoInfo seeded with a path and root index.
func NewRepoInfoForRoot(pth string, rootIndex int) *RepoInfo {
	return &RepoInfo{
		RootIndex: rootIndex,
		Path:      pth,
	}
}

// NoGit creates a RepoInfo for a non-git directory.
func NoGit(pth string) *RepoInfo {
	return &RepoInfo{
		RootIndex: -1,
		Path:      pth,
		IsGit:     false,
		SCM:       SCMNone,
		Kind:      RepoKindNotGit,
	}
}

// DefaultBranchHash returns the commit hash of the default branch,
// or an empty string if the default branch is not found.
func (r *RepoInfo) DefaultBranchHash() string {
	if r == nil {
		return ""
	}

	for _, b := range r.Branches {
		if !b.IsRemote && b.Name == r.DefaultBranch {
			return b.Hash
		}
	}

	return ""
}

// IsEmpty reports whether the RepoInfo has no data.
func (r *RepoInfo) IsEmpty() bool {
	return r == nil || !r.IsGit
}

// RepoErr returns the fatal error, if any.
func (r *RepoInfo) RepoErr() error {
	if r == nil {
		return nil
	}

	return r.Err
}

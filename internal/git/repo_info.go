package git

import "time"

// RepoInfo holds the git-derived data for a single repository.
type RepoInfo struct {
	Path          string
	IsGit         bool
	Status        Status
	Branches      []Branch
	Remotes       []Remote
	Stashes       []Stash
	DefaultBranch string
	SCM           string // github, gitlab, other, no-scm
	Kind          string // clone, fork, not-git
	LastCommit    time.Time
	Worktrees     []Worktree
	Health        *HealthReport
	IsShallow     bool
	HasSubmodules bool
	HasLFS        bool
	Size          *RepoSize
	Config        *RepoConfig
	FileStats     *FileStats
	Tags          []Tag
	LastTagDate   time.Time // most recent tag date (any tag)
	LastSemverTag string    // latest semver tag by version ordering
	LastSemverDate time.Time // date of LastSemverTag
	Activity      *Activity
	Err           error
}

// IsRepoInfo is a marker method satisfying the engine.RepoInfo interface.
func (*RepoInfo) IsRepoInfo() {}

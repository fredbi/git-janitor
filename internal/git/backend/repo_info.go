package backend

import (
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// RepoInfo holds the git-derived data for a single repository.
type RepoInfo struct {
	Path           string
	IsGit          bool
	Status         Status
	Branches       []Branch
	Remotes        []Remote
	Stashes        []Stash
	DefaultBranch  string
	SCM            models.RepoSCM  // github, gitlab, other, no-scm
	Kind           models.RepoKind // clone, fork, not-git
	LastCommit     time.Time
	Worktrees      []Worktree
	Health         *HealthReport
	IsShallow      bool
	HasSubmodules  bool
	HasLFS         bool
	Size           *RepoSize
	Config         *RepoConfig
	FileStats      *FileStats
	Tags           []Tag
	LastTagDate    time.Time // most recent tag date (any tag)
	LastSemverTag  string    // latest semver tag by version ordering
	LastSemverDate time.Time // date of LastSemverTag
	Activity       *Activity
	Err            error
	// FetchErr records a fetch failure (e.g. remote unavailable).
	// Non-nil when the remote could not be reached, but local data is still valid.
	FetchErr error
}

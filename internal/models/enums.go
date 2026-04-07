package models

type RunnerKind uint8

const (
	RunnerKindGit = iota
	RunnerKindGitHub
)

type RepoKind string

func (e RepoKind) String() string {
	return string(e)
}

const (
	RepoKindNone    RepoKind = ""
	RepoKindClone   RepoKind = "clone"
	RepoKindFork    RepoKind = "fork"
	RepoKindTracked RepoKind = "tracked"
	RepoKindNotGit  RepoKind = "not-git"
)

type RepoSCM string

func (e RepoSCM) String() string {
	return string(e)
}

// SCM provider constants.
const (
	SCMGitHub = "github"
	SCMGitLab = "gitlab"
	SCMOther  = "other"
	SCMNone   = "no-scm"
)

// SubjectKind categorizes what a check or action operates on.
type SubjectKind uint8

const (
	SubjectNone   SubjectKind = iota // no specific subject (repo-level)
	SubjectRepo                      // the repository itself
	SubjectRemote                    // a git remote
	SubjectBranch                    // a git branch
	SubjectStash                     // a git stash entry
	SubjectTag                       // a git tag
	SubjectFile                      // a git file object
)

// String returns the human-readable name of a SubjectKind.
func (s SubjectKind) String() string {
	switch s {
	case SubjectNone:
		return "none"
	case SubjectRepo:
		return "repo"
	case SubjectRemote:
		return "remote"
	case SubjectBranch:
		return "branch"
	case SubjectStash:
		return "stash"
	case SubjectTag:
		return "tag"
	default:
		return "unknown"
	}
}

// Severity levels for alerts. The zero value (SeverityNone) means
// "check ran, nothing wrong.".
type Severity uint8

const (
	SeverityNone     Severity = iota // check passed, no alert
	SeverityInfo                     // informational, no action needed
	SeverityLow                      // minor housekeeping
	SeverityMedium                   // should address soon
	SeverityHigh                     // needs attention now
	SeverityCritical                 // needs manual repair
)

// String returns the human-readable name of a Severity.
func (s Severity) String() string {
	switch s {
	case SeverityNone:
		return "none"
	case SeverityInfo:
		return "info"
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// CheckKind identifies the provider of a check.
type CheckKind uint8

const (
	CheckKindGit    CheckKind = iota // git CLI check
	CheckKindGitHub                  // GitHub API check
	CheckKindGitLab                  // GitLab API check (future)
	CheckKindCustom                  // external/custom check (Phase 3)
)

// ActionKind identifies the provider of an action.
type ActionKind uint8

const (
	ActionKindGit    ActionKind = iota // git CLI action
	ActionKindGitHub                   // GitHub API action
	ActionKindCustom                   // external/custom action (Phase 3)
)

type CollectOption uint8

const (
	CollectNone CollectOption = iota
	CollectFast
	CollectForceRefresh
	CollectSecurityAlerts
	CollectPlatform // collect hosting-platform metadata (GitHub/GitLab API)
)

type EvaluateOption uint8

const (
	EvaluateAll EvaluateOption = iota
)

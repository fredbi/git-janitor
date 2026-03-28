package engine

// SubjectKind categorizes what a check or action operates on.
type SubjectKind uint8

const (
	SubjectNone    SubjectKind = iota // no specific subject (repo-level)
	SubjectRepo                       // the repository itself
	SubjectRemote                     // a git remote
	SubjectBranch                     // a git branch
	SubjectStash                      // a git stash entry
	SubjectTag                        // a git tag
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
// "check ran, nothing wrong."
type Severity uint8

const (
	SeverityNone   Severity = iota // check passed, no alert
	SeverityInfo                   // informational, no action needed
	SeverityLow                    // minor housekeeping
	SeverityMedium                 // should address soon
	SeverityHigh                   // needs attention now
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

// Alert is the outcome of a check. One alert per SubjectKind per check
// invocation: if a check finds 5 lagging branches, that is one alert
// with 5 subject instances spread across the suggestions.
//
// The zero value (Severity == SeverityNone) means "check ran, nothing wrong."
type Alert struct {
	// CheckName is the name of the check that produced this alert.
	CheckName string

	// Severity indicates urgency. SeverityNone means no alert.
	Severity Severity

	// Summary is a one-line human-readable description.
	Summary string

	// Detail is a longer explanation (useful for custom/AI checks,
	// and for documentation during development/testing).
	Detail string

	// Suggestions lists zero or more suggested fix actions.
	Suggestions []ActionSuggestion
}

// ActionSuggestion links an alert to an executable action.
// The ActionName is a key in the ActionRegistry.
type ActionSuggestion struct {
	// ActionName is the registered name of the action to execute.
	ActionName string

	// SubjectKind identifies what kind of thing the subjects are.
	SubjectKind SubjectKind

	// Subjects lists the specific instances to act on
	// (e.g., branch names, tag names).
	Subjects []string
}

// Result holds the outcome of an executed action.
type Result struct {
	// OK is true if the action completed successfully.
	OK bool

	// Message describes what happened (success or failure).
	Message string
}

// Assignment wraps an action suggestion for execution.
// In Phase 1 this is a thin wrapper for synchronous execution.
// Phase 2 will add scheduling state (pending/running/done/failed),
// priority, and timestamps.
type Assignment struct {
	// Suggestion is the action to execute.
	Suggestion ActionSuggestion

	// RepoPath is the repository this assignment targets.
	RepoPath string
}

// RepoInfo is a marker interface for provider-specific repository data.
// Concrete types (git.RepoInfo, github.RepoData) implement this so the
// engine's Evaluate dispatcher can accept both.
type RepoInfo interface {
	// IsRepoInfo is an exported marker method.
	// Concrete types in other packages implement it to satisfy this interface.
	IsRepoInfo()
}

// SelfDescribed is common to checks and actions: provides a name and
// human-readable description for the registry and config wizard.
type SelfDescribed interface {
	Name() string
	Description() string
}

// Check is the interface for all checks, regardless of provider.
// The sealed marker (isCheck) prevents external implementations
// while keeping the registry type-safe.
type Check interface {
	isCheck() // sealed marker — only engine package types may implement

	SelfDescribed
	Kind() CheckKind
}

// Action is the interface for all actions, regardless of provider.
type Action interface {
	isAction() // sealed marker

	SelfDescribed
	Kind() ActionKind
	ApplyTo() SubjectKind // what kind of subject this action operates on
	Destructive() bool    // needs user confirmation
}

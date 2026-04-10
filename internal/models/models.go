package models

import (
	"iter"
	"strings"
)

// Alert is the outcome of a check. One alert per SubjectKind per check
// invocation: if a check finds 5 lagging branches, that is one alert
// with 5 subject instances spread across the suggestions.
//
// The zero value (Severity == SeverityNone) means "check ran, nothing wrong.".
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
	Subjects []ActionSubject
}

func (s ActionSuggestion) SubjectNames() []string {
	subjects := make([]string, 0, len(s.Subjects))
	for _, subject := range s.Subjects {
		subjects = append(subjects, subject.Subject)
	}

	return subjects
}

// SubjectParams returns the params for the subject at index i,
// or nil if no params are set.
func (s ActionSuggestion) SubjectParams() iter.Seq2[string, []string] {
	return func(yield func(string, []string) bool) {
		for _, subject := range s.Subjects {
			if !yield(subject.Subject, subject.Params) {
				return
			}
		}
	}
}

type ActionSubject struct {
	Subject string

	// Params is a parallel slice to Subjects (same length, index-aligned).
	// Each entry carries action-specific parameters for the corresponding subject.
	// nil or shorter than Subjects means no params for those subjects.
	Params []string
}

// Result holds the outcome of an executed action.
type Result struct {
	// OK is true if the action completed successfully.
	OK bool

	// Message describes what happened (success or failure).
	Message string

	// CommandLog records the commands (git CLI or API calls) executed
	// during the action, in order. Each entry is a human-readable
	// command string (e.g. "git stash apply stash@{3}").
	CommandLog []string
}

// Assignment wraps an action suggestion for execution.
//
// In Phase 1 this is a thin wrapper for synchronous execution.
// Phase 2 will add scheduling state (pending/running/done/failed),
// priority, and timestamps.
type Assignment struct {
	// Suggestion is the action to execute.
	Suggestion ActionSuggestion

	// RepoPath is the repository this assignment targets.
	RepoPath string
}

// RepoItem represents a repository entry in the list.
//
// Namespace is the slash-separated relative parent directory from the
// configured root, used to express GitLab-style nested groups. It is
// empty for top-level repositories (the GitHub-style flat layout).
type RepoItem struct {
	Path      string
	Name      string
	Namespace string // slash-separated parent path relative to the root
	IsGit     bool   // true if a .git directory was found
}

// Title implements the list.DefaultItem interface.
//
// Title returns just the leaf name; the panel delegate is responsible
// for indenting the row according to [RepoItem.Depth].
func (i RepoItem) Title() string {
	if !i.IsGit {
		return i.Name + " (not git)"
	}

	return i.Name
}

// Description implements the list.DefaultItem interface.
func (i RepoItem) Description() string { return i.Path }

// FilterValue implements the list.Item interface. It includes the
// namespace so the regexp filter naturally matches "group/sub/repo".
func (i RepoItem) FilterValue() string {
	if i.Namespace != "" {
		return i.Namespace + "/" + i.Name
	}

	return i.Name
}

// Depth returns the nesting level of the repo within its root: 0 for a
// top-level repo, 1 for "group/repo", 2 for "group/sub/repo", etc.
func (i RepoItem) Depth() int {
	if i.Namespace == "" {
		return 0
	}

	return strings.Count(i.Namespace, "/") + 1
}

// DisplayKey returns a stable, lowercased sort key combining the
// namespace and the leaf name. Sorting by this key clusters siblings
// under the same group regardless of [os.ReadDir] order.
func (i RepoItem) DisplayKey() string {
	if i.Namespace == "" {
		return strings.ToLower(i.Name)
	}

	return strings.ToLower(i.Namespace + "/" + i.Name)
}

package engine

import (
	"errors"
)

// GitHubAction is the base struct for all GitHub actions.
// Concrete GitHub actions embed this and override Execute.
type GitHubAction struct {
	Describer
}

func (GitHubAction) isAction()            {}
func (GitHubAction) Kind() ActionKind     { return ActionKindGitHub }
func (GitHubAction) Destructive() bool    { return false }
func (GitHubAction) ApplyTo() SubjectKind { return SubjectNone }

// Execute is the default implementation that returns "not implemented".
func (GitHubAction) Execute(_ *GitHubRepoData, _ []string) (Result, error) {
	return Result{}, errors.New("not implemented")
}

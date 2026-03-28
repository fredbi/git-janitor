package engine

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/git"
)

// GitAction is the base struct for all git actions.
// Concrete git actions embed this and override Execute.
type GitAction struct {
	Describer
}

func (GitAction) isAction()            {}
func (GitAction) Kind() ActionKind     { return ActionKindGit }
func (GitAction) Destructive() bool    { return false }
func (GitAction) ApplyTo() SubjectKind { return SubjectNone }

// Execute is the default implementation that returns "not implemented".
// Concrete actions override this method.
// The subjects parameter contains the specific instances to act on
// (e.g., branch names), as provided by ActionSuggestion.Subjects.
func (GitAction) Execute(_ context.Context, _ *git.Runner, _ *git.RepoInfo, _ []string) (Result, error) {
	return Result{}, errors.New("not implemented")
}

package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Action = githubAction{}

// githubAction is the base struct for all git actions.
//
// Concrete git actions embed this and override Execute.
type githubAction struct {
	models.Describer
}

func (githubAction) IsAction()                   {}
func (githubAction) Kind() models.ActionKind     { return models.ActionKindGitHub }
func (githubAction) Destructive() bool           { return false }
func (githubAction) ApplyTo() models.SubjectKind { return models.SubjectNone }
func (githubAction) ParamPrompt() string         { return "" }

// Execute is the default implementation that returns "not implemented".
// Concrete actions override this method.
// The subjects parameter contains the specific instances to act on
// (e.g., branch names), as provided by models.ActionSuggestion.Subjects.
func (githubAction) Execute(_ context.Context, _ *models.RepoInfo, _ []string) (models.Result, error) {
	return models.Result{}, errors.New("not implemented")
}

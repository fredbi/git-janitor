package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Action = gitAction{}

// gitAction is the base struct for all git actions.
//
// Concrete git actions embed this and override Execute.
type gitAction struct {
	models.Describer
}

func (gitAction) IsAction()                   {}
func (gitAction) Kind() models.ActionKind     { return models.ActionKindGit }
func (gitAction) Destructive() bool           { return false }
func (gitAction) ApplyTo() models.SubjectKind { return models.SubjectNone }
func (gitAction) ParamPrompt() string         { return "" }

// Execute is the default implementation that returns "not implemented".
// Concrete actions override this method.
// The subjects parameter contains the specific instances to act on
// (e.g., branch names), as provided by models.ActionSuggestion.Subjects.
func (gitAction) Execute(_ context.Context, _ *models.RepoInfo, _ []string) (models.Result, error) {
	return models.Result{}, errors.New("not implemented")
}

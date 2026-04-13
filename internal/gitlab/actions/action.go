// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Action = gitlabAction{}

// gitlabAction is the base struct for all GitLab actions.
//
// Concrete GitLab actions embed this and override Execute.
type gitlabAction struct {
	models.Describer
}

func (gitlabAction) IsAction()                   {}
func (gitlabAction) Kind() models.ActionKind     { return models.ActionKindGitLab }
func (gitlabAction) Destructive() bool           { return false }
func (gitlabAction) ApplyTo() models.SubjectKind { return models.SubjectNone }
func (gitlabAction) ParamPrompt() string         { return "" }

// Execute is the default implementation that returns "not implemented".
// Concrete actions override this method.
func (gitlabAction) Execute(_ context.Context, _ *models.RepoInfo, _ []string) (models.Result, error) {
	return models.Result{}, errors.New("not implemented")
}

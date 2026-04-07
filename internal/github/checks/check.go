package checks

import (
	"context"
	"errors"
	"iter"

	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Check = githubCheck{}

// githubCheck is the base struct for all github checks.
//
// Concrete github checks embed this and override Evaluate.
type githubCheck struct {
	models.Describer
}

func (githubCheck) IsCheck()               {}
func (githubCheck) Kind() models.CheckKind { return models.CheckKindGitHub }

// Evaluate is the default implementation that returns "not implemented".
//
// Concrete checks override this method.
func (githubCheck) Evaluate(_ context.Context, _ *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return nil, errors.New("not implemented")
}

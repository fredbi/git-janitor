package checks

import (
	"context"
	"errors"
	"iter"

	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Check = gitCheck{}

// gitCheck is the base struct for all git checks.
//
// Concrete git checks embed this and override Evaluate.
type gitCheck struct {
	models.Describer
}

func (gitCheck) Kind() models.CheckKind { return models.CheckKindGit }

// Evaluate is the default implementation that returns "not implemented".
//
// Concrete checks override this method.
func (gitCheck) Evaluate(_ context.Context) (iter.Seq[models.Alert], error) {
	return nil, errors.New("not implemented")
}

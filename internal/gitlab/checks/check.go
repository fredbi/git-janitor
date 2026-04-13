// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"errors"
	"iter"

	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Check = gitlabCheck{}

// gitlabCheck is the base struct for all GitLab checks.
//
// Concrete GitLab checks embed this and override Evaluate.
type gitlabCheck struct {
	models.Describer
}

func (gitlabCheck) IsCheck()               {}
func (gitlabCheck) Kind() models.CheckKind { return models.CheckKindGitLab }

// Evaluate is the default implementation that returns "not implemented".
//
// Concrete checks override this method.
func (gitlabCheck) Evaluate(_ context.Context, _ *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return nil, errors.New("not implemented")
}

// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// SetRepoDescription updates the repository description on GitHub.
//
// The description is taken from the "description" param of the first subject.
type SetRepoDescription struct {
	githubAction
}

func NewSetRepoDescription() SetRepoDescription {
	return SetRepoDescription{
		githubAction: githubAction{
			Describer: models.NewDescriber(
				"set-repo-description",
				"set the repository description on GitHub",
			),
		},
	}
}

func (SetRepoDescription) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a SetRepoDescription) Execute(ctx context.Context, repoInfo *models.RepoInfo, subjects []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, repoInfo.Platform, subjects)
}

func (a SetRepoDescription) execute(ctx context.Context, client *backend.Runner, data *models.PlatformInfo, params []string) (models.Result, error) {
	if data == nil {
		return models.Result{}, errors.New("backend repo data is required")
	}

	// Get description from params.
	var description string
	if len(params) > 0 {
		trimmed := strings.TrimSpace(params[0])
		if len(trimmed) > 0 {
			description = trimmed
		}
	}

	if description == "" {
		return models.Result{}, errors.New("no description provided (set 'description' param)")
	}

	if err := client.SetDescription(ctx, data.Owner, data.Repo, description); err != nil {
		return models.Result{
			OK:      false,
			Message: fmt.Sprintf("failed to set description on %s: %v", data.FullName, err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: "set description on " + data.FullName,
	}, nil
}

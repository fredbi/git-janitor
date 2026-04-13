// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// SetProjectDescription updates the project description on GitLab.
//
// The description is taken from the first subject parameter.
type SetProjectDescription struct {
	gitlabAction
}

// NewSetProjectDescription creates a new SetProjectDescription action.
func NewSetProjectDescription() SetProjectDescription {
	return SetProjectDescription{
		gitlabAction: gitlabAction{
			Describer: models.NewDescriber(
				"gitlab-set-project-description",
				"set the project description on GitLab",
			),
		},
	}
}

func (SetProjectDescription) ApplyTo() models.SubjectKind { return models.SubjectRepo }
func (SetProjectDescription) ParamPrompt() string         { return "Description:" }

func (a SetProjectDescription) Execute(ctx context.Context, repoInfo *models.RepoInfo, subjects []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	data := repoInfo.Platform
	if data.ProjectID == 0 {
		return models.Result{}, errors.New("no GitLab project ID available")
	}

	// Get description from params.
	var description string
	if len(subjects) > 0 {
		trimmed := strings.TrimSpace(subjects[0])
		if len(trimmed) > 0 {
			description = trimmed
		}
	}

	if description == "" {
		return models.Result{}, errors.New("no description provided (set 'description' param)")
	}

	if err := runner.SetDescription(ctx, data.ProjectID, description); err != nil {
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

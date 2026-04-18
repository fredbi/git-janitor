// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/fredbi/git-janitor/internal/models"
)

// CloseIssue closes the GitHub issue identified by the first subject
// (issue number). Bound to the X key on the issue detail popup.
//
// This goes through the standard Y/N confirmation flow because the action
// is destructive (state change visible to other collaborators).
type CloseIssue struct {
	githubAction
}

func NewCloseIssue() CloseIssue {
	return CloseIssue{
		githubAction: githubAction{
			Describer: models.NewDescriber(
				"close-issue",
				"close the GitHub issue",
			),
		},
	}
}

func (CloseIssue) ApplyTo() models.SubjectKind { return models.SubjectIssueDetail }
func (CloseIssue) Destructive() bool           { return true }

func (a CloseIssue) Execute(ctx context.Context, repoInfo *models.RepoInfo, subjects []string) (models.Result, error) {
	if repoInfo == nil || repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	if len(subjects) == 0 {
		return models.Result{}, errors.New("no issue number provided")
	}

	number, err := strconv.Atoi(subjects[0])
	if err != nil || number <= 0 {
		return models.Result{}, fmt.Errorf("invalid issue number: %q", subjects[0])
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	data := repoInfo.Platform

	if err := runner.CloseIssue(ctx, data.Owner, data.Repo, number); err != nil {
		return models.Result{
			Message: fmt.Sprintf("failed to close %s#%d: %v", data.FullName, number, err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: fmt.Sprintf("closed %s#%d", data.FullName, number),
	}, nil
}

// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/fredbi/git-janitor/internal/models"
)

// SelfAssignIssue assigns the issue identified by the first subject (issue
// number) to the GitHub user holding the current access token. Useful when
// triaging: press S on an issue's detail popup and claim it.
//
// GitHub silently drops logins it refuses to assign (e.g. the token holder
// is not a collaborator with at least triage permission on the repository),
// so the action succeeds as long as the API call does not error — the user
// is expected to check the repo to confirm the assignment landed.
type SelfAssignIssue struct {
	githubAction
}

func NewSelfAssignIssue() SelfAssignIssue {
	return SelfAssignIssue{
		githubAction: githubAction{
			Describer: models.NewDescriber(
				"self-assign-issue",
				"assign the issue to the authenticated GitHub user",
			),
		},
	}
}

func (SelfAssignIssue) ApplyTo() models.SubjectKind { return models.SubjectIssueDetail }
func (SelfAssignIssue) Destructive() bool           { return true }

func (a SelfAssignIssue) Execute(ctx context.Context, repoInfo *models.RepoInfo, subjects []string) (models.Result, error) {
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

	login, err := runner.AuthenticatedLogin(ctx)
	if err != nil {
		return models.Result{
			Message: fmt.Sprintf("cannot resolve authenticated GitHub user: %v", err),
		}, err
	}

	data := repoInfo.Platform

	if err := runner.AddIssueAssignees(ctx, data.Owner, data.Repo, number, []string{login}); err != nil {
		return models.Result{
			Message: fmt.Sprintf("failed to assign %s to %s#%d: %v", login, data.FullName, number, err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: fmt.Sprintf("assigned %s to %s#%d", login, data.FullName, number),
	}, nil
}

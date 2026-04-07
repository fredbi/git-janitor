// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"

	"github.com/fredbi/git-janitor/internal/models"
	gogithub "github.com/google/go-github/v72/github"
)

// ListWorkflowRuns fetches a page of recent workflow runs.
// Returns the runs, whether more pages exist, and any error.
func (c *Client) ListWorkflowRuns(ctx context.Context, owner, repo string, page, perPage int) ([]models.WorkflowRun, bool, error) {
	runs, resp, err := c.gh.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, &gogithub.ListWorkflowRunsOptions{
		ListOptions: gogithub.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	})
	c.updateRate(resp)

	if err != nil {
		return nil, false, err
	}

	result := make([]models.WorkflowRun, 0, len(runs.WorkflowRuns))

	for _, gh := range runs.WorkflowRuns {
		result = append(result, convertWorkflowRun(gh))
	}

	hasMore := resp != nil && resp.NextPage > 0

	return result, hasMore, nil
}

// GetWorkflowRunDetail fetches full detail for a single workflow run.
func (c *Client) GetWorkflowRunDetail(ctx context.Context, owner, repo string, runID int64) *models.WorkflowRunDetail {
	gh, resp, err := c.gh.Actions.GetWorkflowRunByID(ctx, owner, repo, runID)
	c.updateRate(resp)

	if err != nil {
		return nil
	}

	detail := &models.WorkflowRunDetail{
		RunNumber:  gh.GetRunNumber(),
		RunAttempt: gh.GetRunAttempt(),
	}

	// Compute duration from start to update time.
	if start := gh.GetRunStartedAt(); !start.IsZero() {
		if upd := gh.GetUpdatedAt(); !upd.IsZero() {
			detail.Duration = upd.Sub(start.Time)
		}
	}

	return detail
}

func convertWorkflowRun(gh *gogithub.WorkflowRun) models.WorkflowRun {
	run := models.WorkflowRun{
		ID:         gh.GetID(),
		Name:       gh.GetName(),
		Status:     gh.GetStatus(),
		Conclusion: gh.GetConclusion(),
		Branch:     gh.GetHeadBranch(),
		Event:      gh.GetEvent(),
		HTMLURL:    gh.GetHTMLURL(),
	}

	if t := gh.GetCreatedAt(); !t.IsZero() {
		run.CreatedAt = t.Time
	}

	return run
}

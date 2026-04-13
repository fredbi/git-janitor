// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ListPipelines fetches a page of recent pipelines.
// Returns the pipelines (as WorkflowRuns), whether more pages exist, and any error.
func (c *Client) ListPipelines(ctx context.Context, projectID, page, perPage int) ([]models.WorkflowRun, bool, error) {
	glPipelines, resp, err := c.gl.Pipelines.ListProjectPipelines(projectID, &gitlab.ListProjectPipelinesOptions{
		OrderBy: gitlab.Ptr("updated_at"),
		Sort:    gitlab.Ptr("desc"),
		ListOptions: gitlab.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}, gitlab.WithContext(ctx))
	c.updateRate(resp)

	if err != nil {
		return nil, false, err
	}

	runs := make([]models.WorkflowRun, 0, len(glPipelines))

	for _, gl := range glPipelines {
		runs = append(runs, convertPipeline(gl))
	}

	hasMore := resp != nil && resp.NextPage > 0

	return runs, hasMore, nil
}

// GetPipelineDetail fetches full detail for a single pipeline.
func (c *Client) GetPipelineDetail(ctx context.Context, projectID, pipelineID int) *models.WorkflowRunDetail {
	gl, resp, err := c.gl.Pipelines.GetPipeline(projectID, pipelineID, gitlab.WithContext(ctx))
	c.updateRate(resp)

	if err != nil {
		return nil
	}

	detail := &models.WorkflowRunDetail{
		RunNumber: gl.ID,
	}

	// Compute duration.
	if gl.StartedAt != nil && gl.FinishedAt != nil {
		detail.Duration = gl.FinishedAt.Sub(*gl.StartedAt)
	} else if gl.Duration > 0 {
		detail.Duration = time.Duration(gl.Duration) * time.Second
	}

	return detail
}

func convertPipeline(gl *gitlab.PipelineInfo) models.WorkflowRun {
	run := models.WorkflowRun{
		ID:     int64(gl.ID),
		Name:   gl.Source,
		Status: gl.Status,
		Branch: gl.Ref,
		Event:  gl.Source,
	}

	// GitLab pipeline status maps to conclusion for completed states.
	switch gl.Status {
	case "success":
		run.Conclusion = "success"
	case "failed":
		run.Conclusion = "failure"
	case "canceled":
		run.Conclusion = "cancelled"
	case "skipped":
		run.Conclusion = "skipped"
	}

	if gl.WebURL != "" {
		run.HTMLURL = gl.WebURL
	}

	if gl.CreatedAt != nil {
		run.CreatedAt = *gl.CreatedAt
	}

	return run
}

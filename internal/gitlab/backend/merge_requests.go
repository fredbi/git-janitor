// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"

	"github.com/fredbi/git-janitor/internal/models"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ListMergeRequests fetches a page of open merge requests sorted by last updated (descending).
// Returns the MRs (as PullRequests), whether more pages exist, and any error.
func (c *Client) ListMergeRequests(ctx context.Context, projectID, page, perPage int) ([]models.PullRequest, bool, error) {
	glMRs, resp, err := c.gl.MergeRequests.ListProjectMergeRequests(projectID, &gitlab.ListProjectMergeRequestsOptions{
		State:   gitlab.Ptr("opened"),
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

	prs := make([]models.PullRequest, 0, len(glMRs))

	for _, gl := range glMRs {
		prs = append(prs, convertMergeRequest(gl))
	}

	hasMore := resp != nil && resp.NextPage > 0

	return prs, hasMore, nil
}

// GetMergeRequestDetail fetches full detail for a single merge request.
func (c *Client) GetMergeRequestDetail(ctx context.Context, projectID, mrIID int) *models.PullRequestDetail {
	gl, resp, err := c.gl.MergeRequests.GetMergeRequest(projectID, mrIID, &gitlab.GetMergeRequestsOptions{}, gitlab.WithContext(ctx))
	c.updateRate(resp)

	if err != nil {
		return nil
	}

	detail := &models.PullRequestDetail{
		Body:         gl.Description,
		CommentCount: gl.UserNotesCount,
		Mergeable:    gl.DetailedMergeStatus == "mergeable",
	}

	// Compute changed file stats from the merge request changes.
	if gl.ChangesCount != "" {
		// ChangesCount is a string like "5" in GitLab API.
		// We don't get per-file additions/deletions from the basic MR response.
	}

	for _, l := range gl.Labels {
		detail.Tags = append(detail.Tags, l)
	}

	// Determine review state.
	if gl.Draft {
		detail.ReviewState = "draft"
	} else if gl.DetailedMergeStatus == "checking" {
		detail.ReviewState = "pending"
	}

	return detail
}

func convertMergeRequest(gl *gitlab.BasicMergeRequest) models.PullRequest {
	pr := models.PullRequest{
		Number:  gl.IID,
		Title:   gl.Title,
		State:   gl.State,
		Author:  gl.Author.Username,
		Draft:   gl.Draft,
		Branch:  gl.SourceBranch,
		Base:    gl.TargetBranch,
		HTMLURL: gl.WebURL,
	}

	if gl.CreatedAt != nil {
		pr.CreatedAt = *gl.CreatedAt
	}

	if gl.UpdatedAt != nil {
		pr.UpdatedAt = *gl.UpdatedAt
	}

	return pr
}

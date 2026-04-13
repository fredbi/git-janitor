// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"

	"github.com/fredbi/git-janitor/internal/models"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ListIssues fetches a page of open issues sorted by last updated (descending).
// Returns the issues, whether more pages exist, and any error.
func (c *Client) ListIssues(ctx context.Context, projectID, page, perPage int) ([]models.Issue, bool, error) {
	glIssues, resp, err := c.gl.Issues.ListProjectIssues(projectID, &gitlab.ListProjectIssuesOptions{
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

	issues := make([]models.Issue, 0, len(glIssues))

	for _, gl := range glIssues {
		issues = append(issues, convertIssue(gl))
	}

	hasMore := resp != nil && resp.NextPage > 0

	return issues, hasMore, nil
}

// GetIssueDetail fetches full detail for a single issue.
func (c *Client) GetIssueDetail(ctx context.Context, projectID, issueIID int) *models.IssueDetail {
	gl, resp, err := c.gl.Issues.GetIssue(projectID, issueIID, gitlab.WithContext(ctx))
	c.updateRate(resp)

	if err != nil {
		return nil
	}

	detail := &models.IssueDetail{
		Body:         gl.Description,
		CommentCount: gl.UserNotesCount,
	}

	for _, a := range gl.Assignees {
		detail.Assignees = append(detail.Assignees, a.Username)
	}

	for _, l := range gl.Labels {
		detail.Tags = append(detail.Tags, l)
	}

	return detail
}

func convertIssue(gl *gitlab.Issue) models.Issue {
	issue := models.Issue{
		Number:  gl.IID,
		Title:   gl.Title,
		State:   gl.State,
		Author:  gl.Author.Username,
		HTMLURL: gl.WebURL,
	}

	if gl.CreatedAt != nil {
		issue.CreatedAt = *gl.CreatedAt
	}

	if gl.UpdatedAt != nil {
		issue.UpdatedAt = *gl.UpdatedAt
	}

	issue.Labels = append(issue.Labels, gl.Labels...)

	return issue
}

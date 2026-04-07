// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"

	"github.com/fredbi/git-janitor/internal/models"
	gogithub "github.com/google/go-github/v72/github"
)

// ListIssues fetches a page of open issues sorted by last updated (descending).
// Returns the issues, whether more pages exist, and any error.
// The GitHub API includes PRs in the issues endpoint — we filter them out.
func (c *Client) ListIssues(ctx context.Context, owner, repo string, page, perPage int) ([]models.Issue, bool, error) {
	ghIssues, resp, err := c.gh.Issues.ListByRepo(ctx, owner, repo, &gogithub.IssueListByRepoOptions{
		State:     "open",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: gogithub.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	})
	c.updateRate(resp)

	if err != nil {
		return nil, false, err
	}

	var issues []models.Issue

	for _, gh := range ghIssues {
		if gh.PullRequestLinks != nil {
			continue
		}

		issues = append(issues, convertIssue(gh))
	}

	hasMore := resp != nil && resp.NextPage > 0

	return issues, hasMore, nil
}

// GetIssueDetail fetches full detail for a single issue.
func (c *Client) GetIssueDetail(ctx context.Context, owner, repo string, number int) *models.IssueDetail {
	gh, resp, err := c.gh.Issues.Get(ctx, owner, repo, number)
	c.updateRate(resp)

	if err != nil {
		return nil
	}

	detail := &models.IssueDetail{
		Body:         gh.GetBody(),
		CommentCount: gh.GetComments(),
	}

	for _, a := range gh.Assignees {
		detail.Assignees = append(detail.Assignees, a.GetLogin())
	}

	for _, l := range gh.Labels {
		detail.Tags = append(detail.Tags, l.GetName())
	}

	return detail
}

func convertIssue(gh *gogithub.Issue) models.Issue {
	issue := models.Issue{
		Number:  gh.GetNumber(),
		Title:   gh.GetTitle(),
		State:   gh.GetState(),
		Author:  gh.GetUser().GetLogin(),
		HTMLURL: gh.GetHTMLURL(),
	}

	if t := gh.GetCreatedAt(); !t.IsZero() {
		issue.CreatedAt = t.Time
	}

	if t := gh.GetUpdatedAt(); !t.IsZero() {
		issue.UpdatedAt = t.Time
	}

	for _, l := range gh.Labels {
		issue.Labels = append(issue.Labels, l.GetName())
	}

	return issue
}

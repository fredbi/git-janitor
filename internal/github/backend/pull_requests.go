// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"

	"github.com/fredbi/git-janitor/internal/models"
	gogithub "github.com/google/go-github/v72/github"
)

// ListPullRequests fetches a page of open pull requests sorted by last updated (descending).
// Returns the PRs, whether more pages exist, and any error.
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, page, perPage int) ([]models.PullRequest, bool, error) {
	ghPRs, resp, err := c.gh.PullRequests.List(ctx, owner, repo, &gogithub.PullRequestListOptions{
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

	prs := make([]models.PullRequest, 0, len(ghPRs))

	for _, gh := range ghPRs {
		prs = append(prs, convertPullRequest(gh))
	}

	hasMore := resp != nil && resp.NextPage > 0

	return prs, hasMore, nil
}

// GetPullRequestDetail fetches full detail for a single pull request.
func (c *Client) GetPullRequestDetail(ctx context.Context, owner, repo string, number int) *models.PullRequestDetail {
	gh, resp, err := c.gh.PullRequests.Get(ctx, owner, repo, number)
	c.updateRate(resp)

	if err != nil {
		return nil
	}

	detail := &models.PullRequestDetail{
		Body:         gh.GetBody(),
		CommentCount: gh.GetComments(),
		Mergeable:    gh.GetMergeable(),
		Additions:    gh.GetAdditions(),
		Deletions:    gh.GetDeletions(),
		ChangedFiles: gh.GetChangedFiles(),
	}

	for _, l := range gh.Labels {
		detail.Tags = append(detail.Tags, l.GetName())
	}

	// Determine review state from requested reviewers.
	if gh.GetDraft() {
		detail.ReviewState = "draft"
	} else if len(gh.RequestedReviewers) > 0 {
		detail.ReviewState = "pending"
	}

	return detail
}

func convertPullRequest(gh *gogithub.PullRequest) models.PullRequest {
	pr := models.PullRequest{
		Number:  gh.GetNumber(),
		Title:   gh.GetTitle(),
		State:   gh.GetState(),
		Author:  gh.GetUser().GetLogin(),
		Draft:   gh.GetDraft(),
		HTMLURL: gh.GetHTMLURL(),
	}

	if head := gh.GetHead(); head != nil {
		pr.Branch = head.GetRef()
	}

	if base := gh.GetBase(); base != nil {
		pr.Base = base.GetRef()
	}

	if t := gh.GetCreatedAt(); !t.IsZero() {
		pr.CreatedAt = t.Time
	}

	if t := gh.GetUpdatedAt(); !t.IsZero() {
		pr.UpdatedAt = t.Time
	}

	return pr
}

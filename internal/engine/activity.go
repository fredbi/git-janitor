// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"log"
	"strconv"
	"strings"

	githubbackend "github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

const defaultPerPage = 20

// collectIssueList fetches a page of issues from the GitHub API
// and stores them on PlatformInfo.
func (e *Interactive) collectIssueList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	client, platform := e.githubClientForPlatform(info)
	if client == nil || platform == nil {
		return
	}

	page, perPage := parsePagination(scope)

	issues, _, err := client.ListIssues(ctx, platform.Owner, platform.Repo, page, perPage)
	if err != nil {
		log.Printf("activity: list issues: %v", err)

		return
	}

	if page <= 1 {
		platform.Issues = issues
	} else {
		platform.Issues = append(platform.Issues, issues...)
	}
}

// collectPullRequestList fetches a page of pull requests from the GitHub API.
func (e *Interactive) collectPullRequestList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	client, platform := e.githubClientForPlatform(info)
	if client == nil || platform == nil {
		return
	}

	page, perPage := parsePagination(scope)

	prs, _, err := client.ListPullRequests(ctx, platform.Owner, platform.Repo, page, perPage)
	if err != nil {
		log.Printf("activity: list pull requests: %v", err)

		return
	}

	if page <= 1 {
		platform.PullRequests = prs
	} else {
		platform.PullRequests = append(platform.PullRequests, prs...)
	}
}

// collectIssueDetail fetches full detail for a single issue identified by
// its number (first subject in the scope) and attaches it to the matching
// entry in PlatformInfo.Issues. The list is assumed to already be
// populated — the detail lookup piggybacks on the cached summary so the
// UI can show both metadata (author, dates) and body in one popup.
func (e *Interactive) collectIssueDetail(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	client, platform := e.githubClientForPlatform(info)
	if client == nil || platform == nil || len(scope.Subjects) == 0 {
		return
	}

	number, err := strconv.Atoi(scope.Subjects[0].Subject)
	if err != nil || number <= 0 {
		return
	}

	// Find the summary entry so we can attach the Detail in place.
	var target *models.Issue

	for i := range platform.Issues {
		if platform.Issues[i].Number == number {
			target = &platform.Issues[i]

			break
		}
	}

	if target == nil {
		return
	}

	if target.Detail != nil {
		return // already fetched
	}

	detail := client.GetIssueDetail(ctx, platform.Owner, platform.Repo, number)
	if detail == nil {
		return
	}

	target.Detail = detail
}

// collectWorkflowRunList fetches a page of workflow runs from the GitHub API.
func (e *Interactive) collectWorkflowRunList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	client, platform := e.githubClientForPlatform(info)
	if client == nil || platform == nil {
		return
	}

	page, perPage := parsePagination(scope)

	runs, _, err := client.ListWorkflowRuns(ctx, platform.Owner, platform.Repo, page, perPage)
	if err != nil {
		log.Printf("activity: list workflow runs: %v", err)

		return
	}

	if page <= 1 {
		platform.WorkflowRuns = runs
	} else {
		platform.WorkflowRuns = append(platform.WorkflowRuns, runs...)
	}
}

// githubClientForPlatform returns the GitHub client and the origin PlatformInfo.
// Returns nil, nil if GitHub is not available or no platform data exists.
func (e *Interactive) githubClientForPlatform(info *models.RepoInfo) (*githubbackend.Client, *models.PlatformInfo) {
	if info.Platform == nil {
		return nil, nil
	}

	client := e.getGitHubClient()
	if client == nil || !client.Available() {
		return nil, nil
	}

	return client, info.Platform
}

// parsePagination extracts page and perPage from the scope's first subject params.
// Params are key=value strings: "page=2", "per_page=20".
func parsePagination(scope models.ActionSuggestion) (page, perPage int) {
	page = 1
	perPage = defaultPerPage

	if len(scope.Subjects) == 0 {
		return page, perPage
	}

	for _, param := range scope.Subjects[0].Params {
		k, v, ok := strings.Cut(param, "=")
		if !ok {
			continue
		}

		switch k {
		case "page":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				page = n
			}
		case "per_page":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				perPage = n
			}
		}
	}

	return page, perPage
}


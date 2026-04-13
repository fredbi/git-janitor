// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"log"
	"strconv"
	"strings"

	githubbackend "github.com/fredbi/git-janitor/internal/github/backend"
	gitlabbackend "github.com/fredbi/git-janitor/internal/gitlab/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

const defaultPerPage = 20

// collectIssueList fetches a page of issues from the platform API
// and stores them on PlatformInfo.
func (e *Interactive) collectIssueList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	switch info.SCM {
	case models.SCMGitHub:
		e.collectGitHubIssueList(ctx, info, scope)
	case models.SCMGitLab:
		e.collectGitLabIssueList(ctx, info, scope)
	}
}

// collectPullRequestList fetches a page of pull requests / merge requests from the platform API.
func (e *Interactive) collectPullRequestList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	switch info.SCM {
	case models.SCMGitHub:
		e.collectGitHubPullRequestList(ctx, info, scope)
	case models.SCMGitLab:
		e.collectGitLabMergeRequestList(ctx, info, scope)
	}
}

// collectWorkflowRunList fetches a page of workflow runs / pipelines from the platform API.
func (e *Interactive) collectWorkflowRunList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	switch info.SCM {
	case models.SCMGitHub:
		e.collectGitHubWorkflowRunList(ctx, info, scope)
	case models.SCMGitLab:
		e.collectGitLabPipelineList(ctx, info, scope)
	}
}

// --- GitHub activity ---

// collectGitHubIssueList fetches a page of issues from the GitHub API.
func (e *Interactive) collectGitHubIssueList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
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

// collectGitHubPullRequestList fetches a page of pull requests from the GitHub API.
func (e *Interactive) collectGitHubPullRequestList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
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

// collectGitHubWorkflowRunList fetches a page of workflow runs from the GitHub API.
func (e *Interactive) collectGitHubWorkflowRunList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
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

// --- GitLab activity ---

// collectGitLabIssueList fetches a page of issues from the GitLab API.
func (e *Interactive) collectGitLabIssueList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	client, platform := e.gitlabClientForPlatform(info)
	if client == nil || platform == nil {
		return
	}

	page, perPage := parsePagination(scope)

	issues, _, err := client.ListIssues(ctx, platform.ProjectID, page, perPage)
	if err != nil {
		log.Printf("activity: list gitlab issues: %v", err)

		return
	}

	if page <= 1 {
		platform.Issues = issues
	} else {
		platform.Issues = append(platform.Issues, issues...)
	}
}

// collectGitLabMergeRequestList fetches a page of merge requests from the GitLab API.
func (e *Interactive) collectGitLabMergeRequestList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	client, platform := e.gitlabClientForPlatform(info)
	if client == nil || platform == nil {
		return
	}

	page, perPage := parsePagination(scope)

	prs, _, err := client.ListMergeRequests(ctx, platform.ProjectID, page, perPage)
	if err != nil {
		log.Printf("activity: list merge requests: %v", err)

		return
	}

	if page <= 1 {
		platform.PullRequests = prs
	} else {
		platform.PullRequests = append(platform.PullRequests, prs...)
	}
}

// collectGitLabPipelineList fetches a page of pipelines from the GitLab API.
func (e *Interactive) collectGitLabPipelineList(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) {
	client, platform := e.gitlabClientForPlatform(info)
	if client == nil || platform == nil {
		return
	}

	page, perPage := parsePagination(scope)

	runs, _, err := client.ListPipelines(ctx, platform.ProjectID, page, perPage)
	if err != nil {
		log.Printf("activity: list pipelines: %v", err)

		return
	}

	if page <= 1 {
		platform.WorkflowRuns = runs
	} else {
		platform.WorkflowRuns = append(platform.WorkflowRuns, runs...)
	}
}

// gitlabClientForPlatform returns the GitLab client and the origin PlatformInfo.
// Returns nil, nil if GitLab is not available or no platform data exists.
func (e *Interactive) gitlabClientForPlatform(info *models.RepoInfo) (*gitlabbackend.Client, *models.PlatformInfo) {
	if info.Platform == nil {
		return nil, nil
	}

	baseURL := info.Platform.WebURL

	client := e.getGitLabClient(baseURL)
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

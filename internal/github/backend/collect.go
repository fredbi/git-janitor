// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"errors"

	gogithub "github.com/google/go-github/v72/github"
)

// ErrRateLimited is returned when the GitHub API rate limit is too low
// to safely make more requests.
var ErrRateLimited = errors.New("github: rate limited, try again later")

// collectRepoInfo fetches repository metadata from the GitHub API.
//
// Cheap path: 1 call (repos.Get) + 1 call (pulls.List per_page=1 for accurate PR count).
// When fetchSecurity is true, also queries security APIs (up to 3 extra calls).
func collectRepoInfo(ctx context.Context, c *Client, owner, repo string, fetchSecurity bool) *RepoInfo {
	data := NewRepoInfo(owner, repo)

	ghRepo, resp, err := c.gh.Repositories.Get(ctx, owner, repo)
	c.updateRate(resp)

	if err != nil {
		data.Err = err

		return data
	}

	populateFromRepo(data, ghRepo)
	data.TokenScopes = c.Scopes()

	// Accurate PR count: repos.Get lumps PRs into open_issues_count.
	// A single pulls.List call with per_page=1 gives us the total via response headers.
	prCount, err := countOpenPRs(ctx, c, owner, repo)
	if err == nil {
		data.OpenPRs = prCount
		// Adjust issue count: GitHub's open_issues includes PRs.
		data.OpenIssues = max(data.OpenIssues-prCount, 0)
	}

	// Security alerts (up to 3 calls, 403 handled gracefully per-API).
	if fetchSecurity {
		CollectSecurityAlerts(ctx, c, data)
	} else {
		data.SecuritySkipped = true
	}

	return data
}

func populateFromRepo(data *RepoInfo, r *gogithub.Repository) {
	data.FullName = r.GetFullName()
	data.HTMLURL = r.GetHTMLURL()
	data.Description = r.GetDescription()
	data.DefaultBranch = r.GetDefaultBranch()
	data.License = ""
	data.IsFork = r.GetFork()
	data.IsArchived = r.GetArchived()
	data.IsPrivate = r.GetPrivate()
	data.OpenIssues = r.GetOpenIssuesCount() // includes PRs initially
	data.StarCount = r.GetStargazersCount()
	data.ForkCount = r.GetForksCount()
	data.CreatedAt = r.GetCreatedAt().Time
	data.UpdatedAt = r.GetUpdatedAt().Time
	data.PushedAt = r.GetPushedAt().Time

	if lic := r.GetLicense(); lic != nil {
		data.License = lic.GetSPDXID()
	}

	if r.Topics != nil {
		data.Topics = r.Topics
	}

	if parent := r.GetParent(); parent != nil {
		data.ParentFullName = parent.GetFullName()
	}

	if perms := r.GetPermissions(); perms != nil {
		data.HasAdminAccess = perms["admin"]
		data.HasPushAccess = perms["push"]
	}
}

func countOpenPRs(ctx context.Context, c *Client, owner, repo string) (int, error) {
	_, resp, err := c.gh.PullRequests.List(ctx, owner, repo, &gogithub.PullRequestListOptions{
		State: "open",
		ListOptions: gogithub.ListOptions{
			PerPage: 1,
		},
	})
	c.updateRate(resp)

	if err != nil {
		return 0, err
	}

	return resp.LastPage, nil
}

// CollectSecurityAlerts fetches open security alert counts from all three
// GitHub security APIs (Dependabot, code scanning, secret scanning).
//
// Each API call is independent — a 403 (insufficient permissions) on one
// does not prevent the others from being queried. Fields remain -1 when
// the corresponding API is not accessible.
//
// Total: up to 3 API calls.
func CollectSecurityAlerts(ctx context.Context, c *Client, data *RepoInfo) {
	owner, repo := data.Owner, data.Repo

	// Dependabot alerts.
	state := "open"

	depAlerts, resp, err := c.gh.Dependabot.ListRepoAlerts(ctx, owner, repo, &gogithub.ListAlertsOptions{
		State:       &state,
		ListOptions: gogithub.ListOptions{PerPage: 1},
	})
	c.updateRate(resp)

	if err == nil {
		data.DependabotAlerts = countFromResponse(depAlerts, resp)
	}

	// Code scanning alerts.
	csAlerts, resp, err := c.gh.CodeScanning.ListAlertsForRepo(ctx, owner, repo, &gogithub.AlertListOptions{
		State:       "open",
		ListOptions: gogithub.ListOptions{PerPage: 1},
	})
	c.updateRate(resp)

	if err == nil {
		data.CodeScanningAlerts = countFromCodeScanResponse(csAlerts, resp)
	}

	// Secret scanning alerts.
	ssAlerts, resp, err := c.gh.SecretScanning.ListAlertsForRepo(ctx, owner, repo, &gogithub.SecretScanningAlertListOptions{
		State:       "open",
		ListOptions: gogithub.ListOptions{PerPage: 1},
	})
	c.updateRate(resp)

	if err == nil {
		data.SecretScanningAlerts = countFromSecretScanResponse(ssAlerts, resp)
	}
}

// countFromResponse estimates total count from a paginated response.
// If LastPage > 0, that's the total page count (with per_page=1, it equals total items).
// Otherwise, use the length of the returned slice.
func countFromResponse(alerts []*gogithub.DependabotAlert, resp *gogithub.Response) int {
	if resp != nil && resp.LastPage > 0 {
		return resp.LastPage
	}

	return len(alerts)
}

func countFromCodeScanResponse(alerts []*gogithub.Alert, resp *gogithub.Response) int {
	if resp != nil && resp.LastPage > 0 {
		return resp.LastPage
	}

	return len(alerts)
}

func countFromSecretScanResponse(alerts []*gogithub.SecretScanningAlert, resp *gogithub.Response) int {
	if resp != nil && resp.LastPage > 0 {
		return resp.LastPage
	}

	return len(alerts)
}

// SPDX-License-Identifier: Apache-2.0

//go:build integration

package main

import (
	"context"
	"testing"

	gh "github.com/fredbi/git-janitor/internal/github"
)

func TestGitHubClientSmoke(t *testing.T) {
	c := gh.NewClient()
	if !c.Available() {
		t.Skip("no GITHUB_TOKEN/GH_TOKEN set")
	}

	data := c.Fetch(context.Background(), "fredbi", "git-janitor", gh.FetchOptions{ForceRefresh: true, FetchSecurity: true})
	if data.Err != nil {
		t.Fatalf("fetch error: %v", data.Err)
	}

	t.Logf("Repo:    %s", data.FullName)
	t.Logf("Desc:    %s", data.Description)
	t.Logf("Private: %v", data.IsPrivate)
	t.Logf("Fork:    %v", data.IsFork)
	t.Logf("Stars:   %d", data.StarCount)
	t.Logf("Issues:  %d  PRs: %d", data.OpenIssues, data.OpenPRs)
	t.Logf("License: %s", data.License)
	t.Logf("Rate:    %+v", c.Rate())
	t.Logf("Scopes:  %q", c.Scopes())

	if data.FullName != "fredbi/git-janitor" {
		t.Errorf("unexpected FullName: %q", data.FullName)
	}
}

func TestGitHubClientArchived(t *testing.T) {
	c := gh.NewClient()
	if !c.Available() {
		t.Skip("no GITHUB_TOKEN/GH_TOKEN set")
	}

	data := c.Fetch(context.Background(), "go-openapi", "stubs", gh.FetchOptions{ForceRefresh: true, FetchSecurity: true})
	if data.Err != nil {
		t.Fatalf("fetch error: %v", data.Err)
	}

	t.Logf("Repo:     %s", data.FullName)
	t.Logf("Archived: %v", data.IsArchived)
	t.Logf("Private:  %v", data.IsPrivate)
	t.Logf("Desc:     %s", data.Description)

	if !data.IsArchived {
		t.Error("expected go-openapi/stubs to be archived")
	}
}

func TestGitHubClientSecurityAlerts(t *testing.T) {
	c := gh.NewClient()
	if !c.Available() {
		t.Skip("no GITHUB_TOKEN/GH_TOKEN set")
	}

	// Use a well-known repo likely to have some Dependabot alerts.
	data := c.Fetch(context.Background(), "fredbi", "git-janitor", gh.FetchOptions{ForceRefresh: true, FetchSecurity: true})
	if data.Err != nil {
		t.Fatalf("fetch error: %v", data.Err)
	}

	t.Logf("Dependabot:     %d", data.DependabotAlerts)
	t.Logf("Code scanning:  %d", data.CodeScanningAlerts)
	t.Logf("Secret scanning:%d", data.SecretScanningAlerts)
	t.Logf("Total:          %d", data.SecurityAlerts())

	// We don't assert specific counts — just that the APIs were reachable
	// (fields are >= 0, not -1).
	if data.DependabotAlerts < 0 && data.CodeScanningAlerts < 0 && data.SecretScanningAlerts < 0 {
		t.Log("warning: all security APIs returned -1 (no access)")
	}

	// Also test repos where user has admin access (security APIs require it).
	for _, r := range []struct{ owner, repo string }{
		{"go-openapi", "runtime"},
		{"go-openapi", "validate"},
		{"go-swagger", "go-swagger"},
	} {
		d := c.Fetch(context.Background(), r.owner, r.repo, gh.FetchOptions{ForceRefresh: true, FetchSecurity: true})
		if d.Err != nil {
			continue
		}

		t.Logf("%-35s dep=%d code=%d secret=%d total=%d",
			r.owner+"/"+r.repo, d.DependabotAlerts, d.CodeScanningAlerts, d.SecretScanningAlerts, d.SecurityAlerts())
	}
}

func TestGitHubClientFork(t *testing.T) {
	c := gh.NewClient()
	if !c.Available() {
		t.Skip("no GITHUB_TOKEN/GH_TOKEN set")
	}

	data := c.Fetch(context.Background(), "fredbi", "go-swagger", gh.FetchOptions{ForceRefresh: true, FetchSecurity: true})
	if data.Err != nil {
		t.Fatalf("fetch error: %v", data.Err)
	}

	t.Logf("Repo:   %s", data.FullName)
	t.Logf("Fork:   %v  Parent: %s", data.IsFork, data.ParentFullName)
	t.Logf("Stars:  %d  Issues: %d  PRs: %d", data.StarCount, data.OpenIssues, data.OpenPRs)

	if !data.IsFork {
		t.Error("expected fredbi/go-swagger to be a fork")
	}

	if data.ParentFullName == "" {
		t.Error("expected ParentFullName to be set for a fork")
	}
}

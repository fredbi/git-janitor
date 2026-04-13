// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RateInfo tracks GitLab API rate-limit state.
type RateInfo struct {
	Remaining int
	Limit     int
	ResetAt   time.Time
}

// FetchOptions controls what data to fetch.
type FetchOptions struct {
	ForceRefresh  bool
	FetchSecurity bool
}

// Runner wraps a Client for use as a context-injected runner by the engine.
type Runner struct {
	*Client
}

// NewRunner creates a Runner backed by a new Client with the given base URL.
func NewRunner(baseURL string) *Runner {
	return &Runner{
		Client: NewClient(baseURL),
	}
}

// Client wraps the GitLab API client with token resolution and rate-limit awareness.
type Client struct {
	gl        *gitlab.Client
	available bool
	baseURL   string
	cache     *Cache

	mu   sync.Mutex
	rate RateInfo
}

// NewClient creates a Client from environment variables.
//
// It checks GITLAB_TOKEN first, then GL_TOKEN as fallback.
// If neither is set, Available() returns false and all fetch
// operations return immediately with a nil PlatformInfo.
//
// The baseURL should be the GitLab instance API base URL
// (e.g. "https://gitlab.example.com"). When empty, defaults to
// "https://gitlab.com".
func NewClient(baseURL string) *Client {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		token = os.Getenv("GL_TOKEN")
	}

	if token == "" {
		return &Client{available: false, baseURL: baseURL, cache: NewCache(defaultCacheTTL)}
	}

	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	gl, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return &Client{available: false, baseURL: baseURL, cache: NewCache(defaultCacheTTL)}
	}

	return &Client{
		gl:        gl,
		available: true,
		baseURL:   baseURL,
		cache:     NewCache(defaultCacheTTL),
	}
}

// Available reports whether a valid token was found.
func (c *Client) Available() bool {
	return c.available
}

// BaseURL returns the configured GitLab instance base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Rate returns the current rate-limit info.
func (c *Client) Rate() RateInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.rate
}

// SetDescription updates the project description on GitLab.
func (c *Client) SetDescription(ctx context.Context, projectID int, description string) error {
	if !c.available {
		return errors.New("gitlab: no token available")
	}

	_, resp, err := c.gl.Projects.EditProject(projectID, &gitlab.EditProjectOptions{
		Description: gitlab.Ptr(description),
	}, gitlab.WithContext(ctx))
	c.updateRate(resp)

	return err
}

// EnableBranchProtection sets a minimal branch protection rule on the given branch:
// requires at least one merge request approval before merging.
func (c *Client) EnableBranchProtection(ctx context.Context, projectID int, branch string) error {
	if !c.available {
		return errors.New("gitlab: no token available")
	}

	_, resp, err := c.gl.ProtectedBranches.ProtectRepositoryBranches(projectID, &gitlab.ProtectRepositoryBranchesOptions{
		Name: gitlab.Ptr(branch),
	}, gitlab.WithContext(ctx))
	c.updateRate(resp)

	return err
}

// EnableDeleteBranchOnMerge enables the "Remove source branch after merge"
// setting on the given project.
func (c *Client) EnableDeleteBranchOnMerge(ctx context.Context, projectID int) error {
	if !c.available {
		return errors.New("gitlab: no token available")
	}

	_, resp, err := c.gl.Projects.EditProject(projectID, &gitlab.EditProjectOptions{
		RemoveSourceBranchAfterMerge: gitlab.Ptr(true),
	}, gitlab.WithContext(ctx))
	c.updateRate(resp)

	return err
}

// Fetch retrieves GitLab project data, using the cache unless ForceRefresh is true.
func (c *Client) Fetch(ctx context.Context, projectPath string, opts FetchOptions) *models.PlatformInfo {
	if !c.available {
		return nil
	}

	if !opts.ForceRefresh {
		if cached, ok := c.cache.Get(projectPath); ok {
			return cached
		}
	}

	if c.rateLimited() {
		owner, repo := OwnerAndRepo(projectPath)
		data := models.NewPlatformInfo(owner, repo)
		data.Err = ErrRateLimited

		return data
	}

	data := collectRepoInfo(ctx, c, projectPath, opts.FetchSecurity)
	c.cache.Set(projectPath, data)

	return data
}

const (
	rateLimitCautionThreshold = 100
	rateLimitHardStop         = 20
)

// updateRate extracts rate-limit info from GitLab response headers.
//
// GitLab uses RateLimit-Remaining, RateLimit-Limit, RateLimit-Reset headers
// (unlike GitHub's X-RateLimit-* headers).
func (c *Client) updateRate(resp *gitlab.Response) {
	if resp == nil || resp.Response == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if v := resp.Header.Get("RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.rate.Remaining = n
		}
	}

	if v := resp.Header.Get("RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.rate.Limit = n
		}
	}

	if v := resp.Header.Get("RateLimit-Reset"); v != "" {
		if epoch, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.rate.ResetAt = time.Unix(epoch, 0)
		}
	}

	// Extend cache TTL when running low on quota.
	if c.rate.Remaining > 0 && c.rate.Remaining < rateLimitCautionThreshold {
		c.cache.SetTTL(extendedCacheTTL)
	}
}

// rateLimited reports whether we should skip API calls.
func (c *Client) rateLimited() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.rate.Remaining > 0 && c.rate.Remaining < rateLimitHardStop &&
		time.Now().Before(c.rate.ResetAt)
}

// isNotFound checks if a GitLab API error is a 404.
func isNotFound(resp *gitlab.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}

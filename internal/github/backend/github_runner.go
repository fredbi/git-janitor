// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	gogithub "github.com/google/go-github/v72/github"
)

// RateInfo tracks GitHub API rate-limit state.
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

// NewRunner creates a Runner backed by a new Client.
func NewRunner() *Runner {
	return &Runner{
		Client: NewClient(),
	}
}

// Client wraps the go-github client with token resolution and rate-limit awareness.
type Client struct {
	gh        *gogithub.Client
	available bool
	cache     *Cache

	mu     sync.Mutex
	rate   RateInfo
	scopes string // token scopes from X-OAuth-Scopes header
}

// NewClient creates a Client from environment variables.
//
// It checks GITHUB_TOKEN first, then GH_TOKEN as fallback.
// If neither is set, Available() returns false and all fetch
// operations return immediately with a nil PlatformInfo.
func NewClient() *Client {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}

	if token == "" {
		return &Client{available: false, cache: NewCache(defaultCacheTTL)}
	}

	gh := gogithub.NewClient(nil).WithAuthToken(token)

	return &Client{
		gh:        gh,
		available: true,
		cache:     NewCache(defaultCacheTTL),
	}
}

// Available reports whether a valid token was found.
func (c *Client) Available() bool {
	return c.available
}

// Rate returns the current rate-limit info.
func (c *Client) Rate() RateInfo {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.rate
}

// Scopes returns the token's OAuth scopes as reported by GitHub.
// Empty string if not yet known (no API call made) or if using a fine-grained token.
func (c *Client) Scopes() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.scopes
}

// SetDescription updates the repository description on GitHub.
func (c *Client) SetDescription(ctx context.Context, owner, repo, description string) error {
	if !c.available {
		return errors.New("github: no token available")
	}

	_, resp, err := c.gh.Repositories.Edit(ctx, owner, repo, &gogithub.Repository{
		Description: &description,
	})
	c.updateRate(resp)

	return err
}

// Fetch retrieves GitHub repo data, using the cache unless ForceRefresh is true.
func (c *Client) Fetch(ctx context.Context, owner, repo string, opts FetchOptions) *models.PlatformInfo {
	if !c.available {
		return nil
	}

	key := owner + "/" + repo

	if !opts.ForceRefresh {
		if cached, ok := c.cache.Get(key); ok {
			return cached
		}
	}

	if c.rateLimited() {
		data := models.NewPlatformInfo(owner, repo)
		data.Err = ErrRateLimited

		return data
	}

	data := collectRepoInfo(ctx, c, owner, repo, opts.FetchSecurity)
	c.cache.Set(key, data)

	return data
}

const (
	rateLimitCautionThreshold = 100
	rateLimitHardStop         = 20
)

func (c *Client) updateRate(resp *gogithub.Response) {
	if resp == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.rate = RateInfo{
		Remaining: resp.Rate.Remaining,
		Limit:     resp.Rate.Limit,
		ResetAt:   resp.Rate.Reset.Time,
	}

	// Capture token scopes from the first response that has them.
	if c.scopes == "" && resp.Response != nil {
		c.scopes = resp.Header.Get("X-Oauth-Scopes")
	}

	// Extend cache TTL when running low on quota.
	if resp.Rate.Remaining < rateLimitCautionThreshold {
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

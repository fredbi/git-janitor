// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"sync"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

const (
	defaultCacheTTL  = 5 * time.Minute
	extendedCacheTTL = 10 * time.Minute
)

type cacheEntry struct {
	data      *models.PlatformInfo
	expiresAt time.Time
}

// Cache is a simple TTL cache for GitLab PlatformInfo, keyed by project path.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// NewCache creates a cache with the given default TTL.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get returns cached data if present and not expired.
func (c *Cache) Get(key string) (*models.PlatformInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.data, true
}

// Set stores data in the cache with the current TTL.
func (c *Cache) Set(key string, data *models.PlatformInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// SetTTL changes the cache TTL for future entries.
// Existing entries keep their original expiry.
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttl = ttl
}

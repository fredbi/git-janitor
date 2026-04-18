// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"log"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/store"
)

// cacheGet retrieves a cached RepoInfo for the given path, if valid.
//
// Returns nil if the store is not configured, no entry exists, the entry
// has expired, or the entry's collect level is insufficient.
func (e *Interactive) cacheGet(path string, requireFull bool) *models.RepoInfo {
	if e.store == nil {
		return nil
	}

	data, err := e.store.Get(store.BucketCache, path)
	if err != nil {
		log.Printf("cache: get %q: %v", path, err)

		return nil
	}

	if data == nil {
		return nil
	}

	info, err := unmarshalRepoInfo(data)
	if err != nil {
		log.Printf("cache: unmarshal %q: %v", path, err)

		return nil
	}

	// TTL check.
	if time.Since(info.CollectedAt) > e.cacheTTL {
		return nil
	}

	// Level check: a full-collect request cannot be satisfied by a fast-level cache entry.
	if requireFull && info.CollectLevel != models.CollectLevelFull {
		return nil
	}

	return info
}

// cachePut stores a RepoInfo in the cache. Best-effort: errors are logged.
func (e *Interactive) cachePut(info *models.RepoInfo) {
	if e.store == nil {
		return
	}

	data, err := marshalRepoInfo(info)
	if err != nil {
		log.Printf("cache: marshal %q: %v", info.Path, err)

		return
	}

	if err := e.store.Put(store.BucketCache, info.Path, data); err != nil {
		log.Printf("cache: put %q: %v", info.Path, err)
	}
}

// cacheDelete removes a cache entry for the given path.
func (e *Interactive) cacheDelete(path string) {
	if e.store == nil {
		return
	}

	if err := e.store.Delete(store.BucketCache, path); err != nil {
		log.Printf("cache: delete %q: %v", path, err)
	}
}

// ClearCache removes every entry from the persistent RepoInfo cache.
// Returns the number of entries that were removed.
func (e *Interactive) ClearCache() (int, error) {
	if e.store == nil {
		return 0, nil
	}

	var count int
	if err := e.store.Scan(store.BucketCache, "", func(_, _ []byte) bool {
		count++

		return true
	}); err != nil {
		return 0, err
	}

	if err := e.store.ClearBucket(store.BucketCache); err != nil {
		return 0, err
	}

	return count, nil
}

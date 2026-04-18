// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/store/bolt"
)

func newTestEngine(t *testing.T, ttl time.Duration) *Interactive {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	s, err := bolt.New(dbPath)
	if err != nil {
		t.Fatalf("bolt.New: %v", err)
	}

	t.Cleanup(func() {
		_ = s.Close()
	})

	return NewInteractive(
		WithStore(s),
		WithCacheTTL(ttl),
	)
}

func TestCache_PutAndGet(t *testing.T) {
	eng := newTestEngine(t, time.Minute)

	info := &models.RepoInfo{
		Path:          "/repo/test",
		IsGit:         true,
		CollectedAt:   time.Now(),
		CollectLevel:  models.CollectLevelFull,
		DefaultBranch: "main",
	}

	eng.cachePut(info)

	got := eng.cacheGet("/repo/test", false)
	if got == nil {
		t.Fatal("cacheGet returned nil, expected cached entry")
	}

	if got.Path != info.Path {
		t.Errorf("Path = %q, want %q", got.Path, info.Path)
	}

	if got.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q, want %q", got.DefaultBranch, "main")
	}
}

func TestCache_TTLExpiry(t *testing.T) {
	eng := newTestEngine(t, 50*time.Millisecond)

	info := &models.RepoInfo{
		Path:         "/repo/test",
		IsGit:        true,
		CollectedAt:  time.Now(),
		CollectLevel: models.CollectLevelFast,
	}

	eng.cachePut(info)

	// Should be available immediately.
	if got := eng.cacheGet("/repo/test", false); got == nil {
		t.Fatal("cacheGet returned nil before TTL expiry")
	}

	// Wait for TTL to expire.
	time.Sleep(100 * time.Millisecond)

	if got := eng.cacheGet("/repo/test", false); got != nil {
		t.Error("cacheGet returned non-nil after TTL expiry")
	}
}

func TestCache_FastVsFullLevel(t *testing.T) {
	eng := newTestEngine(t, time.Minute)

	// Cache a fast-level entry.
	info := &models.RepoInfo{
		Path:         "/repo/test",
		IsGit:        true,
		CollectedAt:  time.Now(),
		CollectLevel: models.CollectLevelFast,
	}

	eng.cachePut(info)

	// Fast request should be satisfied.
	if got := eng.cacheGet("/repo/test", false); got == nil {
		t.Error("fast request should be satisfied by fast-level cache")
	}

	// Full request should NOT be satisfied by fast-level cache.
	if got := eng.cacheGet("/repo/test", true); got != nil {
		t.Error("full request should NOT be satisfied by fast-level cache")
	}

	// Now cache a full-level entry.
	info.CollectLevel = models.CollectLevelFull
	info.CollectedAt = time.Now()

	eng.cachePut(info)

	// Both fast and full requests should be satisfied.
	if got := eng.cacheGet("/repo/test", false); got == nil {
		t.Error("fast request should be satisfied by full-level cache")
	}

	if got := eng.cacheGet("/repo/test", true); got == nil {
		t.Error("full request should be satisfied by full-level cache")
	}
}

func TestCache_NilStore(t *testing.T) {
	eng := NewInteractive() // no store configured

	info := &models.RepoInfo{
		Path:         "/repo/test",
		IsGit:        true,
		CollectedAt:  time.Now(),
		CollectLevel: models.CollectLevelFull,
	}

	// These should be no-ops, not panics.
	eng.cachePut(info)

	got := eng.cacheGet("/repo/test", false)
	if got != nil {
		t.Error("cacheGet with nil store should return nil")
	}

	eng.cacheDelete("/repo/test") // should not panic
}

func TestClearCache(t *testing.T) {
	eng := newTestEngine(t, time.Minute)

	// Populate three entries.
	for _, p := range []string{"/repo/a", "/repo/b", "/repo/c"} {
		eng.cachePut(&models.RepoInfo{
			Path:         p,
			IsGit:        true,
			CollectedAt:  time.Now(),
			CollectLevel: models.CollectLevelFull,
		})
	}

	n, err := eng.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache: %v", err)
	}

	if n != 3 {
		t.Errorf("ClearCache returned %d, want 3", n)
	}

	for _, p := range []string{"/repo/a", "/repo/b", "/repo/c"} {
		if got := eng.cacheGet(p, false); got != nil {
			t.Errorf("cacheGet(%q) returned non-nil after ClearCache", p)
		}
	}

	// Second call on empty bucket should report zero and not fail.
	n, err = eng.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache (empty): %v", err)
	}

	if n != 0 {
		t.Errorf("ClearCache on empty bucket returned %d, want 0", n)
	}
}

func TestClearCache_NilStore(t *testing.T) {
	eng := NewInteractive()

	n, err := eng.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache with nil store: %v", err)
	}

	if n != 0 {
		t.Errorf("ClearCache with nil store returned %d, want 0", n)
	}
}

func TestCache_Delete(t *testing.T) {
	eng := newTestEngine(t, time.Minute)

	info := &models.RepoInfo{
		Path:         "/repo/test",
		IsGit:        true,
		CollectedAt:  time.Now(),
		CollectLevel: models.CollectLevelFull,
	}

	eng.cachePut(info)

	// Verify it's there.
	if got := eng.cacheGet("/repo/test", false); got == nil {
		t.Fatal("cacheGet returned nil before delete")
	}

	eng.cacheDelete("/repo/test")

	// Should be gone.
	if got := eng.cacheGet("/repo/test", false); got != nil {
		t.Error("cacheGet returned non-nil after delete")
	}
}

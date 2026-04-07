// SPDX-License-Identifier: Apache-2.0

package bolt_test

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/fredbi/git-janitor/internal/store"
	"github.com/fredbi/git-janitor/internal/store/bolt"
)

func newTestStore(t *testing.T) *bolt.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	s, err := bolt.New(dbPath)
	if err != nil {
		t.Fatalf("New(%q): %v", dbPath, err)
	}

	t.Cleanup(func() {
		_ = s.Close()
	})

	return s
}

func TestBoltStore_PutGetDelete(t *testing.T) {
	s := newTestStore(t)

	// Put a value.
	if err := s.Put(store.BucketCache, "/repo/a", []byte("hello")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Get it back.
	got, err := s.Get(store.BucketCache, "/repo/a")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(got) != "hello" {
		t.Errorf("Get = %q, want %q", got, "hello")
	}

	// Delete it.
	if err := s.Delete(store.BucketCache, "/repo/a"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should be gone.
	got, err = s.Get(store.BucketCache, "/repo/a")
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}

	if got != nil {
		t.Errorf("Get after delete = %q, want nil", got)
	}
}

func TestBoltStore_GetMissing(t *testing.T) {
	s := newTestStore(t)

	got, err := s.Get(store.BucketCache, "/nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got != nil {
		t.Errorf("Get missing = %q, want nil", got)
	}
}

func TestBoltStore_DeleteMissing(t *testing.T) {
	s := newTestStore(t)

	// Deleting a non-existent key should not error.
	if err := s.Delete(store.BucketCache, "/nonexistent"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestBoltStore_MultipleBuckets(t *testing.T) {
	s := newTestStore(t)

	// Same key in different buckets should be independent.
	if err := s.Put(store.BucketCache, "key", []byte("cache-val")); err != nil {
		t.Fatalf("Put cache: %v", err)
	}

	if err := s.Put(store.BucketHistory, "key", []byte("history-val")); err != nil {
		t.Fatalf("Put history: %v", err)
	}

	gotCache, _ := s.Get(store.BucketCache, "key")
	gotHistory, _ := s.Get(store.BucketHistory, "key")

	if string(gotCache) != "cache-val" {
		t.Errorf("cache bucket = %q, want %q", gotCache, "cache-val")
	}

	if string(gotHistory) != "history-val" {
		t.Errorf("history bucket = %q, want %q", gotHistory, "history-val")
	}
}

func TestBoltStore_ConcurrentReads(t *testing.T) {
	s := newTestStore(t)

	if err := s.Put(store.BucketCache, "key", []byte("value")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	const numReaders = 10

	var wg sync.WaitGroup

	wg.Add(numReaders)

	for range numReaders {
		go func() {
			defer wg.Done()

			got, err := s.Get(store.BucketCache, "key")
			if err != nil {
				t.Errorf("concurrent Get: %v", err)

				return
			}

			if string(got) != "value" {
				t.Errorf("concurrent Get = %q, want %q", got, "value")
			}
		}()
	}

	wg.Wait()
}

func TestBoltStore_CloseAndReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reopen.db")

	// Open, write, close.
	s1, err := bolt.New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s1.Put(store.BucketCache, "persist", []byte("data")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := s1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen and verify.
	s2, err := bolt.New(dbPath)
	if err != nil {
		t.Fatalf("New (reopen): %v", err)
	}

	defer func() {
		_ = s2.Close()
	}()

	got, err := s2.Get(store.BucketCache, "persist")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if string(got) != "data" {
		t.Errorf("Get after reopen = %q, want %q", got, "data")
	}
}

func TestBoltStore_Scan(t *testing.T) {
	s := newTestStore(t)

	// Insert keys with a common prefix and one without.
	for _, kv := range []struct{ k, v string }{
		{"/repo/a#ts1", "entry1"},
		{"/repo/a#ts2", "entry2"},
		{"/repo/a#ts3", "entry3"},
		{"/repo/b#ts1", "other"},
	} {
		if err := s.Put(store.BucketHistory, kv.k, []byte(kv.v)); err != nil {
			t.Fatalf("Put %q: %v", kv.k, err)
		}
	}

	// Scan with prefix "/repo/a#" should return 3 entries in order.
	var keys []string

	err := s.Scan(store.BucketHistory, "/repo/a#", func(k, _ []byte) bool {
		keys = append(keys, string(k))

		return true
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(keys) != 3 {
		t.Fatalf("Scan returned %d keys, want 3", len(keys))
	}

	// Should be in sorted order.
	if keys[0] != "/repo/a#ts1" || keys[2] != "/repo/a#ts3" {
		t.Errorf("keys not in expected order: %v", keys)
	}
}

func TestBoltStore_ScanEarlyStop(t *testing.T) {
	s := newTestStore(t)

	for i := range 5 {
		k := "prefix#" + string(rune('0'+i))
		if err := s.Put(store.BucketHistory, k, []byte("v")); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	var count int

	_ = s.Scan(store.BucketHistory, "prefix#", func(_, _ []byte) bool {
		count++

		return count < 2 // stop after 2
	})

	if count != 2 {
		t.Errorf("Scan with early stop: visited %d, want 2", count)
	}
}

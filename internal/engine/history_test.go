// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/store/bolt"
)

func newTestEngineForHistory(t *testing.T) *Interactive {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	s, err := bolt.New(dbPath)
	if err != nil {
		t.Fatalf("bolt.New: %v", err)
	}

	t.Cleanup(func() {
		_ = s.Close()
	})

	return NewInteractive(WithStore(s))
}

func TestHistory_AppendAndRetrieve(t *testing.T) {
	eng := newTestEngineForHistory(t)
	now := time.Now()

	eng.appendHistory(models.HistoryEntry{
		Timestamp:  now,
		RepoPath:   "/repo/a",
		ActionName: "delete-branch",
		Subjects:   []string{"feature/old"},
		Result:     models.Result{OK: true, Message: "deleted"},
	})

	entries := eng.RecentHistory("/repo/a", now.Add(-time.Hour))
	if len(entries) != 1 {
		t.Fatalf("RecentHistory returned %d entries, want 1", len(entries))
	}

	if entries[0].ActionName != "delete-branch" {
		t.Errorf("ActionName = %q, want %q", entries[0].ActionName, "delete-branch")
	}

	if !entries[0].Result.OK {
		t.Error("Result.OK should be true")
	}

	if len(entries[0].Subjects) != 1 || entries[0].Subjects[0] != "feature/old" {
		t.Errorf("Subjects = %v, want [feature/old]", entries[0].Subjects)
	}
}

func TestHistory_TimeFilter(t *testing.T) {
	eng := newTestEngineForHistory(t)
	now := time.Now()

	// Old entry (2 hours ago).
	eng.appendHistory(models.HistoryEntry{
		Timestamp:  now.Add(-2 * time.Hour),
		RepoPath:   "/repo/a",
		ActionName: "old-action",
		Result:     models.Result{OK: true, Message: "ok"},
	})

	// Recent entry (5 minutes ago).
	eng.appendHistory(models.HistoryEntry{
		Timestamp:  now.Add(-5 * time.Minute),
		RepoPath:   "/repo/a",
		ActionName: "recent-action",
		Result:     models.Result{OK: true, Message: "ok"},
	})

	// Query with 1-hour window: should only return the recent entry.
	entries := eng.RecentHistory("/repo/a", now.Add(-time.Hour))
	if len(entries) != 1 {
		t.Fatalf("RecentHistory returned %d entries, want 1", len(entries))
	}

	if entries[0].ActionName != "recent-action" {
		t.Errorf("ActionName = %q, want %q", entries[0].ActionName, "recent-action")
	}

	// Query with 3-hour window: should return both entries.
	entries = eng.RecentHistory("/repo/a", now.Add(-3*time.Hour))
	if len(entries) != 2 {
		t.Fatalf("RecentHistory returned %d entries, want 2", len(entries))
	}

	// Newest first.
	if entries[0].ActionName != "recent-action" {
		t.Errorf("entries[0].ActionName = %q, want %q", entries[0].ActionName, "recent-action")
	}

	if entries[1].ActionName != "old-action" {
		t.Errorf("entries[1].ActionName = %q, want %q", entries[1].ActionName, "old-action")
	}
}

func TestHistory_PrefixIsolation(t *testing.T) {
	eng := newTestEngineForHistory(t)
	now := time.Now()

	eng.appendHistory(models.HistoryEntry{
		Timestamp:  now,
		RepoPath:   "/repo/a",
		ActionName: "action-a",
		Result:     models.Result{OK: true, Message: "ok"},
	})

	eng.appendHistory(models.HistoryEntry{
		Timestamp:  now,
		RepoPath:   "/repo/b",
		ActionName: "action-b",
		Result:     models.Result{OK: true, Message: "ok"},
	})

	// Repo A should only see its own entries.
	entriesA := eng.RecentHistory("/repo/a", now.Add(-time.Hour))
	if len(entriesA) != 1 {
		t.Fatalf("repo/a: got %d entries, want 1", len(entriesA))
	}

	if entriesA[0].ActionName != "action-a" {
		t.Errorf("repo/a: ActionName = %q, want %q", entriesA[0].ActionName, "action-a")
	}

	// Repo B should only see its own entries.
	entriesB := eng.RecentHistory("/repo/b", now.Add(-time.Hour))
	if len(entriesB) != 1 {
		t.Fatalf("repo/b: got %d entries, want 1", len(entriesB))
	}

	if entriesB[0].ActionName != "action-b" {
		t.Errorf("repo/b: ActionName = %q, want %q", entriesB[0].ActionName, "action-b")
	}
}

func TestHistory_NilStore(t *testing.T) {
	eng := NewInteractive() // no store

	// Should not panic.
	eng.appendHistory(models.HistoryEntry{
		Timestamp:  time.Now(),
		RepoPath:   "/repo/a",
		ActionName: "test",
		Result:     models.Result{OK: true, Message: "ok"},
	})

	entries := eng.RecentHistory("/repo/a", time.Now().Add(-time.Hour))
	if entries != nil {
		t.Errorf("RecentHistory with nil store should return nil, got %d entries", len(entries))
	}
}

func TestPurgeHistory_All(t *testing.T) {
	eng := newTestEngineForHistory(t)
	now := time.Now()

	for _, p := range []string{"/repo/a", "/repo/b"} {
		eng.appendHistory(models.HistoryEntry{
			Timestamp:  now,
			RepoPath:   p,
			ActionName: "act",
			Result:     models.Result{OK: true, Message: "ok"},
		})
	}

	n, err := eng.PurgeHistory(0)
	if err != nil {
		t.Fatalf("PurgeHistory(0): %v", err)
	}

	if n != 2 {
		t.Errorf("PurgeHistory(0) = %d, want 2", n)
	}

	for _, p := range []string{"/repo/a", "/repo/b"} {
		if entries := eng.RecentHistory(p, now.Add(-time.Hour)); len(entries) != 0 {
			t.Errorf("RecentHistory(%q) returned %d entries after purge", p, len(entries))
		}
	}
}

func TestPurgeHistory_OlderThanDays(t *testing.T) {
	eng := newTestEngineForHistory(t)
	now := time.Now()

	entries := []models.HistoryEntry{
		{Timestamp: now.AddDate(0, 0, -60), RepoPath: "/repo/a", ActionName: "very-old", Result: models.Result{OK: true, Message: "ok"}},
		{Timestamp: now.AddDate(0, 0, -45), RepoPath: "/repo/a", ActionName: "old", Result: models.Result{OK: true, Message: "ok"}},
		{Timestamp: now.AddDate(0, 0, -10), RepoPath: "/repo/a", ActionName: "recent", Result: models.Result{OK: true, Message: "ok"}},
		{Timestamp: now.AddDate(0, 0, -1), RepoPath: "/repo/a", ActionName: "brand-new", Result: models.Result{OK: true, Message: "ok"}},
	}
	for _, e := range entries {
		eng.appendHistory(e)
	}

	n, err := eng.PurgeHistory(30)
	if err != nil {
		t.Fatalf("PurgeHistory(30): %v", err)
	}

	if n != 2 {
		t.Errorf("PurgeHistory(30) = %d, want 2 (two entries older than 30 days)", n)
	}

	kept := eng.RecentHistory("/repo/a", now.AddDate(-1, 0, 0))
	if len(kept) != 2 {
		t.Fatalf("expected 2 surviving entries, got %d", len(kept))
	}

	surviving := map[string]bool{}
	for _, e := range kept {
		surviving[e.ActionName] = true
	}

	for _, name := range []string{"recent", "brand-new"} {
		if !surviving[name] {
			t.Errorf("expected %q to survive, got %v", name, surviving)
		}
	}
}

func TestPurgeHistory_NilStore(t *testing.T) {
	eng := NewInteractive()

	n, err := eng.PurgeHistory(0)
	if err != nil {
		t.Fatalf("PurgeHistory with nil store: %v", err)
	}

	if n != 0 {
		t.Errorf("PurgeHistory with nil store returned %d, want 0", n)
	}
}

func TestHistory_NewestFirst(t *testing.T) {
	eng := newTestEngineForHistory(t)
	now := time.Now()

	for i := range 5 {
		eng.appendHistory(models.HistoryEntry{
			Timestamp:  now.Add(time.Duration(i) * time.Minute),
			RepoPath:   "/repo/a",
			ActionName: "action-" + string(rune('0'+i)),
			Result:     models.Result{OK: true, Message: "ok"},
		})
	}

	entries := eng.RecentHistory("/repo/a", now.Add(-time.Hour))
	if len(entries) != 5 {
		t.Fatalf("got %d entries, want 5", len(entries))
	}

	// First entry should be the newest (action-4).
	if entries[0].Timestamp.Before(entries[len(entries)-1].Timestamp) {
		t.Error("entries are not newest-first")
	}
}

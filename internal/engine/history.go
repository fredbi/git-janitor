// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"log"
	"slices"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/store"
)

const historyKeySep = "#"

// historyKey builds a bbolt key for a history entry: "{repoPath}#{RFC3339Nano}".
// Keys with the same repoPath prefix are lexicographically sorted by time.
func historyKey(repoPath string, ts time.Time) string {
	return repoPath + historyKeySep + ts.Format(time.RFC3339Nano)
}

// appendHistory persists a history entry to the bbolt store.
// Best-effort: errors are logged, not returned.
func (e *Interactive) appendHistory(entry models.HistoryEntry) {
	if e.store == nil {
		return
	}

	data, err := marshalHistoryEntry(entry)
	if err != nil {
		log.Printf("history: marshal: %v", err)

		return
	}

	key := historyKey(entry.RepoPath, entry.Timestamp)
	if err := e.store.Put(store.BucketHistory, key, data); err != nil {
		log.Printf("history: put: %v", err)
	}
}

// PurgeHistory removes persisted action-history entries. When
// olderThanDays is greater than zero, only entries strictly older than
// that window are removed; otherwise all history is wiped. Returns the
// number of entries that were removed.
func (e *Interactive) PurgeHistory(olderThanDays int) (int, error) {
	if e.store == nil {
		return 0, nil
	}

	if olderThanDays <= 0 {
		var count int
		if err := e.store.Scan(store.BucketHistory, "", func(_, _ []byte) bool {
			count++

			return true
		}); err != nil {
			return 0, err
		}

		if err := e.store.ClearBucket(store.BucketHistory); err != nil {
			return 0, err
		}

		return count, nil
	}

	cutoff := time.Now().AddDate(0, 0, -olderThanDays)

	var victims []string

	if err := e.store.Scan(store.BucketHistory, "", func(key, value []byte) bool {
		entry, decErr := unmarshalHistoryEntry(value)
		if decErr != nil {
			// Treat undecodable entries as old so they get swept up.
			victims = append(victims, string(key))

			return true
		}

		if entry.Timestamp.Before(cutoff) {
			victims = append(victims, string(key))
		}

		return true
	}); err != nil {
		return 0, err
	}

	for _, k := range victims {
		if err := e.store.Delete(store.BucketHistory, k); err != nil {
			return 0, err
		}
	}

	return len(victims), nil
}

// RecentHistory returns history entries for the given repo with timestamps
// after since, ordered newest first.
func (e *Interactive) RecentHistory(repoPath string, since time.Time) []models.HistoryEntry {
	if e.store == nil {
		return nil
	}

	prefix := repoPath + historyKeySep
	var entries []models.HistoryEntry

	err := e.store.Scan(store.BucketHistory, prefix, func(_, value []byte) bool {
		entry, decErr := unmarshalHistoryEntry(value)
		if decErr != nil {
			log.Printf("history: unmarshal: %v", decErr)

			return true // skip bad entries, continue
		}

		if entry.Timestamp.Before(since) {
			return true // skip entries before cutoff, continue
		}

		entries = append(entries, entry)

		return true
	})
	if err != nil {
		log.Printf("history: scan %q: %v", repoPath, err)

		return nil
	}

	// Reverse to newest-first (bbolt scan is ascending by key).
	slices.Reverse(entries)

	return entries
}

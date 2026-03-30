package engine

import (
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// HistoryEntry records a single executed action and its result.
type HistoryEntry struct {
	Timestamp  time.Time
	RepoPath   string
	ActionName string
	Subjects   []string
	Result     models.Result
}

// History is an in-memory ring buffer of HistoryEntry records.
// Phase 2 will persist this to a KV store.
type History struct {
	entries []HistoryEntry
	max     int
}

// NewHistory creates a History with the given capacity.
func NewHistory(capacity int) *History {
	return &History{
		entries: make([]HistoryEntry, 0, capacity),
		max:     capacity,
	}
}

// Append adds an entry to the history, evicting the oldest if at capacity.
func (h *History) Append(entry HistoryEntry) {
	if len(h.entries) >= h.max {
		// Drop the oldest entry.
		copy(h.entries, h.entries[1:])
		h.entries = h.entries[:len(h.entries)-1]
	}

	h.entries = append(h.entries, entry)
}

// Entries returns all history entries, newest first.
func (h *History) Entries() []HistoryEntry {
	result := make([]HistoryEntry, len(h.entries))
	for i, e := range h.entries {
		result[len(h.entries)-1-i] = e
	}

	return result
}

// Len returns the number of entries in the history.
func (h *History) Len() int {
	return len(h.entries)
}

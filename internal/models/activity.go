// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// Staleness classification constants.
const (
	StalenessActive  = "active"  // commits in the last 30 days
	StalenessRecent  = "recent"  // commits in the last 90 days
	StalenessStale   = "stale"   // commits in the last 360 days
	StalenessDormant = "dormant" // no commits in the last 360 days
)

// Activity holds commit activity metrics for a repository.
//
// All windows are rolling: 7d, 30d, 90d, 360d from now.
// Counts are on HEAD only (merged activity).
type Activity struct {
	// Commit counts over rolling windows.
	Commits7d   int
	Commits30d  int
	Commits90d  int
	Commits360d int

	// TagsLast360d is the number of tags on the default branch created in the last 360 days.
	TagsLast360d int

	// Staleness is derived from commit counts:
	//   "active"  - Commits30d > 0
	//   "recent"  - Commits90d > 0
	//   "stale"   - Commits360d > 0
	//   "dormant" - otherwise
	Staleness string

	// Authors is populated on-demand via LoadAuthors.
	Authors []AuthorActivity
}

// AuthorActivity holds per-author commit counts.
type AuthorActivity struct {
	Name    string
	Email   string
	Commits int
}

// CountTagsInWindow counts how many tags from the given list fall within the last nDays.
func CountTagsInWindow(tags []Tag, nDays int) int {
	cutoff := time.Now().AddDate(0, 0, -nDays)

	var count int

	for i := range tags {
		if tags[i].RemoteOnly {
			continue
		}

		if tags[i].Date.After(cutoff) {
			count++
		}
	}

	return count
}

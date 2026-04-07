// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// Stash represents a single stash entry.
type Stash struct {
	// Ref is the stash reference (e.g. "stash@{0}").
	Ref string

	// Branch is the branch the stash was created on.
	Branch string

	// Message is the stash description.
	Message string

	// LastUpdatedAt is the timestamp of the stash entry.
	LastUpdatedAt time.Time

	// Detail is populated on demand by CollectDetails (nil until requested).
	Detail *StashDetail
}

// StashDetail holds on-demand detail information for a stash entry.
type StashDetail struct {
	// DiffStat is the output of git stash show --include-untracked <ref>.
	DiffStat string
}

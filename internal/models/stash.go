// SPDX-License-Identifier: Apache-2.0

package models

// Stash represents a single stash entry.
type Stash struct {
	// Ref is the stash reference (e.g. "stash@{0}").
	Ref string

	// Branch is the branch the stash was created on.
	Branch string

	// Message is the stash description.
	Message string
}

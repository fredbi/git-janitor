// SPDX-License-Identifier: Apache-2.0

package models

import "fmt"

// ActionResult holds the outcome of a git backend action.
type ActionResult struct {
	// OK is true if the action completed successfully.
	OK bool

	// Message describes what happened (success or failure).
	Message string
}

// DefaultPushRemote returns the remote to push to for the given repo.
// For forks (upstream exists): push to upstream.
// For clones (origin only): push to origin.
func DefaultPushRemote(remotes []Remote) string {
	if FindRemote(remotes, RemoteUpstream) != nil {
		return RemoteUpstream
	}

	return RemoteOrigin
}

// FormatBytes returns a human-readable byte size.
func FormatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

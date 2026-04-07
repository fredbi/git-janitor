// SPDX-License-Identifier: Apache-2.0

package models

// StatusEntry represents a single entry from git status --porcelain=v2.
type StatusEntry struct {
	// XY is the two-character status code (e.g. "M.", ".M", "A.", "??").
	XY string

	// Path is the file path relative to the repo root.
	Path string

	// OrigPath is set for renames/copies (the source path).
	OrigPath string
}

// IsUntracked reports whether the entry is an untracked file.
func (e StatusEntry) IsUntracked() bool {
	return e.XY == "??"
}

// IsIgnored reports whether the entry is an ignored file.
func (e StatusEntry) IsIgnored() bool {
	return e.XY == "!!"
}

// Status holds the parsed output of git status.
type Status struct {
	// Branch is the current branch name (empty if detached HEAD).
	Branch string

	// OID is the commit hash of HEAD.
	OID string

	// Upstream is the upstream tracking branch (e.g. "origin/main").
	Upstream string

	// AheadBehind holds the ahead/behind counts relative to upstream.
	Ahead  int
	Behind int

	// Entries are the changed/untracked files.
	Entries []StatusEntry
}

// IsDirty reports whether the working tree has any changes.
func (s Status) IsDirty() bool {
	return len(s.Entries) > 0
}

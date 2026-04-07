// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// HistoryEntry records a single executed action and its result.
type HistoryEntry struct {
	Timestamp  time.Time
	RepoPath   string
	ActionName string
	Subjects   []string
	Params     []string // user-provided input parameters (e.g. description text)
	Result     Result
}

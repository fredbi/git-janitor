// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// Issue represents a GitHub issue summary.
type Issue struct {
	Number    int
	Title     string
	State     string // "open", "closed"
	Author    string
	Labels    []string
	CreatedAt time.Time
	UpdatedAt time.Time
	HTMLURL   string
	Detail    *IssueDetail // nil until requested via CollectDetails
}

// IssueDetail holds on-demand detail information for an issue.
type IssueDetail struct {
	Body         string
	CommentCount int
	Assignees    []string
	Tags         []string
}

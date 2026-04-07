// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// PullRequest represents a GitHub pull request summary.
type PullRequest struct {
	Number    int
	Title     string
	State     string // "open", "closed", "merged"
	Author    string
	Branch    string // head branch
	Base      string // base branch
	Draft     bool
	CreatedAt time.Time
	UpdatedAt time.Time
	HTMLURL   string
	Detail    *PullRequestDetail // nil until requested via CollectDetails
}

// PullRequestDetail holds on-demand detail information for a pull request.
type PullRequestDetail struct {
	Body         string
	CommentCount int
	ReviewState  string // "approved", "changes_requested", "pending"
	Mergeable    bool
	Additions    int
	Deletions    int
	ChangedFiles int
	Tags         []string
}

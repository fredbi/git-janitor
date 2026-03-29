// SPDX-License-Identifier: Apache-2.0

package github

import "time"

// RepoData holds GitHub API data for a single repository.
//
// The cheap path (single repos.Get call) populates most fields.
// Expensive fields (UnrespondedIssues, PendingReviewPRs, VulnerabilityAlerts)
// are populated only when the corresponding checks are enabled.
type RepoData struct {
	// Identity.
	Owner    string
	Repo     string
	FullName string // "owner/repo"
	HTMLURL  string

	// Metadata.
	Description   string
	DefaultBranch string
	Topics        []string
	License       string // SPDX identifier or ""
	IsFork        bool
	IsArchived    bool
	IsPrivate     bool

	// Permissions (from repos.Get).
	HasAdminAccess bool // true if the token has admin access
	HasPushAccess  bool // true if the token has push access

	// Counts (from repos.Get).
	OpenIssues int // includes PRs (GitHub API behavior)
	OpenPRs    int // accurate count from pulls.List
	StarCount  int
	ForkCount  int

	// Fork lineage.
	ParentFullName string // "" if not a fork

	// Timestamps.
	CreatedAt time.Time
	UpdatedAt time.Time
	PushedAt  time.Time

	// Expensive fields (populated only when corresponding checks are enabled).
	UnrespondedIssues int // -1 = not fetched
	PendingReviewPRs  int // -1 = not fetched

	// Security alerts (-1 = not fetched / no access, -2 = not queried by config).
	DependabotAlerts     int // open Dependabot alerts
	CodeScanningAlerts   int // open code scanning alerts
	SecretScanningAlerts int // open secret scanning alerts
	SecuritySkipped      bool // true when config says securityAlerts: false

	// Token scopes (from X-OAuth-Scopes header). Empty for fine-grained tokens.
	TokenScopes string

	// Cross-check field: injected by the UX layer from git.RepoInfo.
	LocalDefaultBranch string

	// Err is non-nil if the API call failed.
	Err error
}

// IsRepoInfo satisfies the engine.RepoInfo marker interface.
func (*RepoData) IsRepoInfo() {}

// SecurityAlerts returns the total count of open security alerts,
// or -1 if none of the security APIs were queried.
func (d *RepoData) SecurityAlerts() int {
	total := 0
	queried := false

	for _, n := range []int{d.DependabotAlerts, d.CodeScanningAlerts, d.SecretScanningAlerts} {
		if n >= 0 {
			total += n
			queried = true
		}
	}

	if !queried {
		return -1
	}

	return total
}

// NewRepoData returns a RepoData with expensive fields initialized to -1
// (not fetched).
func NewRepoData(owner, repo string) *RepoData {
	return &RepoData{
		Owner:                owner,
		Repo:                 repo,
		FullName:             owner + "/" + repo,
		UnrespondedIssues:    -1,
		PendingReviewPRs:     -1,
		DependabotAlerts:     -1,
		CodeScanningAlerts:   -1,
		SecretScanningAlerts: -1,
	}
}

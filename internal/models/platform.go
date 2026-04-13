// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// PlatformInfo holds hosting-platform metadata for a repository.
//
// This covers GitHub, GitLab, Gitea, etc. Provider-specific fields
// that don't generalize across platforms are nil/zero when not applicable.
type PlatformInfo struct {
	// Identity.
	Owner     string
	Repo      string
	FullName  string // "owner/repo" or "group/subgroup/project"
	HTMLURL   string
	ProjectID int    // GitLab integer project ID (0 for GitHub)
	WebURL    string // Base URL of the hosting instance (e.g. "https://gitlab.example.com")

	// Metadata.
	Description   string
	DefaultBranch string
	Topics        []string
	License       string // SPDX identifier or ""
	IsFork        bool
	IsArchived    bool
	IsPrivate     bool

	// Permissions (from API).
	HasAdminAccess bool
	HasPushAccess  bool

	// Counts.
	OpenIssues int // includes PRs on GitHub
	OpenPRs    int // accurate count
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
	DependabotAlerts     int
	CodeScanningAlerts   int
	SecretScanningAlerts int
	SecuritySkipped      bool // true when config says securityAlerts: false

	// Token scopes (from X-OAuth-Scopes header). Empty for fine-grained tokens.
	TokenScopes string

	// Branch protection (-1 = not fetched, 0 = no protection, 1 = protected).
	DefaultBranchProtected int

	// DeleteBranchOnMerge tracks the "Automatically delete head branches" setting.
	// For forks, this is checked on the parent (upstream) repo.
	// -1 = not fetched, 0 = disabled, 1 = enabled.
	DeleteBranchOnMerge int

	// ActionsEnabled tracks whether GitHub Actions (CI) is enabled on the fork.
	// -1 = not fetched, 0 = disabled, 1 = enabled.
	ActionsEnabled int

	// Activity data (populated on demand via CollectDetails).
	Issues       []Issue
	PullRequests []PullRequest
	WorkflowRuns []WorkflowRun

	// Cross-check field: injected from git data.
	LocalDefaultBranch string

	// Err is non-nil if the API call failed.
	Err error
}

// NewPlatformInfo returns a PlatformInfo with expensive fields initialized to -1
// (not fetched).
func NewPlatformInfo(owner, repo string) *PlatformInfo {
	return &PlatformInfo{
		Owner:                  owner,
		Repo:                   repo,
		FullName:               owner + "/" + repo,
		UnrespondedIssues:      -1,
		PendingReviewPRs:       -1,
		DependabotAlerts:       -1,
		CodeScanningAlerts:     -1,
		SecretScanningAlerts:   -1,
		DefaultBranchProtected: -1,
		DeleteBranchOnMerge:    -1,
		ActionsEnabled:         -1,
	}
}

// SecurityAlerts returns the total count of open security alerts,
// or -1 if none of the security APIs were queried.
func (d *PlatformInfo) SecurityAlerts() int {
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

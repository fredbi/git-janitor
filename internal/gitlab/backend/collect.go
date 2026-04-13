// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ErrRateLimited is returned when the GitLab API rate limit is too low
// to safely make more requests.
var ErrRateLimited = errors.New("gitlab: rate limited, try again later")

// collectRepoInfo fetches project metadata from the GitLab API.
//
// Cheap path: 1 call (Projects.GetProject) + 1 call (MergeRequests.ListProjectMergeRequests
// per_page=1 for accurate MR count).
// When fetchSecurity is true, also queries vulnerability APIs (requires GitLab Ultimate).
func collectRepoInfo(ctx context.Context, c *Client, projectPath string, fetchSecurity bool) *models.PlatformInfo {
	owner, repo := OwnerAndRepo(projectPath)
	data := models.NewPlatformInfo(owner, repo)

	project, resp, err := c.gl.Projects.GetProject(projectPath, &gitlab.GetProjectOptions{
		Statistics: gitlab.Ptr(true),
	}, gitlab.WithContext(ctx))
	c.updateRate(resp)

	if err != nil {
		data.Err = err

		return data
	}

	populateFromProject(data, project, c.baseURL)

	// Accurate MR count: the OpenIssuesCount from GitLab does NOT include MRs
	// (unlike GitHub), but the project response doesn't have an OpenMRs field.
	// A single MR list call with per_page=1 gives us the total via X-Total header.
	mrCount, err := countOpenMRs(ctx, c, project.ID)
	if err == nil {
		data.OpenPRs = mrCount
	}

	// Branch protection check.
	collectBranchProtection(ctx, c, data, project.ID)

	// Security alerts (requires GitLab Ultimate).
	if fetchSecurity {
		// Phase 2: collectSecurityAlerts(ctx, c, data, project.ID)
		data.SecuritySkipped = true // stub until Phase 2
	} else {
		data.SecuritySkipped = true
	}

	return data
}

// collectBranchProtection checks whether the default branch has protection rules.
// Sets DefaultBranchProtected to 1 (protected) or 0 (not protected).
// GitLab auto-protects default branches, so this will usually be 1.
func collectBranchProtection(ctx context.Context, c *Client, data *models.PlatformInfo, projectID int) {
	if data.DefaultBranch == "" {
		return
	}

	branches, resp, err := c.gl.ProtectedBranches.ListProtectedBranches(projectID,
		&gitlab.ListProtectedBranchesOptions{},
		gitlab.WithContext(ctx),
	)
	c.updateRate(resp)

	if err != nil {
		return // leave as -1 (not fetched)
	}

	for _, pb := range branches {
		if pb.Name == data.DefaultBranch {
			data.DefaultBranchProtected = 1

			return
		}
	}

	data.DefaultBranchProtected = 0
}

func populateFromProject(data *models.PlatformInfo, p *gitlab.Project, baseURL string) {
	data.ProjectID = p.ID
	data.FullName = p.PathWithNamespace
	data.WebURL = baseURL

	if p.Namespace != nil {
		data.Owner = p.Namespace.FullPath
	}

	data.Repo = p.Path
	data.HTMLURL = p.WebURL
	data.Description = p.Description
	data.DefaultBranch = p.DefaultBranch
	data.Topics = p.Topics
	data.IsArchived = p.Archived
	data.IsPrivate = p.Visibility == gitlab.PrivateVisibility

	// Fork detection.
	if p.ForkedFromProject != nil {
		data.IsFork = true
		data.ParentFullName = p.ForkedFromProject.PathWithNamespace
	}

	// Permissions.
	if p.Permissions != nil {
		if p.Permissions.ProjectAccess != nil {
			data.HasPushAccess = p.Permissions.ProjectAccess.AccessLevel >= gitlab.DeveloperPermissions
			data.HasAdminAccess = p.Permissions.ProjectAccess.AccessLevel >= gitlab.MaintainerPermissions
		}

		// Group access can grant higher permissions.
		if p.Permissions.GroupAccess != nil {
			if p.Permissions.GroupAccess.AccessLevel >= gitlab.DeveloperPermissions {
				data.HasPushAccess = true
			}
			if p.Permissions.GroupAccess.AccessLevel >= gitlab.MaintainerPermissions {
				data.HasAdminAccess = true
			}
		}
	}

	// Counts.
	data.OpenIssues = p.OpenIssuesCount
	data.StarCount = p.StarCount
	data.ForkCount = p.ForksCount

	// Timestamps.
	if p.CreatedAt != nil {
		data.CreatedAt = time.Time(*p.CreatedAt)
	}

	if p.LastActivityAt != nil {
		data.UpdatedAt = time.Time(*p.LastActivityAt)
	}

	// Delete-branch-on-merge: GitLab has RemoveSourceBranchAfterMerge.
	if p.RemoveSourceBranchAfterMerge {
		data.DeleteBranchOnMerge = 1
	} else {
		data.DeleteBranchOnMerge = 0
	}

	// No equivalent to GitHub Actions enable/disable toggle.
	data.ActionsEnabled = -1
}

// countOpenMRs fetches the total count of open merge requests using a minimal API call.
func countOpenMRs(ctx context.Context, c *Client, projectID int) (int, error) {
	_, resp, err := c.gl.MergeRequests.ListProjectMergeRequests(projectID,
		&gitlab.ListProjectMergeRequestsOptions{
			State: gitlab.Ptr("opened"),
			ListOptions: gitlab.ListOptions{
				PerPage: 1,
			},
		},
		gitlab.WithContext(ctx),
	)
	c.updateRate(resp)

	if err != nil {
		return 0, err
	}

	// GitLab returns X-Total header with the total count.
	if resp != nil && resp.Response != nil {
		if total := resp.Header.Get("X-Total"); total != "" {
			if n, parseErr := strconv.Atoi(total); parseErr == nil {
				return n, nil
			}
		}
	}

	return 0, nil
}

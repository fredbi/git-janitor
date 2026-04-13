// SPDX-License-Identifier: Apache-2.0

// Package backend provides a GitLab API client for git-janitor.
//
// It fetches project metadata (visibility, fork status, description,
// issues/MR counts, etc.) to power GitLab-specific checks and enrich
// the Facts tab in the TUI.
//
// Authentication uses GITLAB_TOKEN or GL_TOKEN from the environment.
// When no token is available, all GitLab features are silently disabled.
package backend

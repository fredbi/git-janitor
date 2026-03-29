// SPDX-License-Identifier: Apache-2.0

// Package github provides a GitHub API client for git-janitor.
//
// It fetches repository metadata (visibility, fork status, description,
// issues/PR counts, etc.) to power GitHub-specific checks and enrich
// the Facts tab in the TUI.
//
// Authentication uses GITHUB_TOKEN or GH_TOKEN from the environment.
// When no token is available, all GitHub features are silently disabled.
package github

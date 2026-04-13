// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"fmt"
	"net/url"
	"strings"
)

// ExtractProjectPath parses a GitLab remote URL into the full project path.
//
// GitLab project paths are N-segment (group/subgroup/project), unlike
// GitHub's fixed 2-segment owner/repo. The returned path is suitable for
// use with the GitLab API (which accepts URL-encoded paths or integer IDs).
//
// Supported formats:
//   - git@gitlab.example.com:group/subgroup/project.git
//   - https://gitlab.example.com/group/subgroup/project.git
//   - https://gitlab.example.com/group/subgroup/project
//   - ssh://git@gitlab.example.com/group/subgroup/project.git
func ExtractProjectPath(rawURL string) (string, error) {
	path, err := extractPath(rawURL)
	if err != nil {
		return "", err
	}

	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return "", fmt.Errorf("gitlab: empty project path in %q", rawURL)
	}

	// GitLab requires at least 2 segments: namespace/project.
	if !strings.Contains(path, "/") {
		return "", fmt.Errorf("gitlab: cannot extract project path from %q (need at least namespace/project)", rawURL)
	}

	return path, nil
}

// ExtractBaseURL derives the GitLab instance base URL from a remote URL.
//
// Returns the scheme + host portion, e.g. "https://gitlab.example.com".
// For SCP-style SSH URLs (git@host:path), the scheme defaults to "https".
func ExtractBaseURL(rawURL string) (string, error) {
	// SCP-style SSH: git@gitlab.example.com:group/project.git
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
		at := strings.Index(rawURL, "@")
		colon := strings.Index(rawURL[at:], ":")
		host := rawURL[at+1 : at+colon]

		if host == "" {
			return "", fmt.Errorf("gitlab: no host in SCP URL %q", rawURL)
		}

		return "https://" + host, nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("gitlab: invalid URL %q: %w", rawURL, err)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("gitlab: no host in URL %q", rawURL)
	}

	scheme := parsed.Scheme
	if scheme == "ssh" {
		scheme = "https"
	}

	return scheme + "://" + parsed.Host, nil
}

// OwnerAndRepo splits a GitLab project path into owner (namespace) and repo (project name).
//
// For "group/subgroup/project", owner is "group/subgroup" and repo is "project".
// For "group/project", owner is "group" and repo is "project".
func OwnerAndRepo(projectPath string) (owner, repo string) {
	idx := strings.LastIndex(projectPath, "/")
	if idx < 0 {
		return "", projectPath
	}

	return projectPath[:idx], projectPath[idx+1:]
}

// extractPath returns the path portion of a git remote URL,
// handling SSH (git@host:path) and URL schemes (https://, ssh://).
func extractPath(rawURL string) (string, error) {
	// SCP-style SSH: git@gitlab.example.com:group/project.git
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
		colon := strings.LastIndex(rawURL, ":")

		return rawURL[colon+1:], nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("gitlab: invalid URL %q: %w", rawURL, err)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("gitlab: no host in URL %q", rawURL)
	}

	return parsed.Path, nil
}

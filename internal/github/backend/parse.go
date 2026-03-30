// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"fmt"
	"net/url"
	"strings"
)

// ExtractOwnerRepo parses a GitHub remote URL into owner and repo.
//
// Supported formats:
//   - git@github.com:owner/repo.git
//   - https://github.com/owner/repo.git
//   - https://github.com/owner/repo
//   - ssh://git@github.com/owner/repo.git
func ExtractOwnerRepo(rawURL string) (owner, repo string, err error) {
	path, err := extractPath(rawURL)
	if err != nil {
		return "", "", err
	}

	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimPrefix(path, "/")

	const githubPathParts = 2
	parts := strings.SplitN(path, "/", githubPathParts+1)
	if len(parts) < githubPathParts || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github: cannot extract owner/repo from %q", rawURL)
	}

	return parts[0], parts[1], nil
}

// extractPath returns the path portion of a git remote URL,
// handling SSH (git@host:path) and URL schemes (https://, ssh://).
func extractPath(rawURL string) (string, error) {
	// SCP-style SSH: git@github.com:owner/repo.git
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
		colon := strings.LastIndex(rawURL, ":")

		return rawURL[colon+1:], nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("github: invalid URL %q: %w", rawURL, err)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("github: no host in URL %q", rawURL)
	}

	return parsed.Path, nil
}

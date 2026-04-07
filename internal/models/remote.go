// SPDX-License-Identifier: Apache-2.0

package models

// Well-known remote names.
const (
	RemoteOrigin   = "origin"
	RemoteUpstream = "upstream"
)

// Remote represents a git remote with its name and URL.
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// FindRemote returns the Remote with the given name, or nil if not found.
func FindRemote(remotes []Remote, name string) *Remote {
	for i := range remotes {
		if remotes[i].Name == name {
			return &remotes[i]
		}
	}

	return nil
}

// OriginFetchURL returns the fetch URL for the "origin" remote, or empty string.
func OriginFetchURL(remotes []Remote) string {
	for _, rm := range remotes {
		if rm.Name == RemoteOrigin {
			return rm.FetchURL
		}
	}

	return ""
}

// UpstreamFetchURL returns the fetch URL for the "upstream" remote, or empty string.
func UpstreamFetchURL(remotes []Remote) string {
	for _, rm := range remotes {
		if rm.Name == RemoteUpstream {
			return rm.FetchURL
		}
	}

	return ""
}

// HasDistinctRemote reports whether the repo has a remote with a URL
// different from origin's URL (i.e. a potential upstream/fork source).
func HasDistinctRemote(remotes []Remote, normalizeURL func(string) string) (string, bool) {
	originURL := OriginFetchURL(remotes)
	if originURL == "" {
		return "", false
	}

	normOrigin := normalizeURL(originURL)

	for _, rm := range remotes {
		if rm.Name == RemoteOrigin {
			continue
		}

		if rm.FetchURL != "" && normalizeURL(rm.FetchURL) != normOrigin {
			return rm.Name, true
		}
	}

	return "", false
}

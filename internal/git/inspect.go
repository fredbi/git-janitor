package git

import (
	"context"
	"net/url"
	"regexp"
	"strings"
)

// SCM provider constants.
const (
	SCMGitHub = "github"
	SCMGitLab = "gitlab"
	SCMOther  = "other"
	SCMNone   = "no-scm"
)

// Repository kind constants.
const (
	KindClone  = "clone"
	KindFork   = "fork"
	KindNotGit = "not-git"
)

var (
	githubHostRe = regexp.MustCompile(`github\.(\w+)`) //nolint:gochecknoglobals
	gitlabHostRe = regexp.MustCompile(`gitlab\.(\w+)`) //nolint:gochecknoglobals
)

// CollectRepoInfo gathers status, branches, remotes, stashes and default branch,
// then derives SCM, kind, and last commit time.
func CollectRepoInfo(ctx context.Context, r *Runner, path string) RepoInfo {
	info := RepoInfo{Path: path, IsGit: true}

	var err error

	info.Status, err = r.Status(ctx)
	if err != nil {
		info.Err = err

		return info
	}

	info.Branches, err = r.Branches(ctx)
	if err != nil {
		info.Err = err

		return info
	}

	info.Remotes, err = r.Remotes(ctx)
	if err != nil {
		info.Err = err

		return info
	}

	info.Stashes, _ = r.Stashes(ctx)            // non-fatal
	info.DefaultBranch, _ = r.DefaultBranch(ctx) // non-fatal
	info.LastCommit, _ = r.LastCommitTime(ctx)   // non-fatal
	info.Worktrees, _ = r.Worktrees(ctx) // non-fatal

	// Repository traits.
	info.IsShallow = r.IsShallow(ctx)
	info.HasSubmodules = r.HasSubmodules()
	info.HasLFS = r.HasLFS()

	// Repository size and repack advice.
	size := r.Size(ctx)
	info.Size = &size

	// Curated git config.
	cfg := r.Config(ctx)
	info.Config = &cfg

	// File stats: large files, large blobs, binary files.
	fs := r.FileStats(ctx)
	info.FileStats = &fs

	// Tags and derived summary.
	info.Tags, _ = r.Tags(ctx, info.DefaultBranch) // non-fatal
	info.LastTagDate, info.LastSemverTag, info.LastSemverDate = DeriveTagSummary(info.Tags)

	// Commit activity and staleness.
	activity := r.Activity(ctx)
	activity.TagsLast360d = CountTagsInWindow(info.Tags, 360)
	info.Activity = &activity

	// Health check: integrity + GC diagnostics.
	health := r.Health(ctx)
	info.Health = &health

	// Mark branches that have been merged into the default branch.
	// Uses both reachability (--merged) and patch-id comparison (cherry)
	// to catch squash-merged and rebased branches.
	if info.DefaultBranch != "" {
		merged, mergeErr := r.MergedBranches(ctx, info.DefaultBranch)
		if mergeErr != nil {
			merged = make(map[string]bool)
		}

		MarkMerged(ctx, r, info.Branches, info.DefaultBranch, merged)

		// Check merge and rebase feasibility for unmerged local branches.
		r.CheckMergeable(ctx, info.Branches, info.DefaultBranch)
		r.CheckRebaseable(ctx, info.Branches, info.DefaultBranch)
	}

	// Derive SCM and kind from remotes.
	info.SCM = DeriveSCM(info.Remotes)
	info.Kind = DeriveKind(info.Remotes)

	return info
}

// DeriveSCM determines the SCM provider from the origin remote URL.
func DeriveSCM(remotes []Remote) string {
	originURL := OriginFetchURL(remotes)
	if originURL == "" {
		return SCMOther
	}

	host := ExtractHost(originURL)

	switch {
	case githubHostRe.MatchString(host):
		return SCMGitHub
	case gitlabHostRe.MatchString(host):
		return SCMGitLab
	default:
		return SCMOther
	}
}

// DeriveKind determines whether the repo is a clone or a fork.
//
//   - "fork" if origin and upstream exist and point to different hosts or paths
//   - "clone" otherwise (single remote, or all remotes share the same base URL)
func DeriveKind(remotes []Remote) string {
	if len(remotes) == 0 {
		return KindClone
	}

	originURL := OriginFetchURL(remotes)
	upstreamURL := ""

	for _, rm := range remotes {
		if rm.Name == "upstream" {
			upstreamURL = rm.FetchURL

			break
		}
	}

	if upstreamURL == "" || originURL == "" {
		return KindClone
	}

	// Compare normalized URLs (strip scheme).
	if NormalizeURL(originURL) == NormalizeURL(upstreamURL) {
		return KindClone
	}

	return KindFork
}

// OriginFetchURL returns the fetch URL for the "origin" remote, or empty string.
func OriginFetchURL(remotes []Remote) string {
	for _, rm := range remotes {
		if rm.Name == "origin" {
			return rm.FetchURL
		}
	}

	return ""
}

// ExtractHost extracts the hostname from a git remote URL.
// Handles both SSH (git@host:path) and HTTPS (https://host/path) forms.
func ExtractHost(rawURL string) string {
	// SSH form: git@github.com:user/repo.git
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
		at := strings.Index(rawURL, "@")
		colon := strings.Index(rawURL[at:], ":")

		return rawURL[at+1 : at+colon]
	}

	// HTTPS or other scheme.
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return parsed.Hostname()
}

// NormalizeURL strips the scheme from a URL for comparison purposes.
// "https://github.com/user/repo.git" and "git@github.com:user/repo.git"
// both normalize to "github.com/user/repo.git".
func NormalizeURL(rawURL string) string {
	// SSH form.
	if strings.Contains(rawURL, "@") && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
		at := strings.Index(rawURL, "@")
		rest := rawURL[at+1:]
		// Convert "host:path" to "host/path".
		rest = strings.Replace(rest, ":", "/", 1)

		return strings.TrimSuffix(rest, ".git")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	return strings.TrimSuffix(parsed.Host+parsed.Path, ".git")
}

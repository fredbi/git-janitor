package backend

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// Well-known remote names.
// For phases 1-3, we only consider these two remotes.
// Phase 4+ may make these configurable.
const (
	RemoteOrigin   = "origin"
	RemoteUpstream = "upstream"
)

var (
	githubHostRe = regexp.MustCompile(`github\.(\w+)`)
	gitlabHostRe = regexp.MustCompile(`gitlab\.(\w+)`)
)

// CollectRepoInfoFast gathers the essentials for displaying repo facts:
// status, branches, remotes, stashes, default branch, traits, config,
// and last commit. Skips expensive operations (fsck, file stats, health,
// merge/rebase checks, activity) that can take 10+ seconds on large repos.
//
// Use [Runner.CollectRepoInfo] for a full deep inspection (Ctrl+R refresh).
func (r *Runner) CollectRepoInfoFast(ctx context.Context) *RepoInfo {
	info := &RepoInfo{Path: r.Dir, IsGit: true}
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

	info.Stashes, _ = r.Stashes(ctx)             // non-fatal
	info.DefaultBranch, _ = r.DefaultBranch(ctx) // non-fatal
	info.LastCommit, _ = r.LastCommitTime(ctx)   // non-fatal
	info.Worktrees, _ = r.Worktrees(ctx)         // non-fatal

	// Lightweight traits (filesystem checks, no git commands).
	info.HasSubmodules = r.HasSubmodules()
	info.HasLFS = r.HasLFS()

	// Curated git config (single git command).
	cfg := r.Config(ctx)
	info.Config = &cfg

	// Derive SCM and kind from remotes.
	info.SCM = DeriveSCM(info.Remotes)
	info.Kind = DeriveKind(info.Remotes)

	return info
}

// CollectRepoInfo gathers status, branches, remotes, stashes and default branch,
// then derives SCM, kind, and last commit time.
func (r *Runner) CollectRepoInfo(ctx context.Context) *RepoInfo {
	return r.collectRepoInfo(ctx)
}

func (r *Runner) collectRepoInfo(ctx context.Context) *RepoInfo {
	info := &RepoInfo{Path: r.Dir, IsGit: true}

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

	info.Stashes, _ = r.Stashes(ctx)             // non-fatal
	info.DefaultBranch, _ = r.DefaultBranch(ctx) // non-fatal
	info.LastCommit, _ = r.LastCommitTime(ctx)   // non-fatal
	info.Worktrees, _ = r.Worktrees(ctx)         // non-fatal

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
func DeriveSCM(remotes []Remote) models.RepoSCM {
	originURL := OriginFetchURL(remotes)
	if originURL == "" {
		return models.SCMOther
	}

	host := ExtractHost(originURL)

	switch {
	case githubHostRe.MatchString(host):
		return models.SCMGitHub
	case gitlabHostRe.MatchString(host):
		return models.SCMGitLab
	default:
		return models.SCMOther
	}
}

// DeriveKind determines whether the repo is a clone or a fork.
//
// A repo is a "fork" if it has at least two remotes with distinct normalized URLs.
// This catches cases where the upstream remote is misspelled (e.g. "upstram").
//
// A repo is a "clone" if it has zero or one unique remote URL.
func DeriveKind(remotes []Remote) models.RepoKind {
	if len(remotes) <= 1 {
		return models.RepoKindClone
	}

	// Collect distinct normalized URLs across all remotes.
	seen := make(map[string]bool, len(remotes))

	for _, rm := range remotes {
		if rm.FetchURL != "" {
			seen[NormalizeURL(rm.FetchURL)] = true
		}
	}

	if len(seen) >= 2 {
		return models.RepoKindFork
	}

	return models.RepoKindClone
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

// FindRemote returns the Remote with the given name, or nil if not found.
func FindRemote(remotes []Remote, name string) *Remote {
	for i := range remotes {
		if remotes[i].Name == name {
			return &remotes[i]
		}
	}

	return nil
}

// HasDistinctRemote reports whether the repo has a remote with a URL
// different from origin's URL (i.e. a potential upstream/fork source).
// Returns the name of the first such remote, or empty string.
func HasDistinctRemote(remotes []Remote) (string, bool) {
	originURL := OriginFetchURL(remotes)
	if originURL == "" {
		return "", false
	}

	normOrigin := NormalizeURL(originURL)

	for _, rm := range remotes {
		if rm.Name == RemoteOrigin {
			continue
		}

		if rm.FetchURL != "" && NormalizeURL(rm.FetchURL) != normOrigin {
			return rm.Name, true
		}
	}

	return "", false
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

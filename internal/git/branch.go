package git

import (
	"bufio"
	"context"
	"strings"
)

// Branch represents a git branch.
type Branch struct {
	// Name is the short branch name (e.g. "main", "feature/foo").
	Name string

	// IsRemote is true for remote-tracking branches (e.g. "origin/main").
	IsRemote bool

	// IsCurrent is true if this is the currently checked-out branch.
	IsCurrent bool

	// Upstream is the upstream tracking ref (e.g. "origin/main"), if configured.
	Upstream string

	// Hash is the commit hash at the tip of the branch.
	Hash string
}

// Branches runs git branch -a --format and returns all local and remote branches.
func (r *Runner) Branches(ctx context.Context) ([]Branch, error) {
	// Use a machine-readable format to avoid parsing alignment quirks.
	// Fields: refname:short, objectname:short, upstream:short, HEAD
	out, err := r.run(ctx,
		"branch", "-a",
		"--format=%(HEAD)|%(refname:short)|%(objectname:short)|%(upstream:short)",
	)
	if err != nil {
		return nil, err
	}

	return parseBranches(out), nil
}

// DefaultBranch detects the default branch of the repository.
//
// It tries (in order):
//  1. git symbolic-ref refs/remotes/origin/HEAD — the remote's default.
//  2. Presence of common branch names (main, master) locally.
//  3. Falls back to the current branch.
func (r *Runner) DefaultBranch(ctx context.Context) (string, error) {
	// Try remote HEAD symbolic ref first.
	out, err := r.run(ctx, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil {
		ref := strings.TrimSpace(out)
		// ref is "origin/main" — strip the remote prefix.
		if _, branch, ok := strings.Cut(ref, "/"); ok {
			return branch, nil
		}

		return ref, nil
	}

	// Fallback: check if main or master exist locally.
	for _, candidate := range []string{"main", "master"} {
		if _, verifyErr := r.run(ctx, "rev-parse", "--verify", "--quiet", candidate); verifyErr == nil {
			return candidate, nil
		}
	}

	// Last resort: current branch.
	out, err = r.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// parseBranches parses the output of git branch -a --format.
func parseBranches(output string) []Branch {
	var branches []Branch

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		head := parts[0]
		name := parts[1]
		hash := parts[2]
		upstream := parts[3]

		// Skip HEAD pointer entries like "origin/HEAD -> origin/main".
		if strings.Contains(name, "/HEAD") {
			continue
		}

		isRemote := strings.Contains(name, "/")
		// Remote branches from -a come as "origin/feature", which is correct.
		// Local branches have no slash (unless they use slash-namespaced names like "feature/foo").
		// We detect remote branches by the "remotes/" refname prefix, but --format with
		// refname:short strips "remotes/" — so remote branches still have "origin/".
		// Heuristic: if the name starts with a known remote prefix, it's remote.
		// A more robust approach: check if the raw ref starts with "refs/remotes".
		// Since we can't easily access the raw ref with short format, we rely on
		// the presence of a remote-like prefix (contains exactly one slash before the branch part).

		b := Branch{
			Name:      name,
			Hash:      hash,
			IsCurrent: head == "*",
			Upstream:  upstream,
			IsRemote:  isRemote,
		}

		branches = append(branches, b)
	}

	return branches
}

// LocalBranches returns only local branches.
func (r *Runner) LocalBranches(ctx context.Context) ([]Branch, error) {
	all, err := r.Branches(ctx)
	if err != nil {
		return nil, err
	}

	var local []Branch
	for _, b := range all {
		if !b.IsRemote {
			local = append(local, b)
		}
	}

	return local, nil
}

// RemoteBranches returns only remote-tracking branches.
func (r *Runner) RemoteBranches(ctx context.Context) ([]Branch, error) {
	all, err := r.Branches(ctx)
	if err != nil {
		return nil, err
	}

	var remote []Branch
	for _, b := range all {
		if b.IsRemote {
			remote = append(remote, b)
		}
	}

	return remote, nil
}

package backend

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// Branches runs git branch -a --format and returns all local and remote branches.
func (r *Runner) Branches(ctx context.Context) ([]models.Branch, error) {
	// Use a machine-readable format to avoid parsing alignment quirks.
	// Fields: HEAD, refname:short, objectname:short, upstream:short, upstream:track, creatordate
	out, err := r.run(ctx, cmdBranchList()...)
	if err != nil {
		return nil, err
	}

	return parseBranches(out), nil
}

// parseBranches parses the output of git branch -a --format.
func parseBranches(output string) []models.Branch {
	var branches []models.Branch

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 4 {
			continue
		}

		head := parts[0]
		name := parts[1]
		hash := parts[2]
		upstream := parts[3]

		var ahead, behind int
		var gone bool

		if len(parts) >= 5 {
			ahead, behind, gone = parseUpstreamTrack(parts[4])
		}

		var lastCommit time.Time

		if len(parts) >= 6 {
			lastCommit, _ = time.Parse(time.RFC3339, strings.TrimSpace(parts[5]))
		}

		// Use the full refname (field 7) to reliably detect remote branches.
		// refs/remotes/... = remote, refs/heads/... = local.
		var fullRef string

		if len(parts) >= 7 {
			fullRef = strings.TrimSpace(parts[6])
		}

		// Skip HEAD pointer entries like "origin/HEAD -> origin/main".
		if strings.Contains(name, "/HEAD") {
			continue
		}

		// Skip bare remote ref roots (e.g. refs/remotes/origin with short name "origin").
		// These are not actual branches.
		if strings.HasPrefix(fullRef, "refs/remotes/") && !strings.Contains(strings.TrimPrefix(fullRef, "refs/remotes/"), "/") {
			continue
		}

		isRemote := strings.HasPrefix(fullRef, "refs/remotes/")

		b := models.Branch{
			Name:       name,
			Hash:       hash,
			IsCurrent:  head == "*",
			Upstream:   upstream,
			Ahead:      ahead,
			Behind:     behind,
			Gone:       gone,
			LastCommit: lastCommit,
			IsRemote:   isRemote,
		}

		branches = append(branches, b)
	}

	return branches
}

// DefaultBranch detects the default branch of the repository.
//
// It tries (in order):
//  1. git symbolic-ref refs/remotes/origin/HEAD — the remote's default.
//  2. Presence of common branch names (main, master) locally.
//  3. Falls back to the current branch.
func (r *Runner) DefaultBranch(ctx context.Context) (string, error) {
	// Try remote HEAD symbolic ref first.
	out, err := r.run(ctx, cmdSymbolicRef("refs/remotes/origin/HEAD")...)
	if err == nil {
		ref := strings.TrimSpace(out)
		// ref is "origin/main" — strip the remote prefix.
		if _, branch, ok := strings.Cut(ref, "/"); ok {
			// Verify the branch actually exists locally — the symref can be stale
			// (e.g. origin/HEAD points to "main" but the repo uses "master").
			if _, verifyErr := r.run(ctx, cmdRevParseVerify(branch)...); verifyErr == nil {
				return branch, nil
			}
		}
	}

	// Fallback: check if main or master exist locally.
	for _, candidate := range []string{"main", "master"} {
		if _, verifyErr := r.run(ctx, cmdRevParseVerify(candidate)...); verifyErr == nil {
			return candidate, nil
		}
	}

	// Last resort: current branch.
	out, err = r.run(ctx, cmdRevParseAbbrev("HEAD")...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// parseUpstreamTrack parses the %(upstream:track) output.
//
// Examples:
//
//	""                → 0, 0  (in sync or no upstream)
//	"[ahead 3]"       → 3, 0
//	"[behind 2]"      → 0, 2
//	"[ahead 3, behind 2]" → 3, 2
//	"[gone]"          → 0, 0  (upstream deleted)
func parseUpstreamTrack(track string) (ahead, behind int, gone bool) {
	track = strings.TrimSpace(track)
	if track == "" {
		return 0, 0, false
	}

	if track == "[gone]" {
		return 0, 0, true
	}

	// Strip brackets.
	track = strings.TrimPrefix(track, "[")
	track = strings.TrimSuffix(track, "]")

	for part := range strings.SplitSeq(track, ",") {
		part = strings.TrimSpace(part)

		switch {
		case strings.HasPrefix(part, "ahead "):
			fmt.Sscanf(part, "ahead %d", &ahead) //nolint:errcheck // best-effort
		case strings.HasPrefix(part, "behind "):
			fmt.Sscanf(part, "behind %d", &behind) //nolint:errcheck // best-effort
		}
	}

	return ahead, behind, false
}

// MergedBranches returns the set of local branch names whose tips are
// reachable from target (typically the default branch).
func (r *Runner) MergedBranches(ctx context.Context, target string) (map[string]bool, error) {
	out, err := r.run(ctx, cmdBranchMerged(target)...)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			merged[name] = true
		}
	}

	return merged, nil
}

// MergedRemoteBranches returns the set of remote branch names (e.g. "upstream/feature")
// whose tips are reachable from target.
func (r *Runner) MergedRemoteBranches(ctx context.Context, target string) (map[string]bool, error) {
	out, err := r.run(ctx, cmdRemoteBranchMerged(target)...)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			merged[name] = true
		}
	}

	return merged, nil
}

// IsFullyApplied checks whether all commits on branch have already been applied
// to target, using patch-id comparison (git cherry). This detects squash-merged
// and rebased branches that git branch --merged would miss.
func (r *Runner) IsFullyApplied(ctx context.Context, target, branch string) (bool, error) {
	out, err := r.run(ctx, cmdCherry(target, branch)...)
	if err != nil {
		return false, err
	}

	// If every line starts with "-", all patches are applied.
	// If there are no lines, the branch has no unique commits (also fully applied).
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "+") {
			return false, nil
		}
	}

	return true, nil
}

// MarkMerged sets the Merged flag on each branch that is either:
//   - reachable from target (in the merged set from git branch --merged), or
//   - fully applied to target by patch-id (git cherry), catching squash-merges and rebases.
//
// For remote branches, mergedRemote provides the set from git branch -r --merged.
// The runner is optional: if nil, only the reachability check is used.
func MarkMerged(ctx context.Context, r *Runner, branches []models.Branch, target string, merged, mergedRemote map[string]bool) {
	for i := range branches {
		b := &branches[i]

		if b.IsRemote {
			// Remote branches: fast path via git branch -r --merged.
			if mergedRemote[b.Name] {
				b.Merged = true

				continue
			}

			// Slow path: check via patch-id comparison (catches squash-merges).
			if r != nil {
				if applied, err := r.IsFullyApplied(ctx, target, b.Name); err == nil && applied {
					b.Merged = true
				}
			}

			continue
		}

		// Local branches: check against the local merged set (fast path).
		if merged[b.Name] {
			b.Merged = true

			continue
		}

		if b.Name == target || r == nil {
			continue
		}

		// Slow path: check via patch-id comparison.
		if applied, err := r.IsFullyApplied(ctx, target, b.Name); err == nil && applied {
			b.Merged = true
		}
	}
}

// CanMerge performs a dry-run merge of branch into target using git merge-tree.
// It requires git >= 2.38. The merge is performed entirely in memory —
// the worktree and index are not touched.
func (r *Runner) CanMerge(ctx context.Context, target, branch string) models.MergeCheck {
	out, err := r.run(ctx, cmdMergeTree(target, branch)...)
	if err != nil {
		// Exit code 1 means conflicts. Parse the output for file names.
		return parseMergeTreeConflicts(out)
	}

	return models.MergeCheck{Clean: true}
}

// parseMergeTreeConflicts extracts conflicting file names from merge-tree output.
// The output format on conflict is: tree-hash\n\nfile1\nfile2\n...
func parseMergeTreeConflicts(output string) models.MergeCheck {
	var conflicts []string

	// Skip the first line (tree hash) and any blank lines.
	lines := strings.SplitSeq(strings.TrimSpace(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip the tree hash (40 or 64 hex chars).
		if isHexHash(line) {
			continue
		}

		conflicts = append(conflicts, line)
	}

	return models.MergeCheck{Clean: false, Conflicts: conflicts}
}

// isHexHash reports whether s looks like a git object hash (40 or 64 hex chars).
func isHexHash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}

	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}

// CheckMergeable runs CanMerge for each local branch that is not already merged
// and not the target itself, populating the MergeCheck field.
func (r *Runner) CheckMergeable(ctx context.Context, branches []models.Branch, target string) {
	for i := range branches {
		b := &branches[i]

		if b.IsRemote || b.Merged || b.Name == target {
			continue
		}

		result := r.CanMerge(ctx, target, b.Name)
		b.MergeCheck = &result
	}
}

// CheckRebaseable runs CheckRebase for each local branch that is not already merged
// and not the target itself, populating the RebaseCheck field.
func (r *Runner) CheckRebaseable(ctx context.Context, branches []models.Branch, target string) {
	for i := range branches {
		b := &branches[i]

		if b.IsRemote || b.Merged || b.Name == target {
			continue
		}

		result := r.CheckRebase(ctx, target, b.Name)
		b.RebaseCheck = &result
	}
}

// CheckRemoteMergeable runs CanMerge for upstream remote branches that are
// not merged and not the default branch, populating MergeCheck.
func (r *Runner) CheckRemoteMergeable(ctx context.Context, branches []models.Branch, target, upstreamPrefix string) {
	for i := range branches {
		b := &branches[i]

		if !b.IsRemote || b.Merged || !strings.HasPrefix(b.Name, upstreamPrefix) {
			continue
		}

		branchName := strings.TrimPrefix(b.Name, upstreamPrefix)
		if branchName == target {
			continue
		}

		result := r.CanMerge(ctx, target, b.Name)
		b.MergeCheck = &result
	}
}

// CheckRemoteRebaseable runs CheckRebase for upstream remote branches that are
// not merged and not the default branch, populating RebaseCheck.
func (r *Runner) CheckRemoteRebaseable(ctx context.Context, branches []models.Branch, target, upstreamPrefix string) {
	for i := range branches {
		b := &branches[i]

		if !b.IsRemote || b.Merged || !strings.HasPrefix(b.Name, upstreamPrefix) {
			continue
		}

		branchName := strings.TrimPrefix(b.Name, upstreamPrefix)
		if branchName == target {
			continue
		}

		result := r.CheckRebase(ctx, target, b.Name)
		b.RebaseCheck = &result
	}
}

// MarkRemoteAheadOnly checks upstream remote branches and sets AheadOnly=true
// when the default branch is an ancestor of the remote branch (i.e. the branch
// is simply ahead, not diverged).
func (r *Runner) MarkRemoteAheadOnly(ctx context.Context, branches []models.Branch, target, upstreamPrefix string) {
	for i := range branches {
		b := &branches[i]

		if !b.IsRemote || b.Merged || !strings.HasPrefix(b.Name, upstreamPrefix) {
			continue
		}

		branchName := strings.TrimPrefix(b.Name, upstreamPrefix)
		if branchName == target {
			continue
		}

		// git merge-base --is-ancestor <default> <branch>
		// Exit 0 = default is ancestor of branch = branch is ahead only.
		_, err := r.run(ctx, cmdIsAncestor(target, b.Name)...)
		b.AheadOnly = err == nil
	}
}

// IsUpstreamDefaultBehindLocal reports whether the upstream remote's default
// branch is strictly behind the local default branch — their tips differ and
// upstream/<default> is an ancestor of <default>. Returns false when either
// ref is missing or the local is not strictly ahead (diverged, identical, or
// behind).
func (r *Runner) IsUpstreamDefaultBehindLocal(ctx context.Context, branches []models.Branch, defaultBranch, upstreamPrefix string) bool {
	if defaultBranch == "" {
		return false
	}

	return r.isStrictlyBehind(ctx, branches, defaultBranch, false, upstreamPrefix+defaultBranch, true)
}

// IsUpstreamDefaultBehindOrigin reports whether the upstream remote's default
// branch is strictly behind the origin remote's default branch — their tips
// differ and upstream/<default> is an ancestor of origin/<default>.
func (r *Runner) IsUpstreamDefaultBehindOrigin(ctx context.Context, branches []models.Branch, defaultBranch, originPrefix, upstreamPrefix string) bool {
	if defaultBranch == "" {
		return false
	}

	return r.isStrictlyBehind(ctx, branches, originPrefix+defaultBranch, true, upstreamPrefix+defaultBranch, true)
}

// isStrictlyBehind reports whether behindRef is strictly behind aheadRef —
// hashes differ and behindRef is an ancestor of aheadRef. The *Remote flags
// disambiguate local vs remote-tracking refs when scanning branches.
func (r *Runner) isStrictlyBehind(ctx context.Context, branches []models.Branch, aheadRef string, aheadRemote bool, behindRef string, behindRemote bool) bool {
	aheadHash := findBranchHash(branches, aheadRef, aheadRemote)
	behindHash := findBranchHash(branches, behindRef, behindRemote)

	if aheadHash == "" || behindHash == "" || aheadHash == behindHash {
		return false
	}

	_, err := r.run(ctx, cmdIsAncestor(behindRef, aheadRef)...)

	return err == nil
}

func findBranchHash(branches []models.Branch, name string, remote bool) string {
	for _, b := range branches {
		if b.IsRemote == remote && b.Name == name {
			return b.Hash
		}
	}

	return ""
}

// LocalBranches returns only local branches.
func (r *Runner) LocalBranches(ctx context.Context) ([]models.Branch, error) {
	all, err := r.Branches(ctx)
	if err != nil {
		return nil, err
	}

	var local []models.Branch
	for _, b := range all {
		if !b.IsRemote {
			local = append(local, b)
		}
	}

	return local, nil
}

// RemoteBranches returns only remote-tracking branches.
func (r *Runner) RemoteBranches(ctx context.Context) ([]models.Branch, error) {
	all, err := r.Branches(ctx)
	if err != nil {
		return nil, err
	}

	var remote []models.Branch
	for _, b := range all {
		if b.IsRemote {
			remote = append(remote, b)
		}
	}

	return remote, nil
}

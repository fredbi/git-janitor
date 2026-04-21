// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// BranchUniqueBytes returns the on-disk size of objects reachable from
// ref "target" but from no other ref in the repository. This is the
// storage a subsequent `run-gc-deep-clean` could reclaim if the caller
// deleted the ref first.
//
// Computes: git rev-list --objects --disk-usage <target> ^<ref1> ^<ref2> …
// where the exclusions are every ref returned by `git for-each-ref`
// except the target itself.
//
// Returns -1 on error (the command is best-effort; never fatal).
func (r *Runner) BranchUniqueBytes(ctx context.Context, target string) int64 {
	refs, err := r.allRefNames(ctx)
	if err != nil {
		return -1
	}

	exclude := make([]string, 0, len(refs))

	for _, ref := range refs {
		if ref == target {
			continue
		}

		exclude = append(exclude, ref)
	}

	out, err := r.run(ctx, cmdRevListUniqueDiskUsage(target, exclude)...)
	if err != nil {
		return -1
	}

	var bytes int64

	fmt.Sscanf(strings.TrimSpace(out), "%d", &bytes) //nolint:errcheck // best-effort

	return bytes
}

// MarkBranchUniqueBytes populates Branch.UniqueBytes for every local,
// non-default branch in the slice. Remote-tracking branches and the
// default branch are left with UniqueBytes = -1 (not computed).
//
// Each branch triggers one git rev-list invocation; on a repo with
// many branches this is the dominant cost of full collection, but
// each call only traverses objects uniquely reachable from that ref.
func (r *Runner) MarkBranchUniqueBytes(ctx context.Context, branches []models.Branch, defaultBranch string) {
	refs, err := r.allRefNames(ctx)
	if err != nil {
		for i := range branches {
			branches[i].UniqueBytes = -1
		}

		return
	}

	for i := range branches {
		b := &branches[i]
		b.UniqueBytes = -1

		if b.IsRemote || b.Name == defaultBranch || b.Hash == "" {
			continue
		}

		b.UniqueBytes = uniqueDiskUsageExcluding(ctx, r, "refs/heads/"+b.Name, refs)
	}
}

// uniqueDiskUsageExcluding computes `rev-list --objects --disk-usage
// <target> ^<other>...` excluding every ref equal to target.
func uniqueDiskUsageExcluding(ctx context.Context, r *Runner, target string, refs []string) int64 {
	exclude := make([]string, 0, len(refs))

	for _, ref := range refs {
		if ref == target {
			continue
		}

		exclude = append(exclude, ref)
	}

	out, err := r.run(ctx, cmdRevListUniqueDiskUsage(target, exclude)...)
	if err != nil {
		return -1
	}

	var bytes int64

	fmt.Sscanf(strings.TrimSpace(out), "%d", &bytes) //nolint:errcheck // best-effort

	return bytes
}

// allRefNames enumerates every ref name in the repository.
func (r *Runner) allRefNames(ctx context.Context) ([]string, error) {
	out, err := r.run(ctx, cmdForEachRefNames()...)
	if err != nil {
		return nil, err
	}

	var refs []string

	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			refs = append(refs, line)
		}
	}

	return refs, nil
}

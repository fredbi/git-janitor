// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// CollectBranchDetail fetches on-demand detail for a single branch:
// last commit message and diff stat against the default branch.
func (r *Runner) CollectBranchDetail(ctx context.Context, branchName, defaultBranch string) *models.BranchDetail {
	detail := &models.BranchDetail{}

	// Last commit message.
	if out, err := r.run(ctx, cmdLogMessage(branchName)...); err == nil {
		detail.LastCommitMessage = strings.TrimSpace(out)
	}

	// Diff stat against default branch.
	if defaultBranch != "" && branchName != defaultBranch {
		rangeSpec := defaultBranch + "..." + branchName
		if out, err := r.run(ctx, cmdDiffStat(rangeSpec)...); err == nil {
			detail.DiffStat = strings.TrimSpace(out)
		}
	}

	return detail
}

// CollectStashDetail fetches on-demand detail for a single stash entry.
func (r *Runner) CollectStashDetail(ctx context.Context, stashRef string) *models.StashDetail {
	detail := &models.StashDetail{}

	if out, err := r.run(ctx, cmdStashShow(stashRef)...); err == nil {
		detail.DiffStat = strings.TrimSpace(out)
	}

	return detail
}

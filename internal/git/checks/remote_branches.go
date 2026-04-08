// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// RemoteBranchMergedNotDeleted detects remote branches on the "upstream" remote
// (the fork) that are already merged into the default branch and can be deleted.
//
// This only applies to fork repos (where "upstream" points to the user's fork).
type RemoteBranchMergedNotDeleted struct {
	gitCheck
}

func NewRemoteBranchMergedNotDeleted() RemoteBranchMergedNotDeleted {
	return RemoteBranchMergedNotDeleted{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"remote-branch-merged-not-deleted",
				"detects merged remote branches on the upstream (fork) that can be deleted",
			),
		},
	}
}

func (c RemoteBranchMergedNotDeleted) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c RemoteBranchMergedNotDeleted) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	// Only applies when there's an upstream remote (fork setup).
	if models.FindRemote(info.Remotes, models.RemoteUpstream) == nil {
		return noAlert(c.Name())
	}

	upstreamPrefix := models.RemoteUpstream + "/"

	subjects := filterBranches(info, func(b models.Branch) bool {
		// Only remote branches on the upstream remote.
		if !b.IsRemote || !strings.HasPrefix(b.Name, upstreamPrefix) {
			return false
		}

		// Skip the default branch on the upstream.
		branchName := strings.TrimPrefix(b.Name, upstreamPrefix)
		if branchName == info.DefaultBranch {
			return false
		}

		// Must be merged.
		return b.Merged
	})

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	suggestion := models.ActionSuggestion{
		ActionName:  "delete-remote-branch",
		SubjectKind: models.SubjectBranch,
		Subjects:    subjects,
	}

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityLow,
		Summary:     fmt.Sprintf("%d merged branch(es) on upstream can be deleted", len(subjects)),
		Detail:      subjectsDetail(subjects),
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

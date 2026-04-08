// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// RebaseRemoteBranch rebases a remote branch onto the default branch
// using a temporary worktree, then force-pushes with --force-with-lease.
//
// Subjects are remote branch names in "remote/branch" form (e.g. "upstream/feature").
type RebaseRemoteBranch struct {
	gitAction
}

func NewRebaseRemoteBranch() RebaseRemoteBranch {
	return RebaseRemoteBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"rebase-remote-branch",
				"rebase a remote branch onto the default branch and force-push",
			),
		},
	}
}

func (RebaseRemoteBranch) ApplyTo() models.SubjectKind { return models.SubjectBranch }
func (RebaseRemoteBranch) Destructive() bool           { return true }

func (a RebaseRemoteBranch) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a RebaseRemoteBranch) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, subjects []string) (models.Result, error) {
	if info == nil || info.DefaultBranch == "" {
		return models.Result{}, errors.New("repo info with default branch is required")
	}

	var succeeded, failed int
	var lastErr string

	for _, fullName := range subjects {
		remote, branchName, ok := strings.Cut(fullName, "/")
		if !ok {
			failed++
			lastErr = "invalid subject: " + fullName

			continue
		}

		branch := models.Branch{
			Name:     branchName,
			IsRemote: true,
			Upstream: remote + "/" + branchName,
		}

		result := r.RebaseBranchRemote(ctx, info.DefaultBranch, branch)
		if result.OK {
			succeeded++
		} else {
			failed++
			lastErr = result.Message
		}
	}

	msg := fmt.Sprintf("%d rebased", succeeded)
	if failed > 0 {
		msg += fmt.Sprintf(", %d failed (last: %s)", failed, lastErr)
	}

	return models.Result{OK: failed == 0, Message: msg}, nil
}

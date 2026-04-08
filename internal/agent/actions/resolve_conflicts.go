// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agentbackend "github.com/fredbi/git-janitor/internal/agent/backend"
	"github.com/fredbi/git-janitor/internal/agent/prompts"
	gitbackend "github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

const dryRunFlag = "dry-run"

// ResolveConflicts uses an AI agent to resolve merge conflicts on a branch.
//
// Two-phase execution:
//   - Phase 1 (dry-run): builds and returns the prompt for user review
//   - Phase 2 (execute): runs the full pipeline in a worktree
//
// Params from the check:
//   - params[0]: subject (branch name, e.g. "upstream/feature")
//   - params[1]: target branch (e.g. "master")
//   - params[2]: JSON-encoded conflict info
//   - params[3]: (optional) "dry-run" flag
type ResolveConflicts struct {
	agentAction
}

func NewResolveConflicts() ResolveConflicts {
	return ResolveConflicts{
		agentAction: agentAction{
			Describer: models.NewDescriber(
				"agent-resolve-conflicts",
				"use AI agent to resolve merge conflicts",
			),
		},
	}
}

func (ResolveConflicts) ApplyTo() models.SubjectKind { return models.SubjectBranch }

func (a ResolveConflicts) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	if len(params) < 3 { //nolint:mnd // need branch, target, conflict info
		return models.Result{}, errors.New("agent-resolve-conflicts requires [branch, target, conflictInfo] params")
	}

	branchName := params[0]
	targetBranch := params[1]

	conflictInfo, err := prompts.ParseConflictInfo(params[2]) //nolint:mnd // index 2
	if err != nil {
		return models.Result{}, fmt.Errorf("parsing conflict info: %w", err)
	}

	// Check for dry-run flag.
	isDryRun := len(params) > 3 && params[3] == dryRunFlag //nolint:mnd // index 3

	if isDryRun {
		// Phase 1: generate and return the prompt for review.
		prompt := prompts.BuildResolveConflictsPrompt("/tmp/worktree", branchName, targetBranch, conflictInfo)

		return models.Result{
			OK:      true,
			Message: prompt,
		}, nil
	}

	// Phase 2: full execution.
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, branchName, targetBranch, conflictInfo)
}

func (a ResolveConflicts) execute(
	ctx context.Context,
	agentRunner *agentbackend.Runner,
	info *models.RepoInfo,
	branchName, targetBranch string,
	conflicts prompts.ConflictInfo,
) (models.Result, error) {
	// We need the git runner for worktree operations.
	gitRunner := gitbackend.NewRunner(info.Path)
	gitRunner.StartLogging()

	// Parse the remote and short branch name.
	remote, shortBranch, ok := strings.Cut(branchName, "/")
	if !ok {
		// Local branch — use it directly.
		remote = ""
		shortBranch = branchName
	}

	// Create a temporary worktree with a new branch from the target.
	tmpDir, err := os.MkdirTemp("", "janitor-agent-*")
	if err != nil {
		return models.Result{}, fmt.Errorf("cannot create temp directory: %w", err)
	}

	wtBranch := "janitor-agent-" + shortBranch
	wtPath := filepath.Join(tmpDir, wtBranch)

	defer func() {
		gitRunner.Run(ctx, "worktree", "remove", "--force", wtPath) //nolint:errcheck // best-effort
		os.RemoveAll(tmpDir)                                         //nolint:errcheck // best-effort
		gitRunner.Run(ctx, "branch", "-D", wtBranch)                 //nolint:errcheck // best-effort cleanup
	}()

	// Create worktree from the branch.
	_, err = gitRunner.Run(ctx, "worktree", "add", "-b", wtBranch, wtPath, branchName)
	if err != nil {
		return models.Result{Message: fmt.Sprintf("worktree creation failed: %v", err)}, nil
	}

	wtRunner := gitbackend.NewRunner(wtPath)
	wtRunner.StartLogging()

	// Squash all commits on top of target into one.
	// First, reset to target keeping changes staged.
	_, err = wtRunner.Run(ctx, "reset", "--soft", targetBranch)
	if err != nil {
		return models.Result{Message: fmt.Sprintf("reset --soft %s failed: %v", targetBranch, err)}, nil
	}

	// Commit the squashed changes.
	_, err = wtRunner.Run(ctx, "commit", "-m", fmt.Sprintf("squashed %s for conflict resolution", shortBranch))
	if err != nil {
		return models.Result{Message: fmt.Sprintf("squash commit failed: %v", err)}, nil
	}

	// Attempt rebase onto target — this will produce conflicts.
	_, rebaseErr := wtRunner.Run(ctx, "rebase", targetBranch)
	if rebaseErr == nil {
		// No conflicts! The squash+rebase succeeded cleanly.
		return a.pushResult(ctx, wtRunner, gitRunner, remote, shortBranch, wtBranch)
	}

	// Build the prompt and invoke the agent.
	prompt := prompts.BuildResolveConflictsPrompt(wtPath, shortBranch, targetBranch, conflicts)

	agentRunner.WorkDir = wtPath

	output, agentErr := agentRunner.Run(ctx, prompt)
	if agentErr != nil {
		// Abort the rebase to leave the worktree clean for cleanup.
		wtRunner.Run(ctx, "rebase", "--abort") //nolint:errcheck // best-effort

		return models.Result{
			Message:    fmt.Sprintf("agent failed: %v", agentErr),
			CommandLog: append(gitRunner.Commands(), wtRunner.Commands()...),
		}, nil
	}

	// Verify no conflict markers remain.
	grepOut, _ := wtRunner.Run(ctx, "grep", "-r", "<<<<<<<", ".")
	if strings.TrimSpace(grepOut) != "" {
		wtRunner.Run(ctx, "rebase", "--abort") //nolint:errcheck // best-effort

		return models.Result{
			Message:    "agent did not fully resolve conflicts — markers remain",
			CommandLog: append(gitRunner.Commands(), wtRunner.Commands()...),
		}, nil
	}

	// Stage everything and continue the rebase.
	wtRunner.Run(ctx, "add", "-A")                //nolint:errcheck // best-effort
	wtRunner.Run(ctx, "rebase", "--continue") //nolint:errcheck // best-effort

	_ = output // agent output logged but not used further

	return a.pushResult(ctx, wtRunner, gitRunner, remote, shortBranch, wtBranch)
}

func (a ResolveConflicts) pushResult(
	ctx context.Context,
	wtRunner, gitRunner *gitbackend.Runner,
	remote, shortBranch, wtBranch string,
) (models.Result, error) {
	if remote == "" {
		// Local branch — just update the ref.
		return models.Result{
			OK:         true,
			Message:    fmt.Sprintf("resolved conflicts on %s (local)", shortBranch),
			CommandLog: append(gitRunner.Commands(), wtRunner.Commands()...),
		}, nil
	}

	// Push with force-with-lease to the remote.
	_, pushErr := wtRunner.Run(ctx, "push", "--force-with-lease", remote, wtBranch+":"+shortBranch)
	if pushErr != nil {
		return models.Result{
			Message:    fmt.Sprintf("resolved conflicts but push failed: %v", pushErr),
			CommandLog: append(gitRunner.Commands(), wtRunner.Commands()...),
		}, nil
	}

	return models.Result{
		OK:         true,
		Message:    fmt.Sprintf("resolved conflicts on %s/%s and pushed", remote, shortBranch),
		CommandLog: append(gitRunner.Commands(), wtRunner.Commands()...),
	}, nil
}

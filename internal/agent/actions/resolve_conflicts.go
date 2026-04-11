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
		prompt := prompts.BuildResolveConflictsPrompt("<worktree>", branchName, targetBranch, conflictInfo)

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
	gitRunner := gitbackend.NewRunner(info.Path)
	gitRunner.StartLogging()

	_, shortBranch, ok := strings.Cut(branchName, "/")
	if !ok {
		shortBranch = branchName
	}

	// --- Worktree 1: strategized prompt ---
	wt1Path, wt1Branch, err := a.createWorktree(ctx, gitRunner, branchName, shortBranch, "strategized")
	if err != nil {
		return models.Result{Message: fmt.Sprintf("worktree 1 creation failed: %v", err)}, nil
	}

	wt1Runner := gitbackend.NewRunner(wt1Path)
	wt1Runner.StartLogging()

	hasConflicts1, err := a.mergeTarget(ctx, wt1Runner, targetBranch, "strategized")
	if err != nil {
		return models.Result{
			Message:    fmt.Sprintf("worktree 1 setup failed: %v", err),
			CommandLog: append(gitRunner.Commands(), wt1Runner.Commands()...),
		}, nil
	}

	if !hasConflicts1 {
		wt1Runner.AppendLog("# rebase succeeded cleanly — no conflicts to resolve")
	}

	prompt1 := prompts.BuildResolveConflictsPrompt(wt1Path, shortBranch, targetBranch, conflicts)

	agentRunner.WorkDir = wt1Path
	wt1Runner.AppendLog(agentRunner.CommandString(prompt1))

	_, agent1Err := agentRunner.Run(ctx, prompt1)
	if agent1Err != nil {
		wt1Runner.AppendLog(fmt.Sprintf("# agent FAILED: %v", agent1Err))
	}

	// --- Worktree 2: minimal prompt ---
	wt2Path, wt2Branch, err := a.createWorktree(ctx, gitRunner, branchName, shortBranch, "minimal")
	if err != nil {
		return models.Result{
			Message:    fmt.Sprintf("worktree 2 creation failed: %v", err),
			CommandLog: append(gitRunner.Commands(), wt1Runner.Commands()...),
		}, nil
	}

	wt2Runner := gitbackend.NewRunner(wt2Path)
	wt2Runner.StartLogging()

	hasConflicts2, err := a.mergeTarget(ctx, wt2Runner, targetBranch, "minimal")
	if err != nil {
		return models.Result{
			Message:    fmt.Sprintf("worktree 2 setup failed: %v", err),
			CommandLog: append(append(gitRunner.Commands(), wt1Runner.Commands()...), wt2Runner.Commands()...),
		}, nil
	}

	if !hasConflicts2 {
		wt2Runner.AppendLog("# rebase succeeded cleanly — no conflicts to resolve")
	}

	prompt2 := fmt.Sprintf(
		"Resolve all merge conflicts in this git repository at %s.\n"+
			"Branch %q is being rebased onto %q.\n"+
			"Fix all conflict markers (<<<<<<< ======= >>>>>>>) in the working tree.\n"+
			"Stage resolved files with `git add -A`. Do NOT commit or push.",
		wt2Path, shortBranch, targetBranch,
	)

	agentRunner.WorkDir = wt2Path
	wt2Runner.AppendLog(agentRunner.CommandString(prompt2))

	_, agent2Err := agentRunner.Run(ctx, prompt2)
	if agent2Err != nil {
		wt2Runner.AppendLog(fmt.Sprintf("# agent FAILED: %v", agent2Err))
	}

	// --- Report: no push, no cleanup ---
	cmdLog := make([]string, 0, len(gitRunner.Commands())+len(wt1Runner.Commands())+len(wt2Runner.Commands()))
	cmdLog = append(cmdLog, gitRunner.Commands()...)
	cmdLog = append(cmdLog, wt1Runner.Commands()...)
	cmdLog = append(cmdLog, wt2Runner.Commands()...)

	msg := fmt.Sprintf(
		"A/B prompt test done (no push, no cleanup).\n"+
			"  Worktree 1 (strategized): %s [branch: %s]\n"+
			"  Worktree 2 (minimal):     %s [branch: %s]",
		wt1Path, wt1Branch, wt2Path, wt2Branch,
	)

	return models.Result{
		OK:         true,
		Message:    msg,
		CommandLog: cmdLog,
	}, nil
}

// createWorktree creates a named worktree from the given branch.
// Returns the worktree path and local branch name.
func (a ResolveConflicts) createWorktree(
	ctx context.Context,
	gitRunner *gitbackend.Runner,
	branchName, shortBranch, label string,
) (string, string, error) {
	tmpDir, err := os.MkdirTemp("", "janitor-agent-"+label+"-*")
	if err != nil {
		return "", "", fmt.Errorf("cannot create temp directory: %w", err)
	}

	wtBranch := fmt.Sprintf("janitor-agent-%s-%s", label, shortBranch)
	wtPath := filepath.Join(tmpDir, wtBranch)

	_, err = gitRunner.Run(ctx, "worktree", "add", "-b", wtBranch, wtPath, branchName)
	if err != nil {
		os.RemoveAll(tmpDir) //nolint:errcheck // best-effort
		return "", "", err
	}

	return wtPath, wtBranch, nil
}

// mergeTarget merges the target branch into the worktree branch.
// This produces conflict markers in the working tree for the agent to resolve.
// Returns (true, nil) if the merge produces conflicts (expected).
// Returns (false, nil) if the merge succeeds cleanly (unexpected).
func (a ResolveConflicts) mergeTarget(
	ctx context.Context,
	wtRunner *gitbackend.Runner,
	targetBranch, label string,
) (bool, error) {
	_, mergeErr := wtRunner.Run(ctx, "merge", "--no-commit", targetBranch)
	hasConflicts := mergeErr != nil

	if hasConflicts {
		// Log conflicting files for diagnostics.
		diffOut, _ := wtRunner.Run(ctx, "diff", "--name-only", "--diff-filter=U")
		wtRunner.AppendLog(fmt.Sprintf("# [%s] merge conflicts: %s", label, strings.TrimSpace(diffOut)))
	}

	return hasConflicts, nil
}

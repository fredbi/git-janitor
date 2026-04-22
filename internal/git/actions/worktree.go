// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// DeleteWorktree removes a linked worktree. Subject = worktree path.
//
// Uses `git worktree remove --force`: if the worktree is dirty, the user
// is already warned via the popup (Dirty flag) and chose to proceed.
type DeleteWorktree struct {
	gitAction
}

func NewDeleteWorktree() DeleteWorktree {
	return DeleteWorktree{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"delete-worktree",
				"remove a linked worktree (destructive)",
			),
		},
	}
}

func (DeleteWorktree) ApplyTo() models.SubjectKind { return models.SubjectWorktree }
func (DeleteWorktree) Destructive() bool           { return true }

func (a DeleteWorktree) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a DeleteWorktree) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, subjects []string) (models.Result, error) {
	if len(subjects) == 0 {
		return models.Result{Message: "delete-worktree: no worktree path supplied"}, nil
	}

	return r.RemoveWorktree(ctx, subjects[0]).ToResult(), nil
}

// RepairWorktree fixes the admin-file links for a linked worktree that
// has been moved on disk. Subject is unused; the new path is provided via
// the param prompt.
type RepairWorktree struct {
	gitAction
}

func NewRepairWorktree() RepairWorktree {
	return RepairWorktree{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"repair-worktree",
				"re-link admin files after a worktree was moved on disk",
			),
		},
	}
}

func (RepairWorktree) ApplyTo() models.SubjectKind { return models.SubjectWorktree }
func (RepairWorktree) ParamPrompt() string         { return "New worktree path:" }

func (a RepairWorktree) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, params)
}

func (a RepairWorktree) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, params []string) (models.Result, error) {
	// With ParamPrompt, the UX injects the typed value as the first param
	// of the first subject. Per-subject dispatch places subject name at
	// params[0] and user-entered param at params[1]. Accept either shape.
	var newPath string

	switch len(params) {
	case 0:
		return models.Result{Message: "repair-worktree: no new path supplied"}, nil
	case 1:
		newPath = params[0]
	default:
		newPath = params[1]
	}

	return r.RepairWorktree(ctx, newPath).ToResult(), nil
}

// PruneWorktrees removes admin files for linked worktrees whose working
// copy has been deleted on disk. Repo-level action.
type PruneWorktrees struct {
	gitAction
}

func NewPruneWorktrees() PruneWorktrees {
	return PruneWorktrees{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"prune-worktrees",
				"remove admin files for linked worktrees whose working copy is gone",
			),
		},
	}
}

func (PruneWorktrees) ApplyTo() models.SubjectKind { return models.SubjectRepo }
func (PruneWorktrees) Destructive() bool           { return true }

func (a PruneWorktrees) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a PruneWorktrees) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, _ []string) (models.Result, error) {
	return r.PruneWorktrees(ctx).ToResult(), nil
}

// AddWorktree creates a new linked worktree at the given path. The TUI
// supplies the path inline via the status-bar prompt, so the subject
// slot carries the target path.
type AddWorktree struct {
	gitAction
}

func NewAddWorktree() AddWorktree {
	return AddWorktree{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"add-worktree",
				"create a new linked worktree",
			),
		},
	}
}

func (AddWorktree) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a AddWorktree) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a AddWorktree) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, subjects []string) (models.Result, error) {
	if len(subjects) == 0 {
		return models.Result{Message: "add-worktree: no path supplied"}, nil
	}

	return r.AddWorktree(ctx, subjects[0]).ToResult(), nil
}

// MoveWorktree relocates an existing linked worktree to a new path.
// Per-subject dispatch places the current path at params[0] and the
// user-entered new path at params[1].
type MoveWorktree struct {
	gitAction
}

func NewMoveWorktree() MoveWorktree {
	return MoveWorktree{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"move-worktree",
				"move a linked worktree to a new filesystem location",
			),
		},
	}
}

func (MoveWorktree) ApplyTo() models.SubjectKind { return models.SubjectWorktree }
func (MoveWorktree) ParamPrompt() string         { return "New worktree path:" }

func (a MoveWorktree) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, params)
}

func (a MoveWorktree) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, params []string) (models.Result, error) {
	if len(params) < 2 { //nolint:mnd // subject + new-path
		return models.Result{Message: "move-worktree: missing source or destination path"}, nil
	}

	return r.MoveWorktree(ctx, params[0], params[1]).ToResult(), nil
}

// LockWorktree locks a linked worktree so it cannot be pruned. Subject
// is the worktree path; optional reason is provided via the param slot.
type LockWorktree struct {
	gitAction
}

func NewLockWorktree() LockWorktree {
	return LockWorktree{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"lock-worktree",
				"lock a linked worktree (optional reason)",
			),
		},
	}
}

func (LockWorktree) ApplyTo() models.SubjectKind { return models.SubjectWorktree }

// ParamPrompt returns the status-bar prompt for the optional lock reason.
// The TUI may also capture the reason inline and pass it directly via
// ActionSubject.Params without invoking this prompt.
func (LockWorktree) ParamPrompt() string { return "Lock reason (optional):" }

func (a LockWorktree) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, params)
}

func (a LockWorktree) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, params []string) (models.Result, error) {
	// Per-subject dispatch (the only path the popup uses): params[0] is
	// the worktree path, params[1] is the optional reason.
	if len(params) == 0 {
		return models.Result{Message: "lock-worktree: no worktree path supplied"}, nil
	}

	var reason string
	if len(params) > 1 {
		reason = params[1]
	}

	return r.LockWorktree(ctx, params[0], reason).ToResult(), nil
}

// UnlockWorktree removes the lock on a linked worktree so it can later be
// pruned or moved. Marked destructive because unlocking is typically the
// prelude to pruning the admin files.
type UnlockWorktree struct {
	gitAction
}

func NewUnlockWorktree() UnlockWorktree {
	return UnlockWorktree{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"unlock-worktree",
				"remove the lock on a linked worktree (destructive: enables pruning)",
			),
		},
	}
}

func (UnlockWorktree) ApplyTo() models.SubjectKind { return models.SubjectWorktree }
func (UnlockWorktree) Destructive() bool           { return true }

func (a UnlockWorktree) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a UnlockWorktree) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, subjects []string) (models.Result, error) {
	if len(subjects) == 0 {
		return models.Result{Message: "unlock-worktree: no worktree path supplied"}, nil
	}

	return r.UnlockWorktree(ctx, subjects[0]).ToResult(), nil
}

// StashWorktreeDirty stashes the uncommitted work sitting in a specific
// linked worktree, not the main repo. Subject is the worktree's path.
type StashWorktreeDirty struct {
	gitAction
}

func NewStashWorktreeDirty() StashWorktreeDirty {
	return StashWorktreeDirty{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"stash-worktree-dirty",
				"stash uncommitted work inside a linked worktree",
			),
		},
	}
}

func (StashWorktreeDirty) ApplyTo() models.SubjectKind { return models.SubjectWorktree }
func (StashWorktreeDirty) ParamPrompt() string         { return "Stash message (optional):" }

func (a StashWorktreeDirty) Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error) {
	// Discard the main-repo runner from context — we need a scratch runner
	// pointing at the worktree path.
	return a.execute(ctx, info, params)
}

func (a StashWorktreeDirty) execute(ctx context.Context, _ *models.RepoInfo, params []string) (models.Result, error) {
	if len(params) == 0 {
		return models.Result{Message: "stash-worktree-dirty: no worktree path supplied"}, nil
	}

	worktreePath := params[0]

	message := ""
	if len(params) > 1 {
		message = params[1]
	}

	wt := backend.NewRunner(worktreePath)

	return wt.StashDirty(ctx, message).ToResult(), nil
}

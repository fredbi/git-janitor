package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// DeleteBranch deletes local branches using force delete (for squash-merged branches).
type DeleteBranch struct {
	gitAction
}

func NewDeleteBranch() DeleteBranch {
	return DeleteBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"delete-branch",
				"delete a branch",
			),
		},
	}
}

func (DeleteBranch) ApplyTo() models.SubjectKind { return models.SubjectBranch }
func (DeleteBranch) Destructive() bool           { return true }

func (a DeleteBranch) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a DeleteBranch) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, subjects []string) (models.Result, error) {
	for _, name := range subjects {
		result := r.DeleteBranch(ctx, name, info.DefaultBranch)
		if !result.OK {
			return result.ToResult(), nil
		}
	}

	return models.Result{OK: true, Message: "all branches deleted"}, nil
}

// UpdateBranch fast-forwards local branches from their upstream.
type UpdateBranch struct {
	gitAction
}

func NewUpdateBranch() UpdateBranch {
	return UpdateBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"update-branch",
				"fetch a branch to update from remote",
			),
		},
	}
}

func (UpdateBranch) ApplyTo() models.SubjectKind { return models.SubjectBranch }

func (a UpdateBranch) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a UpdateBranch) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, subjects []string) (models.Result, error) {
	if info == nil {
		return models.Result{}, errors.New("repo info is required to update branches")
	}

	branchByName := indexBranches(info.Branches)

	for _, name := range subjects {
		branch, ok := branchByName[name]
		if !ok {
			continue
		}

		result := r.UpdateBranch(ctx, branch)
		if !result.OK {
			return models.Result{OK: false, Message: result.Message}, nil
		}
	}

	return models.Result{OK: true, Message: "all branches updated"}, nil
}

// RebaseBranch rebases local branches onto the default branch.
type RebaseBranch struct {
	gitAction
}

func NewRebaseBranch() RebaseBranch {
	return RebaseBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"rebase-branch",
				"rebase a local branch onto another one",
			),
		},
	}
}

func (RebaseBranch) ApplyTo() models.SubjectKind { return models.SubjectBranch }

func (a RebaseBranch) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a RebaseBranch) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, subjects []string) (models.Result, error) {
	if info == nil {
		return models.Result{}, errors.New("repo info is required to rebase branches")
	}

	branchByName := indexBranches(info.Branches)

	for _, name := range subjects {
		branch, ok := branchByName[name]
		if !ok {
			continue
		}

		result := r.RebaseBranch(ctx, info.DefaultBranch, branch)
		if !result.OK {
			return models.Result{OK: false, Message: result.Message}, nil
		}
	}

	return models.Result{OK: true, Message: "all branches rebased"}, nil
}

// PushBranch pushes local branches and sets upstream tracking.
// Pushes to upstream if it exists, otherwise to origin.
type PushBranch struct {
	gitAction
}

func NewPushBranch() PushBranch {
	return PushBranch{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"push-branch",
				"push a local branch to remote",
			),
		},
	}
}

func (PushBranch) ApplyTo() models.SubjectKind { return models.SubjectBranch }

func (a PushBranch) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a PushBranch) execute(ctx context.Context, r *backend.Runner, info *models.RepoInfo, subjects []string) (models.Result, error) {
	remote := models.RemoteOrigin
	if info != nil {
		remote = models.DefaultPushRemote(info.Remotes)
	}

	for _, name := range subjects {
		result := r.PushBranch(ctx, remote, name)
		if !result.OK {
			return models.Result{OK: false, Message: result.Message}, nil
		}
	}

	return models.Result{OK: true, Message: "all branches pushed to " + remote}, nil
}

// indexBranches builds a name-to-Branch lookup map from a slice.
func indexBranches(branches []models.Branch) map[string]models.Branch {
	m := make(map[string]models.Branch, len(branches))
	for _, b := range branches {
		m[b.Name] = b
	}

	return m
}

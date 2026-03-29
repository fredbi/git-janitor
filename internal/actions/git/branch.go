package gitactions

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// ActionDeleteBranch deletes local branches using force delete (for squash-merged branches).
type ActionDeleteBranch struct {
	engine.GitAction
}

func (ActionDeleteBranch) ApplyTo() engine.SubjectKind { return engine.SubjectBranch }
func (ActionDeleteBranch) Destructive() bool           { return true }

func (a ActionDeleteBranch) Execute(ctx context.Context, r *git.Runner, _ *git.RepoInfo, subjects []string) (engine.Result, error) {
	for _, name := range subjects {
		result := r.DeleteBranch(ctx, name)
		if !result.OK {
			return engine.Result{OK: false, Message: result.Message}, nil
		}
	}

	return engine.Result{OK: true, Message: "all branches deleted"}, nil
}

// ActionUpdateBranch fast-forwards local branches from their upstream.
type ActionUpdateBranch struct {
	engine.GitAction
}

func (ActionUpdateBranch) ApplyTo() engine.SubjectKind { return engine.SubjectBranch }

func (a ActionUpdateBranch) Execute(ctx context.Context, r *git.Runner, info *git.RepoInfo, subjects []string) (engine.Result, error) {
	if info == nil {
		return engine.Result{}, errors.New("repo info is required to update branches")
	}

	branchByName := indexBranches(info.Branches)

	for _, name := range subjects {
		branch, ok := branchByName[name]
		if !ok {
			continue
		}

		result := r.UpdateBranch(ctx, branch)
		if !result.OK {
			return engine.Result{OK: false, Message: result.Message}, nil
		}
	}

	return engine.Result{OK: true, Message: "all branches updated"}, nil
}

// ActionRebaseBranch rebases local branches onto the default branch.
type ActionRebaseBranch struct {
	engine.GitAction
}

func (ActionRebaseBranch) ApplyTo() engine.SubjectKind { return engine.SubjectBranch }

func (a ActionRebaseBranch) Execute(ctx context.Context, r *git.Runner, info *git.RepoInfo, subjects []string) (engine.Result, error) {
	if info == nil {
		return engine.Result{}, errors.New("repo info is required to rebase branches")
	}

	branchByName := indexBranches(info.Branches)

	for _, name := range subjects {
		branch, ok := branchByName[name]
		if !ok {
			continue
		}

		result := r.RebaseBranch(ctx, info.DefaultBranch, branch)
		if !result.OK {
			return engine.Result{OK: false, Message: result.Message}, nil
		}
	}

	return engine.Result{OK: true, Message: "all branches rebased"}, nil
}

// ActionPushBranch pushes local branches and sets upstream tracking.
// Pushes to upstream if it exists, otherwise to origin.
type ActionPushBranch struct {
	engine.GitAction
}

func (ActionPushBranch) ApplyTo() engine.SubjectKind { return engine.SubjectBranch }

func (a ActionPushBranch) Execute(ctx context.Context, r *git.Runner, info *git.RepoInfo, subjects []string) (engine.Result, error) {
	remote := git.RemoteOrigin
	if info != nil {
		remote = git.DefaultPushRemote(info.Remotes)
	}

	for _, name := range subjects {
		result := r.PushBranch(ctx, remote, name)
		if !result.OK {
			return engine.Result{OK: false, Message: result.Message}, nil
		}
	}

	return engine.Result{OK: true, Message: fmt.Sprintf("all branches pushed to %s", remote)}, nil
}

// indexBranches builds a name-to-Branch lookup map from a slice.
func indexBranches(branches []git.Branch) map[string]git.Branch {
	m := make(map[string]git.Branch, len(branches))
	for _, b := range branches {
		m[b.Name] = b
	}

	return m
}

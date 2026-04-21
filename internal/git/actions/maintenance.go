package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// RunGC runs git gc (standard garbage collection).
type RunGC struct {
	gitAction
}

func NewRunGC() RunGC {
	return RunGC{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"run-gc",
				"run standard git garbage collection",
			),
		},
	}
}

func (RunGC) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a RunGC) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a RunGC) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, _ []string) (models.Result, error) {
	result := r.Compact(ctx)

	return result.ToResult(), nil
}

// RunGCAggressive runs git gc --aggressive (deep repack).
type RunGCAggressive struct {
	gitAction
}

func NewRunGCAggressive() RunGCAggressive {
	return RunGCAggressive{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"run-gc-aggressive",
				"run git garbage collection with --aggressive",
			),
		},
	}
}

func (RunGCAggressive) ApplyTo() models.SubjectKind { return models.SubjectRepo }
func (RunGCAggressive) Destructive() bool           { return true }

func (a RunGCAggressive) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a RunGCAggressive) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, _ []string) (models.Result, error) {
	result := r.CompactAggressive(ctx)

	return result.ToResult(), nil
}

// RunGCDeepClean expires reflogs and runs git gc --prune=now --aggressive.
//
// This is destructive and irreversible: any unreachable object is dropped,
// including those that a standard gc would preserve (reflog-referenced or
// within gc.pruneExpire). Recovery via reflog is not possible afterwards.
type RunGCDeepClean struct {
	gitAction
}

func NewRunGCDeepClean() RunGCDeepClean {
	return RunGCDeepClean{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"run-gc-deep-clean",
				"expire reflogs and run git gc --prune=now --aggressive (destructive: loses reflog recovery)",
			),
		},
	}
}

func (RunGCDeepClean) ApplyTo() models.SubjectKind { return models.SubjectRepo }
func (RunGCDeepClean) Destructive() bool           { return true }

func (a RunGCDeepClean) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a RunGCDeepClean) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, _ []string) (models.Result, error) {
	result := r.CompactDeepClean(ctx)

	return result.ToResult(), nil
}

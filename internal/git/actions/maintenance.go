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

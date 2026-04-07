package actions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// PushTag pushes local tags to the origin remote.
type PushTag struct {
	gitAction
}

func NewPushTag() PushTag {
	return PushTag{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"push-tag",
				"pushes a tag to remote",
			),
		},
	}
}

func (PushTag) ApplyTo() models.SubjectKind { return models.SubjectTag }

func (a PushTag) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a PushTag) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, subjects []string) (models.Result, error) {
	for _, name := range subjects {
		result := r.PushTag(ctx, name)
		if !result.OK {
			return models.Result{OK: false, Message: result.Message}, nil
		}
	}

	return models.Result{OK: true, Message: "all tags pushed"}, nil
}

// FetchTags fetches all tags from all remotes.
type FetchTags struct {
	gitAction
}

func NewFetchTags() FetchTags {
	return FetchTags{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"fetch-tags",
				"fetches all tags from remote, with --force",
			),
		},
	}
}

func (FetchTags) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a FetchTags) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a FetchTags) execute(ctx context.Context, r *backend.Runner, _ *models.RepoInfo, _ []string) (models.Result, error) {
	err := r.FetchAllTags(ctx)
	if err != nil {
		return models.Result{OK: false, Message: err.Error()}, nil //nolint:nilerr // error is handled and wrapped in result
	}

	return models.Result{OK: true, Message: "fetched all tags from all remotes"}, nil
}

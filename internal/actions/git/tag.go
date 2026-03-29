package gitactions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// ActionPushTag pushes local tags to the origin remote.
type ActionPushTag struct {
	engine.GitAction
}

func (ActionPushTag) ApplyTo() engine.SubjectKind { return engine.SubjectTag }

func (a ActionPushTag) Execute(ctx context.Context, r *git.Runner, _ *git.RepoInfo, subjects []string) (engine.Result, error) {
	for _, name := range subjects {
		result := r.PushTag(ctx, name)
		if !result.OK {
			return engine.Result{OK: false, Message: result.Message}, nil
		}
	}

	return engine.Result{OK: true, Message: "all tags pushed"}, nil
}

// ActionFetchTags fetches all tags from all remotes.
type ActionFetchTags struct {
	engine.GitAction
}

func (ActionFetchTags) ApplyTo() engine.SubjectKind { return engine.SubjectRepo }

func (a ActionFetchTags) Execute(ctx context.Context, r *git.Runner, _ *git.RepoInfo, _ []string) (engine.Result, error) {
	err := r.FetchAllTags(ctx)
	if err != nil {
		return engine.Result{OK: false, Message: err.Error()}, nil
	}

	return engine.Result{OK: true, Message: "fetched all tags from all remotes"}, nil
}

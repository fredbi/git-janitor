package gitactions

import (
	"context"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// ActionRunGC runs git gc (standard garbage collection).
type ActionRunGC struct {
	engine.GitAction
}

func (ActionRunGC) ApplyTo() engine.SubjectKind { return engine.SubjectRepo }

func (a ActionRunGC) Execute(ctx context.Context, r *git.Runner, _ *git.RepoInfo, _ []string) (engine.Result, error) {
	result := r.Compact(ctx)

	return engine.Result{OK: result.OK, Message: result.Message}, nil
}

// ActionRunGCAggressive runs git gc --aggressive (deep repack).
type ActionRunGCAggressive struct {
	engine.GitAction
}

func (ActionRunGCAggressive) ApplyTo() engine.SubjectKind { return engine.SubjectRepo }
func (ActionRunGCAggressive) Destructive() bool           { return true }

func (a ActionRunGCAggressive) Execute(ctx context.Context, r *git.Runner, _ *git.RepoInfo, _ []string) (engine.Result, error) {
	result := r.CompactAggressive(ctx)

	return engine.Result{OK: result.OK, Message: result.Message}, nil
}

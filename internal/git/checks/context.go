package checks

import (
	"context"
	"fmt"

	"github.com/fredbi/git-janitor/internal/git/backend"
)

type gitContextKey uint8

const (
	repoInfoKey gitContextKey = iota + 1
	// runnerKey // reserved if some checks need an additional git command via runner.
)

func repoInfoCtx(ctx context.Context) (*backend.RepoInfo, error) {
	raw := ctx.Value(repoInfoKey)

	info, ok := raw.(*backend.RepoInfo)
	if !ok {
		return nil, fmt.Errorf("internal error: expected git *backend.RepoInfo but got: %T", raw)
	}

	return info, nil
}

/*
func runnerCtx(ctx context.Context) (*backend.Runner, error) {
	raw := ctx.Value(runnerKey)

	runner, ok := raw.(*backend.Runner)
	if !ok {
		return nil, fmt.Errorf("internal error: expected git *backend.Runner but got: %T", raw)
	}

	return runner, nil
}
*/

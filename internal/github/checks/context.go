package checks

import (
	"context"
	"fmt"

	"github.com/fredbi/git-janitor/internal/github/backend"
)

type githubContextKey uint8

const (
	repoInfoKey githubContextKey = iota + 1
	// runnerKey // reserved if some check requires an additional github API interaction.
)

func repoInfoCtx(ctx context.Context) (*backend.RepoInfo, error) {
	raw := ctx.Value(repoInfoKey)

	info, ok := raw.(*backend.RepoInfo)
	if !ok {
		return nil, fmt.Errorf("internal error: expected github *backend.RepoInfo but got: %T", raw)
	}

	return info, nil
}

/*
func runnerCtx(ctx context.Context) (*backend.Runner, error) {
	raw := ctx.Value(runnerKey)

	runner, ok := raw.(*backend.Runner)
	if !ok {
		return nil, fmt.Errorf("internal error: expected github *backend.Runner but got: %T", raw)
	}

	return runner, nil
}
*/

// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/fredbi/git-janitor/internal/gitlab/backend"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

func runnerCtx(ctx context.Context) (*backend.Runner, error) {
	r, ok := ifaces.RunnerFromContext(ctx)
	if !ok || r == nil {
		return nil, errors.New("internal error: no runner in context")
	}

	runner, ok := r.(*backend.Runner)
	if !ok {
		return nil, fmt.Errorf("internal error: expected gitlab *backend.Runner but got: %T", r)
	}

	return runner, nil
}

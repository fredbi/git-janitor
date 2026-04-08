// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"

	agentbackend "github.com/fredbi/git-janitor/internal/agent/backend"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Action = agentAction{}

// agentAction is the base struct for all agent actions.
type agentAction struct {
	models.Describer
}

func (agentAction) IsAction()                   {}
func (agentAction) Kind() models.ActionKind     { return models.ActionKindAgent }
func (agentAction) Destructive() bool           { return true } // agent actions always need confirmation
func (agentAction) ApplyTo() models.SubjectKind { return models.SubjectNone }
func (agentAction) ParamPrompt() string         { return "" }

func (agentAction) Execute(_ context.Context, _ *models.RepoInfo, _ []string) (models.Result, error) {
	return models.Result{}, errors.New("not implemented")
}

// runnerCtx extracts the agent runner from context.
func runnerCtx(ctx context.Context) (*agentbackend.Runner, error) {
	r, ok := ifaces.RunnerFromContext(ctx)
	if !ok || r == nil {
		return nil, errors.New("internal error: no agent runner in context")
	}

	runner, ok := r.(*agentbackend.Runner)
	if !ok {
		return nil, errors.New("internal error: expected agent *backend.Runner")
	}

	return runner, nil
}

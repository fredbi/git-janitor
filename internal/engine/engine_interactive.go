package engine

import (
	"context"
	"iter"

	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Engineer = &Interactive{}

// Interactive engine is an [iface.Engineer] that runs checks and actions
// following user interactions.
type Interactive struct {
	options
}

func NewInteractive(opts ...Option) *Interactive {
	return &Interactive{
		options: optionsWithDefaults(opts),
	}
}

func (r *Interactive) WithRunner(parent context.Context, name, repo string) context.Context {
	return nil
}
func (r *Interactive) WithRepoInfo(parent context.Context, name string, info ifaces.RepoInfo) context.Context {
	return nil
}
func (r *Interactive) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	return nil, nil
}
func (r *Interactive) RunnerEnabledFor(name, repo string) bool {
	return false
}
func (r *Interactive) GetCheck(name string) (ifaces.Check, bool) {
	return nil, false
}
func (r *Interactive) GetAction(name string) (ifaces.Action, bool) {
	return nil, false
}
func (r *Interactive) Collect(ctx context.Context, opts ...models.CollectOption) ifaces.RepoInfo {
	return nil
}
func (r *Interactive) Refresh(ctx context.Context) ifaces.RepoInfo {
	return nil
}
func (r *Interactive) Reload(cfg *config.Config) {
}
func (r *Interactive) Execute(ctx context.Context, action models.ActionSuggestion) (models.Result, error) {
	return models.Result{}, nil
}

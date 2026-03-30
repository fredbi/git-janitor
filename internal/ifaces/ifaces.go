package ifaces

import (
	"context"
	"iter"

	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/models"
)

// Engineer describes how engines operate.
//
// The [Engineer] is the high-level interface to interact with multiple registered providers.
type Engineer interface {
	WithRunner(parent context.Context, name, repo string) context.Context
	WithRepoInfo(parent context.Context, repo string, info RepoInfo) context.Context

	// Evaluate all configured checks for the runner and repo info in the current context
	Evaluate(ctx context.Context) (iter.Seq[models.Alert], error)

	// Execute an action for the runner and repo in the current context
	Execute(ctx context.Context, action models.ActionSuggestion) (models.Result, error)

	// Tells if a runner is enabled for this repo
	RunnerEnabledFor(name, repo string) bool

	GetCheck(name string) (Check, bool)
	GetAction(name string) (Action, bool)

	// Collectors
	Collect(ctx context.Context, opts ...models.CollectOption) RepoInfo
	Refresh(ctx context.Context) RepoInfo

	// Reload config and sets checks configured for individual roots & repos
	Reload(cfg *config.Config)
}

type RepoInfo interface {
	IsRepoInfo()
}

// RunnerFactory produces new runner for a given repo (with path or URL).
//
// Each provider exposes such a factory with a unique name.
type RunnerFactory interface {
	NewRunner(dir string) Runner
	Name() string
}

// Runner knows how to get things done.
type Runner interface {
	// Run(ctx context.Context, args ...string) (models.Result, error)
	Run(ctx context.Context, args ...string) (string, error) // TODO
}

// SelfDescribed is common to checks and actions: provides a name and
// human-readable description for the registry and config wizard.
type SelfDescribed interface {
	Name() string
	Description() string
}

// Check is the interface for all checks, regardless of provider.
type Check interface {
	SelfDescribed
	Kind() models.CheckKind
	Evaluate(ctx context.Context) (iter.Seq[models.Alert], error)
}

// Action is the interface for all actions, regardless of provider.
type Action interface {
	SelfDescribed
	Kind() models.ActionKind
	ApplyTo() models.SubjectKind // what kind of subject this action operates on
	Destructive() bool           // needs user confirmation
	Execute(ctx context.Context, params []string) (models.Result, error)
}

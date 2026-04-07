package ifaces

import (
	"context"
	"iter"
	"time"

	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/models"
)

// Engineer describes how engines operate.
//
// The [Engineer] is the high-level interface to interact with multiple registered providers.
//
// The engine owns the runner factories, configuration, and registries —
// callers never need to set up runner context or check provider availability.
type Engineer interface {
	// Evaluate all configured checks for the given repo info.
	//
	// By default, all enabled checks for the input repo are evaluated.
	Evaluate(ctx context.Context, info *models.RepoInfo, opts ...models.EvaluateOption) (iter.Seq[models.Alert], error)

	// Execute an action. The engine creates the appropriate runner
	// based on the action's Kind and the repo info.
	Execute(ctx context.Context, info *models.RepoInfo, action models.ActionSuggestion) (models.Result, error)

	GetCheck(name string) (Check, bool)
	GetAction(name string) (Action, bool)

	// Collect gathers or enriches repo info.
	// The info parameter must be non-nil and must have Path set.
	// When the info only has Path populated, a fresh collection is performed.
	// When the info already has data, the engine can enrich it (e.g. add platform data)
	// and skip work that's already done.
	//
	// Use [models.CollectPlatform] to request hosting-platform metadata.
	// The engine resolves the origin URL and checks config/token availability internally.
	Collect(ctx context.Context, info *models.RepoInfo, opts ...models.CollectOption) *models.RepoInfo

	// Refresh fetches from remotes then re-collects repo info.
	// The info parameter must be non-nil and must have Path set.
	Refresh(ctx context.Context, info *models.RepoInfo) *models.RepoInfo

	// RecentHistory returns action history entries for the given repo
	// with timestamps after the since cutoff, newest first.
	RecentHistory(repoPath string, since time.Time) []models.HistoryEntry

	// Reload config and sets checks configured for individual roots & repos.
	Reload(cfg *config.Config)
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
	Evaluate(ctx context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error)
}

// Action is the interface for all actions, regardless of provider.
type Action interface {
	SelfDescribed
	Kind() models.ActionKind
	ApplyTo() models.SubjectKind // what kind of subject this action operates on
	Destructive() bool           // needs user confirmation
	Execute(ctx context.Context, info *models.RepoInfo, params []string) (models.Result, error)
}

// runnerContextKey is a shared context key type for runner injection.
// The engine sets it; actions read it.
type runnerContextKey struct{}

// WithRunner stores a runner in the context.
// The runner is stored as any — actions type-assert to their concrete runner type.
func WithRunner(ctx context.Context, runner any) context.Context {
	return context.WithValue(ctx, runnerContextKey{}, runner)
}

// RunnerFromContext extracts a runner from the context.
func RunnerFromContext(ctx context.Context) (any, bool) {
	r := ctx.Value(runnerContextKey{})

	return r, r != nil
}

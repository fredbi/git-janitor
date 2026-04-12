package engine

import (
	"time"

	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/quickactions"
	"github.com/fredbi/git-janitor/internal/registry"
	"github.com/fredbi/git-janitor/internal/store"
)

type Option func(*options)

const defaultCacheTTL = 5 * time.Minute

type options struct {
	cfg          *config.Config
	checks       *registry.Registry[ifaces.Check]
	actions      *registry.Registry[ifaces.Action]
	store        store.Store
	cacheTTL     time.Duration
	quickActions *registry.Registry[*quickactions.QuickAction]
}

func WithConfig(cfg *config.Config) Option {
	return func(o *options) {
		o.cfg = cfg
	}
}

func WithChecks(checks *registry.Registry[ifaces.Check]) Option {
	return func(o *options) {
		o.checks = checks
	}
}

func WithActions(actions *registry.Registry[ifaces.Action]) Option {
	return func(o *options) {
		o.actions = actions
	}
}

// WithStore sets the persistent key-value store used for caching.
// When nil, caching is disabled (same behavior as without this option).
func WithStore(s store.Store) Option {
	return func(o *options) {
		o.store = s
	}
}

// WithCacheTTL sets the time-to-live for cached RepoInfo entries.
// Default is 5 minutes.
func WithCacheTTL(d time.Duration) Option {
	return func(o *options) {
		o.cacheTTL = d
	}
}

func optionsWithDefaults(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	if o.cacheTTL == 0 {
		o.cacheTTL = defaultCacheTTL
	}

	if o.quickActions == nil {
		// Build from cfg now so the engine is usable straight after construction.
		// Build errors are silently ignored here — they will resurface on Reload.
		reg, _ := quickactions.BuildRegistry(o.cfg)
		o.quickActions = reg
	}

	return o
}

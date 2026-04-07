package engine

import (
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/registry"
)

type Option func(*options)

type options struct {
	cfg     *config.Config
	checks  *registry.Registry[ifaces.Check]
	actions *registry.Registry[ifaces.Action]
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

func optionsWithDefaults(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

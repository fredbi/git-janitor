package ux

import (
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/registry"
	"github.com/fredbi/git-janitor/internal/ux/themes"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

type Option func(*options)

type options struct {
	Cfg     *config.Config
	Engine  ifaces.Engineer
	Theme   uxtypes.Theme
	themes  *registry.Registry[uxtypes.Theme]
	checks  *registry.Registry[ifaces.Check]
	actions *registry.Registry[ifaces.Action]
}

func WithConfig(cfg *config.Config) Option {
	return func(o *options) {
		if cfg != nil {
			o.Cfg = cfg
		}
	}
}

func WithEngine(e ifaces.Engineer) Option {
	return func(o *options) {
		o.Engine = e
	}
}

func WithThemes(themes *registry.Registry[uxtypes.Theme]) Option {
	return func(o *options) {
		o.themes = themes
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

func WithDefaultTheme(theme uxtypes.Theme) Option {
	return func(o *options) {
		o.Theme = theme
	}
}

func applyOptionsWithDefaults(opts []Option) options {
	o := options{
		Cfg:    &config.Config{},
		Theme:  themes.Default(),
		Engine: engine.NewInteractive(),
	}

	for _, apply := range opts {
		apply(&o)
	}

	return o
}

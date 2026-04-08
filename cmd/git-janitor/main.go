package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
	"github.com/fredbi/git-janitor/internal/github"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/registry"
	"github.com/fredbi/git-janitor/internal/store"
	"github.com/fredbi/git-janitor/internal/store/bolt"
	"github.com/fredbi/git-janitor/internal/ux"
	"github.com/fredbi/git-janitor/internal/ux/themes"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

const app = "git-janitor"

func main() {
	// loads a config file and merge with defaults
	cfg, err := config.LoadDefault()
	if err != nil {
		fatal(err)

		return
	}

	// registers all supported color themes
	themes := registry.New[uxtypes.Theme](
		registry.With(themes.AllThemes()),
	)
	defaultTheme, ok := themes.Get("default")
	if !ok {
		fatal(errors.New("no default color theme defined"))

		return
	}

	// Open persistent store for caching and history.
	// A lock error means another instance is running — exit immediately.
	kvStore, err := bolt.OpenDefault()
	if err != nil {
		if errors.Is(err, bolt.ErrLocked) {
			fatal(err)

			return
		}

		fmt.Fprintf(os.Stderr, "%s: store disabled: %v\n", app, err)
	}

	if kvStore != nil {
		defer func() {
			_ = kvStore.Close()
		}()

		// Clear the RepoInfo cache on startup. The cache is a performance
		// optimization; clearing it ensures no stale data from older binary
		// versions persists (new fields would deserialize as zero values).
		// History is preserved.
		_ = kvStore.ClearBucket(store.BucketCache)
	}

	// registers all supported checks
	checks := registry.New[ifaces.Check](
		registry.With(git.AllChecks(), github.AllChecks()),
	)

	// registers all supported actions
	actions := registry.New[ifaces.Action](
		registry.With(git.AllActions(), github.AllActions()),
	)

	// injects all dependencies into the UI model
	model := ux.New(
		ux.WithConfig(cfg),
		ux.WithThemes(themes),
		ux.WithDefaultTheme(defaultTheme),
		ux.WithChecks(checks),
		ux.WithActions(actions),
		ux.WithEngine(
			engine.NewInteractive(
				engine.WithConfig(cfg),
				engine.WithChecks(checks),
				engine.WithActions(actions),
				engine.WithStore(kvStore),
			),
		),
	)

	// build the TUI model
	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// runs the UI
	if _, err := p.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", app, err)
	os.Exit(1)
}

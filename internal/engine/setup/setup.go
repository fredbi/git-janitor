// Package setup creates a fully wired engine with all built-in checks
// and actions registered. It exists as a separate package to break the
// import cycle between engine and the check/action implementations.
package setup

import (
	gitactions "github.com/fredbi/git-janitor/internal/actions/git"
	gitchecks "github.com/fredbi/git-janitor/internal/checks/git"
	"github.com/fredbi/git-janitor/internal/engine"
)

// NewEngine creates an Engine with all built-in checks and actions registered.
func NewEngine() *engine.Engine {
	e := engine.New()

	gitchecks.RegisterAll(e.Checks)
	gitactions.RegisterAll(e.Actions)

	return e
}

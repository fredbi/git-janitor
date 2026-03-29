// Package setup creates a fully wired engine with all built-in checks
// and actions registered. It exists as a separate package to break the
// import cycle between engine and the check/action implementations.
package setup

import (
	gitactions "github.com/fredbi/git-janitor/internal/actions/git"
	githubactions "github.com/fredbi/git-janitor/internal/actions/github"
	gitchecks "github.com/fredbi/git-janitor/internal/checks/git"
	githubchecks "github.com/fredbi/git-janitor/internal/checks/github"
	"github.com/fredbi/git-janitor/internal/engine"
)

// NewEngine creates an Engine with all built-in checks and actions registered.
func NewEngine() *engine.Engine {
	e := engine.New()

	gitchecks.RegisterAll(e.Checks)
	githubchecks.RegisterAll(e.Checks)
	gitactions.RegisterAll(e.Actions)
	githubactions.RegisterAll(e.Actions)

	return e
}

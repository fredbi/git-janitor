package github

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

func RunnerFactory() iter.Seq[ifaces.RunnerFactory] {
	return slices.Values([]ifaces.RunnerFactory{
		backend.NewRunnerFactory(),
	})
}

package git

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

func RunnerFactory() iter.Seq[ifaces.RunnerFactory] {
	return slices.Values([]ifaces.RunnerFactory{
		backend.NewRunnerFactory(),
	})
}

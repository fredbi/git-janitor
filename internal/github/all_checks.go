// Package github provides built-in github checks.
package github

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/github/checks"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

// AllChecks yields all built-in github checks.
func AllChecks() iter.Seq[ifaces.Check] {
	return slices.Values([]ifaces.Check{
		checks.NewDefaultBranchMismatch(),
		checks.NewRepoArchived(),
		checks.NewDescriptionMissing(),
		checks.NewVisibilityPrivate(),
		checks.NewRepoForkParent(),
		checks.NewSecurityNotEnabled(),
		checks.NewSecurityAlerts(),
	})
}

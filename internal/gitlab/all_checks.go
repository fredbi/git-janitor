// Package gitlab provides built-in GitLab checks.
package gitlab

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/gitlab/checks"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

// AllChecks yields all built-in GitLab checks.
func AllChecks() iter.Seq[ifaces.Check] {
	return slices.Values([]ifaces.Check{
		checks.NewDefaultBranchMismatch(),
		checks.NewRepoArchived(),
		checks.NewDescriptionMissing(),
		checks.NewVisibilityPrivate(),
		checks.NewRepoForkParent(),
		checks.NewBranchProtectionMissing(),
		checks.NewDeleteBranchOnMergeMissing(),
	})
}

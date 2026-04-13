// SPDX-License-Identifier: Apache-2.0

// Package gitlab provides built-in GitLab actions that execute
// via the GitLab API in response to alert suggestions.
package gitlab

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/gitlab/actions"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

// AllActions yields all built-in GitLab actions.
func AllActions() iter.Seq[ifaces.Action] {
	return slices.Values([]ifaces.Action{
		actions.NewSetProjectDescription(),
		actions.NewOpenInBrowser(),
		actions.NewEnableBranchProtection(),
		actions.NewEnableDeleteBranchOnMerge(),
	})
}

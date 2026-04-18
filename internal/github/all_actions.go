// SPDX-License-Identifier: Apache-2.0

// Package githubactions provides built-in GitHub actions that execute
// via the GitHub API in response to alert suggestions.
package github

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/github/actions"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

// AllActions yields all built-in GitHub actions.
func AllActions() iter.Seq[ifaces.Action] {
	return slices.Values([]ifaces.Action{
		actions.NewSetRepoDescription(),
		actions.NewOpenInBrowser(),
		actions.NewEnableBranchProtection(),
		actions.NewEnableDeleteBranchOnMerge(),
		actions.NewDisableForkActions(),
		actions.NewSelfAssignIssue(),
		actions.NewCloseIssue(),
	})
}

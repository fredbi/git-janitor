// SPDX-License-Identifier: Apache-2.0

// Package agent provides the AI agent backend for git-janitor.
package agent

import (
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/agent/actions"
	"github.com/fredbi/git-janitor/internal/ifaces"
)

// AllActions yields all built-in agent actions.
func AllActions() iter.Seq[ifaces.Action] {
	return slices.Values([]ifaces.Action{
		actions.NewResolveConflicts(),
	})
}

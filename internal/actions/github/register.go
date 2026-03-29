// SPDX-License-Identifier: Apache-2.0

// Package githubactions provides built-in GitHub actions that execute
// via the GitHub API in response to alert suggestions.
package githubactions

import "github.com/fredbi/git-janitor/internal/engine"

// RegisterAll registers all built-in GitHub actions into the given registry.
func RegisterAll(r *engine.ActionRegistry) {
	r.Register(ActionSetRepoDescription{GitHubAction: engine.GitHubAction{
		Describer: engine.Describer{
			CheckName:        "set-repo-description",
			CheckDescription: "set the repository description on GitHub",
		},
	}})
}

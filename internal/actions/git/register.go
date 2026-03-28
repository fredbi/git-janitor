// Package gitactions provides built-in git actions that execute
// git.Runner methods in response to alert suggestions.
package gitactions

import "github.com/fredbi/git-janitor/internal/engine"

// RegisterAll registers all built-in git actions into the given registry.
func RegisterAll(r *engine.ActionRegistry) {
	r.Register(ActionRunGC{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "run-gc",
			CheckDescription: "run git gc to reclaim space and optimize the repository",
		},
	}})

	r.Register(ActionRunGCAggressive{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "run-gc-aggressive",
			CheckDescription: "run git gc --aggressive for deeper optimization with high compression",
		},
	}})
}

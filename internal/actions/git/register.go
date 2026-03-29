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

	r.Register(ActionDeleteBranch{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "delete-branch",
			CheckDescription: "delete a local branch (force delete, for squash-merged branches)",
		},
	}})

	r.Register(ActionUpdateBranch{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "update-branch",
			CheckDescription: "fast-forward a local branch from its upstream",
		},
	}})

	r.Register(ActionRebaseBranch{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "rebase-branch",
			CheckDescription: "rebase a local branch onto the default branch",
		},
	}})

	r.Register(ActionPushBranch{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "push-branch",
			CheckDescription: "push a local branch to origin and set upstream tracking",
		},
	}})

	r.Register(ActionPushTag{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "push-tag",
			CheckDescription: "push a local tag to the origin remote",
		},
	}})

	r.Register(ActionRenameRemote{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "rename-remote",
			CheckDescription: "rename a git remote (e.g. fix misspelled upstream)",
		},
	}})

	r.Register(ActionFetchTags{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "fetch-tags",
			CheckDescription: "fetch all tags from all remotes",
		},
	}})

	r.Register(ActionDeleteLocalClone{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "delete-local-clone",
			CheckDescription: "permanently delete the local clone directory",
		},
	}})

	r.Register(ActionOpenInBrowser{GitAction: engine.GitAction{
		Describer: engine.Describer{
			CheckName:        "open-in-browser",
			CheckDescription: "open a URL in the default browser",
		},
	}})
}

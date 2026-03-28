// Package gitchecks provides built-in git checks that evaluate
// git.RepoInfo and produce alerts for the engine.
package gitchecks

import "github.com/fredbi/git-janitor/internal/engine"

// RegisterAll registers all built-in git checks into the given registry.
func RegisterAll(r *engine.CheckRegistry) {
	r.Register(CheckHealthGCAdvised{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "health-gc-advised",
			CheckDescription: "detects when git gc would reclaim space or improve performance",
		},
	}})

	r.Register(CheckSizeRepackAdvised{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "size-repack-advised",
			CheckDescription: "detects when git repack would reduce repository bloat",
		},
	}})

	r.Register(CheckHealthFSCK{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "health-fsck-errors",
			CheckDescription: "detects repository corruption found by git fsck",
		},
	}})

	r.Register(CheckBranchLagging{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "branch-lagging",
			CheckDescription: "detects local branches that are behind their remote tracking branch",
		},
	}})

	r.Register(CheckBranchMergedNotDeleted{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "branch-merged-not-deleted",
			CheckDescription: "detects local branches already merged into the default branch",
		},
	}})

	r.Register(CheckBranchGoneUpstream{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "branch-gone-upstream",
			CheckDescription: "detects local branches whose upstream has been deleted from the remote",
		},
	}})

	r.Register(CheckBranchNoUpstream{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "branch-no-upstream",
			CheckDescription: "detects local branches not tracking any remote branch",
		},
	}})

	r.Register(CheckBranchDiverged{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "branch-diverged",
			CheckDescription: "detects local branches that have diverged from their upstream (both ahead and behind)",
		},
	}})
}

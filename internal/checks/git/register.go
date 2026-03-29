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

	r.Register(CheckBranchNotMergeable{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "branch-not-mergeable",
			CheckDescription: "detects local branches that cannot be merged or rebased onto the default branch",
		},
	}})

	r.Register(CheckRemoteNoOrigin{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "remote-no-origin",
			CheckDescription: "detects repos with no remote named origin",
		},
	}})

	r.Register(CheckRemoteMisnamedUpstream{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "remote-misnamed-upstream",
			CheckDescription: "detects fork repos where the upstream remote has an incorrect name",
		},
	}})

	r.Register(CheckTagsLocalOnly{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "tags-local-only",
			CheckDescription: "detects tags that exist locally but not on the remote",
		},
	}})

	r.Register(CheckTagsRemoteOnly{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "tags-remote-only",
			CheckDescription: "detects tags that exist on the remote but not locally",
		},
	}})

	r.Register(CheckDirtyWorktree{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "dirty-worktree",
			CheckDescription: "detects uncommitted changes in the working tree",
		},
	}})

	r.Register(CheckActivityStale{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "activity-stale",
			CheckDescription: "detects repositories with reduced commit activity (30d/90d/360d inactivity)",
		},
	}})

	r.Register(CheckConfigNoEmail{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "config-no-email",
			CheckDescription: "detects repositories with no user.email configured",
		},
	}})

	r.Register(CheckConfigUnsigned{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "config-unsigned",
			CheckDescription: "detects repositories where commit signing is not enabled",
		},
	}})

	r.Register(CheckLargeFiles{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "filestats-large-files",
			CheckDescription: "detects files in HEAD exceeding the size threshold",
		},
	}})

	r.Register(CheckBinaryFiles{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "filestats-binary",
			CheckDescription: "detects binary files tracked in HEAD",
		},
	}})

	r.Register(CheckShallow{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "traits-shallow",
			CheckDescription: "detects shallow clones (incomplete history)",
		},
	}})

	r.Register(CheckSubmodules{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "traits-submodules",
			CheckDescription: "detects repositories using git submodules",
		},
	}})

	r.Register(CheckLFS{GitCheck: engine.GitCheck{
		Describer: engine.Describer{
			CheckName:        "traits-lfs",
			CheckDescription: "detects repositories using Git LFS",
		},
	}})
}

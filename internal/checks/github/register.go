// SPDX-License-Identifier: Apache-2.0

// Package githubchecks provides built-in GitHub checks that evaluate
// github.RepoData and produce alerts for the engine.
package githubchecks

import "github.com/fredbi/git-janitor/internal/engine"

// RegisterAll registers all built-in GitHub checks into the given registry.
func RegisterAll(r *engine.CheckRegistry) {
	r.Register(CheckRepoArchived{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{
			CheckName:        "github-repo-archived",
			CheckDescription: "detects repositories archived on GitHub (read-only)",
		},
	}})

	r.Register(CheckDefaultBranchMismatch{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{
			CheckName:        "github-default-branch-mismatch",
			CheckDescription: "detects when GitHub default branch differs from local",
		},
	}})

	r.Register(CheckDescriptionMissing{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{
			CheckName:        "github-description-missing",
			CheckDescription: "detects repositories with no description on GitHub",
		},
	}})

	r.Register(CheckVisibilityPrivate{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{
			CheckName:        "github-visibility-private",
			CheckDescription: "reports when a repository is private (informational)",
		},
	}})

	r.Register(CheckRepoForkParent{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{
			CheckName:        "github-repo-fork-parent",
			CheckDescription: "identifies fork parent repository (informational)",
		},
	}})

	r.Register(CheckSecurityNotEnabled{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{
			CheckName:        "github-security-not-enabled",
			CheckDescription: "detects when no security scanners are accessible",
		},
	}})

	r.Register(CheckSecurityAlerts{GitHubCheck: engine.GitHubCheck{
		Describer: engine.Describer{
			CheckName:        "github-security-alerts",
			CheckDescription: "detects open security alerts (Dependabot, code scanning, secret scanning)",
		},
	}})
}

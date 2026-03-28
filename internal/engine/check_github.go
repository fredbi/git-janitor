package engine

import (
	"errors"
	"iter"
)

// GitHubRepoData is a placeholder for future GitHub-specific repository data.
// It will be replaced by a concrete type from internal/github when that
// package is implemented.
type GitHubRepoData struct{}

// IsRepoInfo satisfies the RepoInfo marker interface.
func (GitHubRepoData) IsRepoInfo() {}

// GitHubCheck is the base struct for all GitHub checks.
// Concrete GitHub checks embed this and override Evaluate.
type GitHubCheck struct {
	Describer
}

func (GitHubCheck) isCheck()        {}
func (GitHubCheck) Kind() CheckKind { return CheckKindGitHub }

// Evaluate is the default implementation that returns "not implemented".
// Concrete checks override this method.
func (GitHubCheck) Evaluate(_ *GitHubRepoData) (iter.Seq[Alert], error) {
	return nil, errors.New("not implemented")
}

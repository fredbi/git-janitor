// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"errors"
	"iter"

	"github.com/fredbi/git-janitor/internal/github"
)

// GitHubCheck is the base struct for all GitHub checks.
// Concrete GitHub checks embed this and override Evaluate.
type GitHubCheck struct {
	Describer
}

func (GitHubCheck) isCheck()        {}
func (GitHubCheck) Kind() CheckKind { return CheckKindGitHub }

// Evaluate is the default implementation that returns "not implemented".
// Concrete checks override this method.
func (GitHubCheck) Evaluate(_ *github.RepoData) (iter.Seq[Alert], error) {
	return nil, errors.New("not implemented")
}

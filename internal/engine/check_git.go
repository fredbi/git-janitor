package engine

import (
	"errors"
	"iter"

	"github.com/fredbi/git-janitor/internal/git"
)

// GitCheck is the base struct for all git checks.
// Concrete git checks embed this and override Evaluate.
type GitCheck struct {
	Describer
}

func (GitCheck) isCheck()        {}
func (GitCheck) Kind() CheckKind { return CheckKindGit }

// Evaluate is the default implementation that returns "not implemented".
// Concrete checks override this method.
func (GitCheck) Evaluate(_ *git.RepoInfo) (iter.Seq[Alert], error) {
	return nil, errors.New("not implemented")
}

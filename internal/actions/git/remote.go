package gitactions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// ActionRenameRemote renames a git remote.
// Subjects: [oldName, newName].
type ActionRenameRemote struct {
	engine.GitAction
}

func (ActionRenameRemote) ApplyTo() engine.SubjectKind { return engine.SubjectRemote }

func (a ActionRenameRemote) Execute(ctx context.Context, r *git.Runner, _ *git.RepoInfo, subjects []string) (engine.Result, error) {
	if len(subjects) < 2 {
		return engine.Result{}, errors.New("rename-remote requires [oldName, newName] as subjects")
	}

	result := r.RenameRemote(ctx, subjects[0], subjects[1])

	return engine.Result{OK: result.OK, Message: result.Message}, nil
}

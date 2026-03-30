package actions

import (
	"context"
	"errors"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// RenameRemote renames a git remote.
// Subjects: [oldName, newName].
type RenameRemote struct {
	gitAction
}

func NewRenameRemote() RenameRemote {
	return RenameRemote{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"rename-remote",
				"rename a remote declared locally",
			),
		},
	}
}

func (RenameRemote) ApplyTo() models.SubjectKind { return models.SubjectRemote }

func (a RenameRemote) Execute(ctx context.Context, subjects []string) (models.Result, error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a RenameRemote) execute(ctx context.Context, r *backend.Runner, _ *backend.RepoInfo, subjects []string) (models.Result, error) {
	const minSubjects = 2
	if len(subjects) < minSubjects {
		return models.Result{}, errors.New("rename-remote requires [oldName, newName] as subjects")
	}

	result := r.RenameRemote(ctx, subjects[0], subjects[1])

	return models.Result{OK: result.OK, Message: result.Message}, nil
}

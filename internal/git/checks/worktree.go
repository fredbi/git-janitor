package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// DirtyWorktree detects uncommitted changes in the working tree.
type DirtyWorktree struct {
	gitCheck
}

func NewDirtyWorktree() DirtyWorktree {
	return DirtyWorktree{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"dirty-worktree",
				"detects uncommitted changes in the working tree",
			),
		},
	}
}

// Evaluate inspects the Status from RepoInfo.
func (c DirtyWorktree) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c DirtyWorktree) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.Status.IsDirty() {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	var staged, unstaged, untracked int

	for _, e := range info.Status.Entries {
		switch {
		case e.IsUntracked():
			untracked++
		case e.XY[0] != '.':
			// First character is the index (staged) status; '.' means unchanged.
			staged++
		default:
			// First character is '.', second is the worktree (unstaged) status.
			unstaged++
		}
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityHigh,
		Summary:   "working tree has uncommitted changes",
		Detail:    fmt.Sprintf("%d staged, %d unstaged, %d untracked", staged, unstaged, untracked),
	}), nil
}

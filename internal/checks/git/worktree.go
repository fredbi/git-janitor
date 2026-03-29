package gitchecks

import (
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckDirtyWorktree detects uncommitted changes in the working tree.
type CheckDirtyWorktree struct {
	engine.GitCheck
}

// Evaluate inspects the Status from RepoInfo.
func (c CheckDirtyWorktree) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if !info.Status.IsDirty() {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
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

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityHigh,
		Summary:   "working tree has uncommitted changes",
		Detail:    fmt.Sprintf("%d staged, %d unstaged, %d untracked", staged, unstaged, untracked),
	}), nil
}

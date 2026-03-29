// SPDX-License-Identifier: Apache-2.0

package gitactions

import (
	"context"
	"fmt"
	"os"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// ActionDeleteLocalClone removes the local clone directory.
// This is destructive and irreversible.
type ActionDeleteLocalClone struct {
	engine.GitAction
}

func (ActionDeleteLocalClone) Destructive() bool    { return true }
func (ActionDeleteLocalClone) ApplyTo() engine.SubjectKind { return engine.SubjectRepo }

func (a ActionDeleteLocalClone) Execute(
	_ context.Context,
	_ *git.Runner,
	info *git.RepoInfo,
	_ []string,
) (engine.Result, error) {
	if info == nil {
		return engine.Result{}, fmt.Errorf("repo info is required to delete local clone")
	}

	path := info.Path
	if path == "" {
		return engine.Result{}, fmt.Errorf("empty repo path, refusing to delete")
	}

	if err := os.RemoveAll(path); err != nil {
		return engine.Result{
			OK:      false,
			Message: fmt.Sprintf("failed to delete %s: %v", path, err),
		}, err
	}

	return engine.Result{
		OK:      true,
		Message: fmt.Sprintf("deleted local clone: %s", path),
	}, nil
}

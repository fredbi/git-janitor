// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// CleanStaleSubmoduleDirs removes every orphan directory under .git/modules/
// whose submodule name is not referenced by .git/config.
//
// The action is destructive: submodule objects are deleted irreversibly.
// It operates on the snapshot captured in RepoInfo.StaleSubmoduleDirs —
// if the snapshot is stale, the action will skip any path that no longer
// exists or that no longer qualifies as an orphan module directory.
type CleanStaleSubmoduleDirs struct {
	gitAction
}

func NewCleanStaleSubmoduleDirs() CleanStaleSubmoduleDirs {
	return CleanStaleSubmoduleDirs{
		gitAction: gitAction{
			Describer: models.NewDescriber(
				"clean-stale-submodule-dirs",
				"remove orphan .git/modules/* directories (destructive)",
			),
		},
	}
}

func (CleanStaleSubmoduleDirs) ApplyTo() models.SubjectKind { return models.SubjectRepo }
func (CleanStaleSubmoduleDirs) Destructive() bool           { return true }

func (a CleanStaleSubmoduleDirs) Execute(ctx context.Context, info *models.RepoInfo, subjects []string) (models.Result, error) {
	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, info, subjects)
}

func (a CleanStaleSubmoduleDirs) execute(_ context.Context, r *backend.Runner, info *models.RepoInfo, _ []string) (models.Result, error) {
	if len(info.StaleSubmoduleDirs) == 0 {
		return models.Result{OK: true, Message: "no stale submodule directories"}, nil
	}

	repoRoot, err := filepath.Abs(r.Dir)
	if err != nil {
		return models.Result{}, fmt.Errorf("resolve repo root: %w", err)
	}

	var (
		reclaimed int64
		removed   []string
		skipped   []string
	)

	for _, s := range info.StaleSubmoduleDirs {
		// Safety: never remove anything outside the repo's .git tree.
		if !isInsideRepoGitDir(repoRoot, s.Path) {
			skipped = append(skipped, s.Name+" (outside repo)")

			continue
		}

		if rmErr := os.RemoveAll(s.Path); rmErr != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%v)", s.Name, rmErr))

			continue
		}

		reclaimed += s.SizeBytes

		removed = append(removed, s.Name)
	}

	msg := fmt.Sprintf("removed %d orphan submodule dir(s), reclaimed %s",
		len(removed), models.FormatBytes(reclaimed))
	if len(skipped) > 0 {
		msg += "; skipped: " + strings.Join(skipped, ", ")
	}

	return models.Result{OK: len(removed) > 0 || len(skipped) == 0, Message: msg}, nil
}

// isInsideRepoGitDir returns true when target is beneath <repoRoot>/.git.
// Guards against absolute-path drift in the snapshot.
func isInsideRepoGitDir(repoRoot, target string) bool {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}

	gitRoot := filepath.Join(repoRoot, ".git") + string(filepath.Separator)

	return strings.HasPrefix(absTarget+string(filepath.Separator), gitRoot)
}

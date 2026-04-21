package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// Shallow detects shallow clones (incomplete history).
type Shallow struct {
	gitCheck
}

func NewShallow() Shallow {
	return Shallow{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"traits-shallow",
				"detects shallow clones (incomplete history)",
			),
		},
	}
}

// Evaluate inspects the IsShallow field from RepoInfo.
func (c Shallow) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c Shallow) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.IsShallow {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "repository is a shallow clone",
	}), nil
}

// Submodules detects repositories using git submodules.
type Submodules struct {
	gitCheck
}

func NewSubmodules() Submodules {
	return Submodules{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"traits-submodules",
				"detects repositories using git submodules",
			),
		},
	}
}

// Evaluate inspects the HasSubmodules field from RepoInfo.
func (c Submodules) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c Submodules) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.HasSubmodules {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "repository uses git submodules",
	}), nil
}

// StaleSubmodules detects orphaned directories under .git/modules/
// whose submodule name is no longer referenced by .git/config.
// These are residue from removed or renamed submodules and hold space
// that a standard git gc cannot reclaim.
type StaleSubmodules struct {
	gitCheck
}

func NewStaleSubmodules() StaleSubmodules {
	return StaleSubmodules{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"stale-submodule-dirs",
				"detects orphaned .git/modules/* directories from removed submodules",
			),
		},
	}
}

// Evaluate inspects the StaleSubmoduleDirs field from RepoInfo.
func (c StaleSubmodules) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c StaleSubmodules) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if len(info.StaleSubmoduleDirs) == 0 {
		return noAlert(c.Name())
	}

	var total int64

	const maxReported = 10

	limit := min(len(info.StaleSubmoduleDirs), maxReported)
	lines := make([]string, 0, limit)

	for _, s := range info.StaleSubmoduleDirs {
		total += s.SizeBytes
	}

	for _, s := range info.StaleSubmoduleDirs[:limit] {
		lines = append(lines, fmt.Sprintf("%s (%s)", s.Name, models.FormatBytes(s.SizeBytes)))
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary: fmt.Sprintf("%d orphan .git/modules/* dir(s) using %s",
			len(info.StaleSubmoduleDirs), models.FormatBytes(total)),
		Detail: strings.Join(lines, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "clean-stale-submodule-dirs",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

// LFS detects repositories using Git LFS.
type LFS struct {
	gitCheck
}

func NewLFS() LFS {
	return LFS{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"traits-lfs",
				"detects repositories using Git LFS",
			),
		},
	}
}

// Evaluate inspects the HasLFS field from RepoInfo.
func (c LFS) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c LFS) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if !info.HasLFS {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "repository uses Git LFS",
	}), nil
}

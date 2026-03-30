package checks

import (
	"context"
	"iter"

	"github.com/fredbi/git-janitor/internal/git/backend"
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
func (c Shallow) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c Shallow) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
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
func (c Submodules) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c Submodules) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
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
func (c LFS) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c LFS) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
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

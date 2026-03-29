package gitchecks

import (
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckShallow detects shallow clones (incomplete history).
type CheckShallow struct {
	engine.GitCheck
}

// Evaluate inspects the IsShallow field from RepoInfo.
func (c CheckShallow) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if !info.IsShallow {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   "repository is a shallow clone",
	}), nil
}

// CheckSubmodules detects repositories using git submodules.
type CheckSubmodules struct {
	engine.GitCheck
}

// Evaluate inspects the HasSubmodules field from RepoInfo.
func (c CheckSubmodules) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if !info.HasSubmodules {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   "repository uses git submodules",
	}), nil
}

// CheckLFS detects repositories using Git LFS.
type CheckLFS struct {
	engine.GitCheck
}

// Evaluate inspects the HasLFS field from RepoInfo.
func (c CheckLFS) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if !info.HasLFS {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   "repository uses Git LFS",
	}), nil
}

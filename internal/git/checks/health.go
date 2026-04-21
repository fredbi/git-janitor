package checks

import (
	"context"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// HealthGCAdvised detects when git gc would be beneficial,
// based on loose objects, prune-packable duplicates, pack count, and garbage.
type HealthGCAdvised struct {
	gitCheck
}

func NewHealthGCAdvised() HealthGCAdvised {
	return HealthGCAdvised{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"health-gc-advised",
				"detects when git gc would reclaim space or improve performance",
			),
		},
	}
}

// Evaluate inspects the HealthReport from RepoInfo.
func (c HealthGCAdvised) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c HealthGCAdvised) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Health == nil || !info.Health.GCAdvised {
		return noAlert(c.Name())
	}

	// Same 10 MiB floor as size-repack-advised: on tiny repos the loose-object
	// / prune-packable / pack-count heuristics fire on noise that a user
	// won't meaningfully benefit from cleaning up.
	if info.Size != nil && info.Size.GitDirBytes < gcAdviceMinGitDirBytes {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   "garbage collection recommended",
		Detail:    strings.Join(info.Health.GCReasons, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "run-gc",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

// gcAdviceMinGitDirBytes silences gc/repack/bloat advisories on repos
// small enough that structural overhead dominates and any heuristic is
// noise. Shared by health-gc-advised, size-repack-advised, and
// size-unreachable-bloat.
const gcAdviceMinGitDirBytes = 10 * 1024 * 1024 // 10 MiB

// SizeRepackAdvised detects when git repack would be beneficial,
// based on pack count, loose/packed ratio, .git size, and bloat ratio.
type SizeRepackAdvised struct {
	gitCheck
}

func NewSizeRepackAdvised() SizeRepackAdvised {
	return SizeRepackAdvised{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"size-repack-advised",
				"detects when git repack would reduce repository bloat",
			),
		},
	}
}

// Evaluate inspects the RepoSize from RepoInfo.
func (c SizeRepackAdvised) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c SizeRepackAdvised) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Size == nil || !info.Size.RepackAdvised {
		return noAlert(c.Name())
	}

	if info.Size.GitDirBytes < gcAdviceMinGitDirBytes {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   "repository repack recommended",
		Detail:    strings.Join(info.Size.RepackReasons, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "run-gc-aggressive",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

// UnreachableBloat detects repositories where the .git directory is
// significantly larger than reachable objects — a signal that unreachable
// objects (held alive by reflogs or the default gc grace period) dominate
// the pack. Aggressive gc will not reclaim this space; a deep clean
// (reflog expiry + gc --prune=now) is required.
type UnreachableBloat struct {
	gitCheck
}

func NewUnreachableBloat() UnreachableBloat {
	return UnreachableBloat{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"size-unreachable-bloat",
				"detects .git bloat from unreachable objects that a standard gc cannot reclaim",
			),
		},
	}
}

// Evaluate inspects the RepoSize from RepoInfo.
func (c UnreachableBloat) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c UnreachableBloat) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Size == nil || !info.Size.UnreachableBloat {
		return noAlert(c.Name())
	}

	if info.Size.GitDirBytes < gcAdviceMinGitDirBytes {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   "unreachable objects dominate .git — deep clean required",
		Detail:    strings.Join(info.Size.UnreachableBloatReasons, "; "),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "run-gc-deep-clean",
			SubjectKind: models.SubjectRepo,
			Subjects:    simpleSubject(info.Path),
		}},
	}), nil
}

// HealthFSCK detects repository corruption found by git fsck.
type HealthFSCK struct {
	gitCheck
}

func NewHealthFSCK() HealthFSCK {
	return HealthFSCK{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"health-fsck-errors",
				"detects repository corruption found by git fsck",
			),
		},
	}
}

// Evaluate inspects the HealthReport for fsck errors.
func (c HealthFSCK) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c HealthFSCK) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Health == nil || len(info.Health.FSCKErrors) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityCritical,
		Summary:   "repository integrity issues detected",
		Detail:    strings.Join(info.Health.FSCKErrors, "\n"),
	}), nil
}

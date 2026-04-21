package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

const (
	// repackPacksThreshold triggers repack advice when too many pack files exist.
	repackPacksThreshold = 20

	// repackLooseRatioThreshold triggers repack advice when loose object size
	// exceeds this fraction of the total packed size.
	repackLooseRatioThreshold = 0.2

	// repackMinPackSizeKB is the minimum packed size required for the
	// loose/packed ratio check to apply. Below this, the ratio is meaningless.
	repackMinPackSizeKB = 1024 // 1 MB

	// repackGitDirThreshold triggers repack advice when .git exceeds this size.
	repackGitDirBytes = 500 * 1024 * 1024 // 500 MB

	// bloatWasteRatio triggers the unreachable-bloat advisory when the
	// .git directory is significantly larger than the reachable objects
	// (bloat from unreachable objects held alive by reflogs or the default
	// gc grace period). A standard gc will not reclaim this space.
	bloatWasteRatio = 2.0

	// bloatMinGitDirBytes is the minimum .git size for the waste ratio
	// check to apply. Below this, structural overhead (hooks, logs, refs,
	// pack indexes) dominates and creates misleading ratios.
	bloatMinGitDirBytes = 5 * 1024 * 1024 // 5 MB

	// bloatMinWasteBytes is the minimum absolute waste (gitDir - reachable)
	// required to trigger the bloat advisory.
	bloatMinWasteBytes = 1024 * 1024 // 1 MB
)

// Size collects repository size metrics.
//
// It uses git rev-list --disk-usage --all (requires git >= 2.31) and
// a filesystem walk of the .git directory. Both are fast even on large repos.
func (r *Runner) Size(ctx context.Context) models.RepoSize {
	var s models.RepoSize

	s.GitDirBytes = gitDirSize(r.gitDir(ctx))
	s.ReachableBytes = r.reachableDiskUsage(ctx)

	r.evaluateRepackAdvice(ctx, &s)

	return s
}

// gitDir returns the absolute path to the .git directory using git rev-parse.
func (r *Runner) gitDir(ctx context.Context) string {
	out, err := r.run(ctx, cmdRevParseGitDir()...)
	if err != nil {
		return filepath.Join(r.Dir, ".git")
	}

	p := strings.TrimSpace(out)
	if filepath.IsAbs(p) {
		return p
	}

	return filepath.Join(r.Dir, p)
}

// gitDirSize returns the total size of a .git directory in bytes.
func gitDirSize(gitDir string) int64 {
	var total int64

	//nolint:errcheck // best-effort walk
	filepath.Walk(gitDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable files
		}

		if !info.IsDir() {
			total += info.Size()
		}

		return nil
	})

	return total
}

// reachableDiskUsage runs git rev-list --disk-usage --all.
func (r *Runner) reachableDiskUsage(ctx context.Context) int64 {
	out, err := r.run(ctx, cmdRevListDiskUsage()...)
	if err != nil {
		return 0
	}

	var bytes int64

	fmt.Sscanf(strings.TrimSpace(out), "%d", &bytes) //nolint:errcheck // best-effort

	return bytes
}

// evaluateRepackAdvice determines whether git repack would be beneficial.
// It reuses the health report data (already collected) when available via context,
// but also checks size-specific conditions.
func (r *Runner) evaluateRepackAdvice(ctx context.Context, s *models.RepoSize) {
	// Get count-objects data for pack/loose analysis.
	out, err := r.run(ctx, cmdCountObjects()...)
	if err != nil {
		return
	}

	var packs, looseSizeKB, packSizeKB int

	for line := range strings.SplitSeq(out, "\n") {
		key, val, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}

		var n int

		fmt.Sscanf(strings.TrimSpace(val), "%d", &n) //nolint:errcheck // best-effort

		switch key {
		case "packs":
			packs = n
		case "size":
			looseSizeKB = n
		case "size-pack":
			packSizeKB = n
		}
	}

	// Too many pack files.
	if packs >= repackPacksThreshold {
		s.RepackAdvised = true
		s.RepackReasons = append(s.RepackReasons,
			fmt.Sprintf("%d pack files (consolidation recommended)", packs))
	}

	// Loose objects are a significant fraction of packed size.
	// Only meaningful when packed size is substantial (> 1 MB).
	if packSizeKB >= repackMinPackSizeKB &&
		float64(looseSizeKB)/float64(packSizeKB) > repackLooseRatioThreshold {
		s.RepackAdvised = true
		s.RepackReasons = append(s.RepackReasons,
			fmt.Sprintf("loose objects (%d KB) are >%.0f%% of packed size (%d KB)",
				looseSizeKB, repackLooseRatioThreshold*100, packSizeKB))
	}

	// .git directory is very large.
	if s.GitDirBytes > repackGitDirBytes {
		s.RepackAdvised = true
		s.RepackReasons = append(s.RepackReasons,
			".git directory is "+models.FormatBytes(s.GitDirBytes))
	}

	// .git directory is much larger than reachable objects (bloat).
	// Skip for small repos (< 5 MB .git) where structural overhead dominates.
	// Skip if absolute waste is below 1 MB.
	//
	// This signal is categorised as unreachable-bloat rather than repack-advised
	// because a standard (even aggressive) gc preserves unreachable objects that
	// are within gc.pruneExpire or still referenced by the reflog — so repack
	// will not reclaim this space. Only a deep clean (reflog expiry +
	// gc --prune=now) resolves it.
	waste := s.GitDirBytes - s.ReachableBytes
	if s.GitDirBytes > bloatMinGitDirBytes &&
		s.ReachableBytes > 0 && waste > bloatMinWasteBytes &&
		float64(s.GitDirBytes)/float64(s.ReachableBytes) > bloatWasteRatio {
		s.UnreachableBloat = true
		s.UnreachableBloatReasons = append(s.UnreachableBloatReasons,
			fmt.Sprintf(".git (%s) is %.1fx larger than reachable objects (%s)",
				models.FormatBytes(s.GitDirBytes),
				float64(s.GitDirBytes)/float64(s.ReachableBytes),
				models.FormatBytes(s.ReachableBytes)))
	}
}

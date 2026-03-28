package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepoSize holds size metrics for a repository.
type RepoSize struct {
	// GitDirBytes is the total size of the .git directory on disk.
	GitDirBytes int64

	// ReachableBytes is the total size of all reachable objects
	// (git rev-list --disk-usage --all). This excludes unreachable
	// garbage but includes packed and loose objects.
	ReachableBytes int64

	// RepackAdvised is true when conditions suggest git repack would be beneficial.
	RepackAdvised bool

	// RepackReasons lists human-readable reasons why repack is advised.
	RepackReasons []string
}

const (
	// repackPacksThreshold triggers repack advice when too many pack files exist.
	repackPacksThreshold = 5

	// repackLooseRatioThreshold triggers repack advice when loose object size
	// exceeds this fraction of the total packed size.
	repackLooseRatioThreshold = 0.2

	// repackGitDirThreshold triggers repack advice when .git exceeds this size.
	repackGitDirBytes = 500 * 1024 * 1024 // 500 MB

	// repackWasteThreshold triggers repack advice when the .git directory is
	// significantly larger than the reachable objects (bloat from old packs,
	// reflogs, etc.).
	repackWasteRatio = 2.0
)

// Size collects repository size metrics.
//
// It uses git rev-list --disk-usage --all (requires git >= 2.31) and
// a filesystem walk of the .git directory. Both are fast even on large repos.
func (r *Runner) Size(ctx context.Context) RepoSize {
	var s RepoSize

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
func (r *Runner) evaluateRepackAdvice(ctx context.Context, s *RepoSize) {
	// Get count-objects data for pack/loose analysis.
	out, err := r.run(ctx, cmdCountObjects()...)
	if err != nil {
		return
	}

	var packs, looseSizeKB, packSizeKB int

	for _, line := range strings.Split(out, "\n") {
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
	if packSizeKB > 0 && float64(looseSizeKB)/float64(packSizeKB) > repackLooseRatioThreshold {
		s.RepackAdvised = true
		s.RepackReasons = append(s.RepackReasons,
			fmt.Sprintf("loose objects (%d KB) are >%.0f%% of packed size (%d KB)",
				looseSizeKB, repackLooseRatioThreshold*100, packSizeKB))
	}

	// .git directory is very large.
	if s.GitDirBytes > repackGitDirBytes {
		s.RepackAdvised = true
		s.RepackReasons = append(s.RepackReasons,
			fmt.Sprintf(".git directory is %s", formatBytes(s.GitDirBytes)))
	}

	// .git directory is much larger than reachable objects (bloat).
	if s.ReachableBytes > 0 && float64(s.GitDirBytes)/float64(s.ReachableBytes) > repackWasteRatio {
		s.RepackAdvised = true
		s.RepackReasons = append(s.RepackReasons,
			fmt.Sprintf(".git (%s) is %.1fx larger than reachable objects (%s)",
				formatBytes(s.GitDirBytes),
				float64(s.GitDirBytes)/float64(s.ReachableBytes),
				formatBytes(s.ReachableBytes)))
	}
}

// formatBytes returns a human-readable byte size.
func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

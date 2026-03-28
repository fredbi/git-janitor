package git

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
)

// HealthReport holds the result of a repository health check.
type HealthReport struct {
	// Integrity
	//
	// FSCKErrors lists corruption issues found by git fsck --connectivity-only.
	// Empty means the repository is structurally sound.
	FSCKErrors []string

	// OK is true when no integrity issues are found.
	OK bool

	// GC diagnostics
	//
	// LooseObjects is the number of unpacked loose objects.
	LooseObjects int

	// LooseSizeKB is the total size of loose objects in kilobytes.
	LooseSizeKB int

	// PackedObjects is the number of objects in pack files.
	PackedObjects int

	// Packs is the number of pack files.
	Packs int

	// PackSizeKB is the total size of all pack files in kilobytes.
	PackSizeKB int

	// PrunePackable is the number of loose objects also present in a pack
	// (wasted space, reclaimable by git gc).
	PrunePackable int

	// Garbage is the number of garbage files in the object store.
	Garbage int

	// GarbageSizeKB is the total size of garbage files in kilobytes.
	GarbageSizeKB int

	// GCAdvised is true when conditions suggest git gc would be beneficial.
	GCAdvised bool

	// GCReasons lists human-readable reasons why GC is advised.
	GCReasons []string
}

const (
	// defaultGCAutoThreshold is git's default threshold for auto-gc (loose objects).
	defaultGCAutoThreshold = 6700

	// gcLooseThreshold triggers a GC advisory when loose objects exceed this fraction
	// of the auto-gc threshold.
	gcLooseThresholdFraction = 0.5

	// gcPrunePackableThreshold triggers a GC advisory when this many objects
	// are both loose and packed (wasted duplicates).
	gcPrunePackableThreshold = 50

	// gcPacksThreshold triggers a GC advisory when there are too many pack files
	// (repack would consolidate them).
	gcPacksThreshold = 10

	// gcGarbageThreshold triggers a GC advisory when garbage files are present.
	gcGarbageThreshold = 1
)

// Health performs a repository health check covering integrity (fsck)
// and garbage collection diagnostics (count-objects).
//
// The fsck check uses --connectivity-only for speed (skips blob content verification).
// Dangling objects are not reported as errors (they are normal after rebase/amend).
func (r *Runner) Health(ctx context.Context) HealthReport {
	report := HealthReport{OK: true}

	// Integrity check.
	r.checkFSCK(ctx, &report)

	// GC diagnostics.
	r.checkCountObjects(ctx, &report)
	r.evaluateGCAdvice(&report)

	return report
}

// checkFSCK runs git fsck --connectivity-only and collects errors.
// Dangling objects and phantom objects are excluded (normal after rebase/amend).
func (r *Runner) checkFSCK(ctx context.Context, report *HealthReport) {
	// fsck writes issues to stderr, exits non-zero on problems.
	// Our run() captures stderr in the error message.
	_, err := r.run(ctx, cmdFSCK()...)
	if err == nil {
		return
	}

	// Parse the error output for real issues.
	errStr := err.Error()
	scanner := bufio.NewScanner(strings.NewReader(errStr))

	for scanner.Scan() {
		line := scanner.Text()

		// Skip known benign messages.
		if strings.Contains(line, "dangling") ||
			strings.Contains(line, "phantom") ||
			strings.Contains(line, "fantôme") { // French locale
			continue
		}

		// Skip the "git fsck: exit status" wrapper line.
		if strings.HasPrefix(line, "git fsck") {
			continue
		}

		line = strings.TrimSpace(line)
		if line != "" {
			report.FSCKErrors = append(report.FSCKErrors, line)
		}
	}

	if len(report.FSCKErrors) > 0 {
		report.OK = false
	}
}

// checkCountObjects runs git count-objects -v and parses the output.
func (r *Runner) checkCountObjects(ctx context.Context, report *HealthReport) {
	out, err := r.run(ctx, cmdCountObjects()...)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		key, val, ok := strings.Cut(scanner.Text(), ": ")
		if !ok {
			continue
		}

		n, _ := strconv.Atoi(strings.TrimSpace(val)) //nolint:errcheck // best-effort

		switch key {
		case "count":
			report.LooseObjects = n
		case "size":
			report.LooseSizeKB = n
		case "in-pack":
			report.PackedObjects = n
		case "packs":
			report.Packs = n
		case "size-pack":
			report.PackSizeKB = n
		case "prune-packable":
			report.PrunePackable = n
		case "garbage":
			report.Garbage = n
		case "size-garbage":
			report.GarbageSizeKB = n
		}
	}
}

// evaluateGCAdvice determines whether git gc would be beneficial.
func (r *Runner) evaluateGCAdvice(report *HealthReport) {
	threshold := defaultGCAutoThreshold

	// Check if the user has a custom gc.auto setting.
	if out, err := r.run(context.Background(), cmdConfigGet("gc.auto")...); err == nil {
		if n, parseErr := strconv.Atoi(strings.TrimSpace(out)); parseErr == nil && n > 0 {
			threshold = n
		}
	}

	// Too many loose objects.
	if report.LooseObjects > int(float64(threshold)*gcLooseThresholdFraction) {
		report.GCAdvised = true
		report.GCReasons = append(report.GCReasons,
			fmt.Sprintf("%d loose objects (threshold: %d)", report.LooseObjects, threshold))
	}

	// Loose objects that are already packed (wasted space).
	if report.PrunePackable >= gcPrunePackableThreshold {
		report.GCAdvised = true
		report.GCReasons = append(report.GCReasons,
			fmt.Sprintf("%d objects are both loose and packed (prune-packable)", report.PrunePackable))
	}

	// Too many pack files (repack would help).
	if report.Packs >= gcPacksThreshold {
		report.GCAdvised = true
		report.GCReasons = append(report.GCReasons,
			fmt.Sprintf("%d pack files (consolidation recommended)", report.Packs))
	}

	// Garbage files present.
	if report.Garbage >= gcGarbageThreshold {
		report.GCAdvised = true
		report.GCReasons = append(report.GCReasons,
			fmt.Sprintf("%d garbage files (%d KB)", report.Garbage, report.GarbageSizeKB))
	}
}

// SPDX-License-Identifier: Apache-2.0

package models

// HealthReport holds the result of a repository health check.
type HealthReport struct {
	// FSCKErrors lists corruption issues found by git fsck --connectivity-only.
	FSCKErrors []string

	// OK is true when no integrity issues are found.
	OK bool

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

	// PrunePackable is the number of loose objects also present in a pack.
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

// StaleSubmoduleDir is an orphan directory under .git/modules/ whose
// submodule name is not referenced by any [submodule "..."] stanza in
// the repository's .git/config. These are historical leftovers (removed
// submodules, renamed paths) that a standard git gc does not reclaim.
type StaleSubmoduleDir struct {
	// Name is the submodule name (path under .git/modules/, may contain slashes).
	Name string

	// Path is the absolute filesystem path to the orphan module directory.
	Path string

	// SizeBytes is the total on-disk size of the orphan directory.
	SizeBytes int64
}

// RepoSize holds size metrics for a repository.
type RepoSize struct {
	// GitDirBytes is the total size of the .git directory on disk.
	GitDirBytes int64

	// ReachableBytes is the total size of all reachable objects.
	ReachableBytes int64

	// RepackAdvised is true when conditions suggest git repack would be beneficial.
	// Triggered by pack count, loose/packed ratio, or oversized .git.
	// A standard aggressive gc fixes these.
	RepackAdvised bool

	// RepackReasons lists human-readable reasons why repack is advised.
	RepackReasons []string

	// UnreachableBloat is true when the .git directory is significantly
	// larger than reachable objects, indicating unreachable objects held
	// alive by reflog entries or kept by the default grace period.
	// A standard gc does not reclaim this space; a deep clean is required
	// (reflog expiry + gc --prune=now).
	UnreachableBloat bool

	// UnreachableBloatReasons lists human-readable reasons why deep clean
	// is advised.
	UnreachableBloatReasons []string
}

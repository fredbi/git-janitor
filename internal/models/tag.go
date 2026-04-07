// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// Tag represents a git tag with metadata.
type Tag struct {
	// Name is the tag name (e.g. "v1.2.3").
	Name string

	// Hash is the object hash the tag points to.
	Hash string

	// TargetHash is the commit hash the tag ultimately points to.
	TargetHash string

	// Date is the tagger date (annotated) or commit date (lightweight).
	Date time.Time

	// Message is the tag message (empty for lightweight tags).
	Message string

	// Annotated is true for annotated tags (objecttype == "tag").
	Annotated bool

	// Signed is true if the tag has a GPG/SSH signature.
	Signed bool

	// IsSemver is true if the tag matches semver pattern.
	IsSemver bool

	// HasVPrefix is true if the semver tag starts with "v".
	HasVPrefix bool

	// IsPrerelease is true if the semver tag has a prerelease suffix.
	IsPrerelease bool

	// OnDefaultBranch is true if the tagged commit is reachable from the default branch.
	OnDefaultBranch bool

	// LocalOnly is true if the tag exists locally but not on the origin remote.
	LocalOnly bool

	// RemoteOnly is true if the tag exists on the origin remote but not locally.
	RemoteOnly bool

	// SemverMajor, SemverMinor, SemverPatch hold the parsed version components.
	SemverMajor int
	SemverMinor int
	SemverPatch int

	// SemverPrerelease holds the prerelease suffix (e.g. "beta.1").
	SemverPrerelease string
}

// CompareSemver returns:
//
//	-1 if a < b
//	 0 if a == b
//	+1 if a > b
//
// Prerelease tags sort before the corresponding release (1.2.3-beta < 1.2.3).
func CompareSemver(a, b Tag) int {
	if a.SemverMajor != b.SemverMajor {
		return cmpInt(a.SemverMajor, b.SemverMajor)
	}

	if a.SemverMinor != b.SemverMinor {
		return cmpInt(a.SemverMinor, b.SemverMinor)
	}

	if a.SemverPatch != b.SemverPatch {
		return cmpInt(a.SemverPatch, b.SemverPatch)
	}

	// Prerelease sorts before release.
	if a.IsPrerelease && !b.IsPrerelease {
		return -1
	}

	if !a.IsPrerelease && b.IsPrerelease {
		return 1
	}

	// Both prerelease or both release: compare prerelease strings lexicographically.
	if a.SemverPrerelease < b.SemverPrerelease {
		return -1
	}

	if a.SemverPrerelease > b.SemverPrerelease {
		return 1
	}

	return 0
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}

	return 1
}

// DeriveTagSummary computes LastTagDate, LastSemverTag, and LastSemverDate from a tag list.
func DeriveTagSummary(tags []Tag) (lastTagDate time.Time, lastSemverTag string, lastSemverDate time.Time) {
	var bestSemver *Tag

	for i := range tags {
		t := &tags[i]

		// Skip remote-only tags (we don't have their date).
		if t.RemoteOnly {
			continue
		}

		// Last tag by date (any tag).
		if t.Date.After(lastTagDate) {
			lastTagDate = t.Date
		}

		// Last semver tag by version ordering (non-prerelease preferred).
		if !t.IsSemver {
			continue
		}

		if bestSemver == nil || CompareSemver(*t, *bestSemver) > 0 {
			bestSemver = t
		}
	}

	if bestSemver != nil {
		lastSemverTag = bestSemver.Name
		lastSemverDate = bestSemver.Date
	}

	return lastTagDate, lastSemverTag, lastSemverDate
}

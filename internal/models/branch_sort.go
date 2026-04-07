// SPDX-License-Identifier: Apache-2.0

package models

import (
	"slices"
	"strings"
)

// SortBranches sorts branches for display:
//
// Local branches first:
//   - Default branch always first
//   - Current branch second (if not default)
//   - Other local branches by descending last commit date
//
// Remote-only branches after locals:
//   - Ordered by remote name (origin first, upstream second, others alphabetically)
//   - Within same remote, by descending last commit date
func SortBranches(branches []Branch, defaultBranch string) {
	slices.SortStableFunc(branches, func(a, b Branch) int {
		// Local before remote.
		if !a.IsRemote && b.IsRemote {
			return -1
		}

		if a.IsRemote && !b.IsRemote {
			return 1
		}

		// Both local.
		if !a.IsRemote && !b.IsRemote {
			return compareLocalBranches(a, b, defaultBranch)
		}

		// Both remote.
		return compareRemoteBranches(a, b)
	})
}

func compareLocalBranches(a, b Branch, defaultBranch string) int {
	// Default branch always first.
	if a.Name == defaultBranch && b.Name != defaultBranch {
		return -1
	}

	if b.Name == defaultBranch && a.Name != defaultBranch {
		return 1
	}

	// Current branch second.
	if a.IsCurrent && !b.IsCurrent {
		return -1
	}

	if b.IsCurrent && !a.IsCurrent {
		return 1
	}

	// By last commit date descending (most recent first).
	return b.LastCommit.Compare(a.LastCommit)
}

func compareRemoteBranches(a, b Branch) int {
	remoteA := remoteName(a.Name)
	remoteB := remoteName(b.Name)

	if remoteA != remoteB {
		return compareRemoteNames(remoteA, remoteB)
	}

	// Same remote: by last commit date descending.
	return b.LastCommit.Compare(a.LastCommit)
}

// remoteName extracts the remote prefix from a remote branch name
// (e.g. "origin/main" → "origin").
func remoteName(branchName string) string {
	if remote, _, ok := strings.Cut(branchName, "/"); ok {
		return remote
	}

	return branchName
}

// compareRemoteNames orders remotes: origin first, upstream second, others alphabetically.
func compareRemoteNames(a, b string) int {
	oa, ob := remoteOrder(a), remoteOrder(b)
	if oa != ob {
		return oa - ob
	}

	// Both are "other" remotes: sort alphabetically.
	return strings.Compare(a, b)
}

func remoteOrder(name string) int {
	switch name {
	case RemoteOrigin:
		return 0
	case RemoteUpstream:
		return 1
	default:
		return 2 //nolint:mnd // origin=0, upstream=1, others=2+
	}
}

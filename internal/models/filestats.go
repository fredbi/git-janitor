// SPDX-License-Identifier: Apache-2.0

package models

// FileStats holds information about large and binary files in the repository.
type FileStats struct {
	// LargeFiles lists files in HEAD that exceed the size threshold,
	// sorted by size descending.
	LargeFiles []FileEntry

	// LargeBlobs lists the largest blob objects across all history,
	// sorted by size descending.
	LargeBlobs []BlobEntry

	// BinaryFiles lists files in HEAD that git considers binary.
	BinaryFiles []string
}

// FileEntry represents a file in the current tree with its size.
type FileEntry struct {
	Path string
	Size int64
}

// BlobEntry represents a blob object with its size and associated path.
type BlobEntry struct {
	Hash string
	Size int64
	Path string // may be empty for orphaned blobs
}

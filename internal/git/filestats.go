package git

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"
)

// emptyTreeHashFallback is the well-known SHA-1 of an empty tree in git.
// Used only if the dynamic lookup fails.
const emptyTreeHashFallback = "4b825dc642cb6eb9a060e54bf899d69f82cf7137"

// FileStats holds information about large and binary files in the repository.
type FileStats struct {
	// LargeFiles lists files in HEAD that exceed the size threshold,
	// sorted by size descending.
	LargeFiles []FileEntry

	// LargeBlobs lists the largest blob objects across all history,
	// sorted by size descending. These may include files that have been
	// deleted but still occupy space in the pack.
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

const (
	// defaultLargeFileThreshold is the minimum size (bytes) for a file to be
	// reported as "large". Default: 1 MB.
	defaultLargeFileThreshold = 1 << 20

	// defaultTopBlobs is the maximum number of large blobs to report.
	defaultTopBlobs = 20
)

// FileStatsOption configures the FileStats query.
type FileStatsOption func(*fileStatsConfig)

type fileStatsConfig struct {
	largeThreshold int64
	topBlobs       int
}

// WithLargeThreshold sets the minimum file size (bytes) to report as large.
func WithLargeThreshold(bytes int64) FileStatsOption {
	return func(c *fileStatsConfig) { c.largeThreshold = bytes }
}

// WithTopBlobs sets the maximum number of large blobs to return.
func WithTopBlobs(n int) FileStatsOption {
	return func(c *fileStatsConfig) { c.topBlobs = n }
}

// FileStats collects information about large and binary files.
//
// Large files in HEAD are found via git ls-tree -r -l HEAD.
// Large blobs across all history are found via git rev-list + cat-file.
// Binary files are detected via git diff --numstat against an empty tree.
func (r *Runner) FileStats(ctx context.Context, opts ...FileStatsOption) FileStats {
	cfg := fileStatsConfig{
		largeThreshold: defaultLargeFileThreshold,
		topBlobs:       defaultTopBlobs,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	var stats FileStats

	stats.LargeFiles = r.findLargeFiles(ctx, cfg.largeThreshold)
	stats.LargeBlobs = r.findLargeBlobs(ctx, cfg.topBlobs)
	stats.BinaryFiles = r.findBinaryFiles(ctx)

	return stats
}

// findLargeFiles lists files in HEAD exceeding the threshold.
// Uses: git ls-tree -r -l HEAD
// Output: mode type hash size\tpath
func (r *Runner) findLargeFiles(ctx context.Context, threshold int64) []FileEntry {
	out, err := r.run(ctx, cmdLsTree()...)
	if err != nil {
		return nil
	}

	var large []FileEntry

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()

		// Format: "mode type hash    size\tpath"
		// The size field is right-aligned with spaces before the tab.
		tabIdx := strings.IndexByte(line, '\t')
		if tabIdx < 0 {
			continue
		}

		meta := line[:tabIdx]
		path := line[tabIdx+1:]

		// Split meta fields.
		fields := strings.Fields(meta)
		if len(fields) < 4 {
			continue
		}

		// fields[3] is the size (or "-" for submodules).
		if fields[3] == "-" {
			continue
		}

		var size int64

		fmt.Sscanf(fields[3], "%d", &size) //nolint:errcheck // best-effort

		if size >= threshold {
			large = append(large, FileEntry{Path: path, Size: size})
		}
	}

	sort.Slice(large, func(i, j int) bool { return large[i].Size > large[j].Size })

	return large
}

// findLargeBlobs finds the largest blobs across all history.
// Uses: git rev-list --objects --all for paths, then
// extracts OIDs and pipes them to git cat-file --batch-check for sizes.
func (r *Runner) findLargeBlobs(ctx context.Context, topN int) []BlobEntry {
	// Get all objects with paths.
	objOut, err := r.run(ctx, cmdRevListObjects()...)
	if err != nil {
		return nil
	}

	// Build OID → path map and collect bare OIDs for cat-file.
	paths := make(map[string]string)

	var oids strings.Builder

	for _, line := range strings.Split(objOut, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		oid, path, _ := strings.Cut(line, " ")
		if path != "" {
			paths[oid] = path
		}

		oids.WriteString(oid)
		oids.WriteByte('\n')
	}

	// Pipe bare OIDs to cat-file --batch-check for type and size.
	checkOut, err := r.runWithStdin(ctx, oids.String(), cmdCatFileBatchCheck()...)
	if err != nil {
		return nil
	}

	var blobs []BlobEntry

	scanner := bufio.NewScanner(strings.NewReader(checkOut))
	for scanner.Scan() {
		line := scanner.Text()

		// Format: "blob 12345 abc1234..."
		if !strings.HasPrefix(line, "blob ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		var size int64

		fmt.Sscanf(fields[1], "%d", &size) //nolint:errcheck // best-effort

		hash := fields[2]

		blobs = append(blobs, BlobEntry{
			Hash: hash,
			Size: size,
			Path: paths[hash],
		})
	}

	sort.Slice(blobs, func(i, j int) bool { return blobs[i].Size > blobs[j].Size })

	if len(blobs) > topN {
		blobs = blobs[:topN]
	}

	return blobs
}

// emptyTreeHash returns the empty tree hash for the current git installation.
// It runs `git hash-object -t tree /dev/null` and falls back to the well-known SHA-1.
func (r *Runner) emptyTreeHash(ctx context.Context) string {
	out, err := r.run(ctx, "hash-object", "-t", "tree", "/dev/null")
	if err != nil {
		return emptyTreeHashFallback
	}

	h := strings.TrimSpace(out)
	if h == "" {
		return emptyTreeHashFallback
	}

	return h
}

// findBinaryFiles lists files in HEAD that git considers binary.
// Uses: git diff --numstat <empty-tree> HEAD
// Binary files show as "-\t-\tpath".
func (r *Runner) findBinaryFiles(ctx context.Context) []string {
	out, err := r.run(ctx, cmdDiffNumstat(r.emptyTreeHash(ctx))...)
	if err != nil {
		return nil
	}

	var binaries []string

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()

		// Binary files: "-\t-\tpath"
		if strings.HasPrefix(line, "-\t-\t") {
			path := strings.TrimPrefix(line, "-\t-\t")
			binaries = append(binaries, path)
		}
	}

	return binaries
}


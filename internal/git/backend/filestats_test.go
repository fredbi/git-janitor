package backend

import (
	"context"
	"os"
	"testing"
)

func TestIntegration_FileStats(t *testing.T) {
	r := &Runner{Dir: "."}

	stats := r.FileStats(context.Background())

	t.Logf("large files (>1MB): %d", len(stats.LargeFiles))
	for _, f := range stats.LargeFiles {
		t.Logf("  %s (%s)", f.Path, formatBytes(f.Size))
	}

	t.Logf("top blobs: %d", len(stats.LargeBlobs))
	for i, b := range stats.LargeBlobs {
		if i >= 5 {
			t.Logf("  ... and %d more", len(stats.LargeBlobs)-5)

			break
		}

		t.Logf("  %s %s (%s)", b.Hash[:8], b.Path, formatBytes(b.Size))
	}

	t.Logf("binary files: %d", len(stats.BinaryFiles))
	for _, f := range stats.BinaryFiles {
		t.Logf("  %s", f)
	}
}

func TestIntegration_FileStats_LowThreshold(t *testing.T) {
	r := &Runner{Dir: "."}

	// Use a low threshold to get results even on this small repo.
	stats := r.FileStats(context.Background(), WithLargeThreshold(1024), WithTopBlobs(5))

	if len(stats.LargeBlobs) == 0 {
		t.Error("expected at least 1 large blob with 1KB threshold")
	}

	// Verify blobs are sorted descending.
	for i := 1; i < len(stats.LargeBlobs); i++ {
		if stats.LargeBlobs[i].Size > stats.LargeBlobs[i-1].Size {
			t.Errorf("blobs not sorted: [%d].Size=%d > [%d].Size=%d",
				i, stats.LargeBlobs[i].Size, i-1, stats.LargeBlobs[i-1].Size)
		}
	}

	if len(stats.LargeBlobs) > 5 {
		t.Errorf("expected at most 5 blobs with WithTopBlobs(5), got %d", len(stats.LargeBlobs))
	}
}

func TestIntegration_FileStats_ExternalRepo(t *testing.T) {
	// Test against a repo known to have LFS and binary files.
	repoPath := "/home/fred/src/github.com/oneconcern/geodude"
	if _, err := os.Stat(repoPath); err != nil {
		t.Skipf("external repo not available: %s", repoPath)
	}

	r := &Runner{Dir: repoPath}
	ctx := context.Background()

	t.Logf("HasLFS: %v", r.HasLFS())

	stats := r.FileStats(ctx, WithLargeThreshold(100*1024), WithTopBlobs(10))

	t.Logf("large files (>100KB): %d", len(stats.LargeFiles))
	for _, f := range stats.LargeFiles {
		t.Logf("  %s (%s)", f.Path, formatBytes(f.Size))
	}

	t.Logf("top blobs: %d", len(stats.LargeBlobs))
	for _, b := range stats.LargeBlobs {
		t.Logf("  %s %s (%s)", b.Hash[:8], b.Path, formatBytes(b.Size))
	}

	t.Logf("binary files: %d", len(stats.BinaryFiles))
	for i, f := range stats.BinaryFiles {
		if i >= 10 {
			t.Logf("  ... and %d more", len(stats.BinaryFiles)-10)

			break
		}

		t.Logf("  %s", f)
	}
}

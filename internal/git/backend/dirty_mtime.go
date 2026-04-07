package backend

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// deriveLastLocalUpdate computes the most recent local activity timestamp.
// If the worktree is clean, this is the last commit time.
// If the worktree is dirty, this is the newest mtime among uncommitted files
// (or the last commit time if no dirty files can be stat'd).
func deriveLastLocalUpdate(repoDir string, status models.Status, lastCommit time.Time) time.Time {
	if !status.IsDirty() {
		return lastCommit
	}

	dirtyMtime := newestDirtyFileMtime(repoDir, status.Entries)
	if dirtyMtime.IsZero() {
		return lastCommit
	}

	if dirtyMtime.After(lastCommit) {
		return dirtyMtime
	}

	return lastCommit
}

// newestDirtyFileMtime returns the most recent modification time among
// the files listed in the status entries. Only tracked changes and
// untracked files are considered (not ignored files).
//
// Returns zero time if no files can be stat'd.
func newestDirtyFileMtime(repoDir string, entries []models.StatusEntry) time.Time {
	var newest time.Time

	for i := range entries {
		e := &entries[i]
		if e.IsIgnored() {
			continue
		}

		pth := filepath.Join(repoDir, e.Path)

		info, err := os.Lstat(pth)
		if err != nil {
			continue // file may have been deleted
		}

		if mt := info.ModTime(); mt.After(newest) {
			newest = mt
		}
	}

	return newest
}

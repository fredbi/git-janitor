package git

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
)

// IsShallow reports whether the repository is a shallow clone.
func (r *Runner) IsShallow(ctx context.Context) bool {
	out, err := r.run(ctx, cmdIsShallow()...)
	if err != nil {
		return false
	}

	return strings.TrimSpace(out) == "true"
}

// HasSubmodules reports whether the repository contains git submodules.
// It checks for the presence of a .gitmodules file in the repository root.
func (r *Runner) HasSubmodules() bool {
	info, err := os.Stat(filepath.Join(r.Dir, ".gitmodules"))

	return err == nil && !info.IsDir()
}

// HasLFS reports whether the repository uses Git LFS.
// It checks for "filter=lfs" in .gitattributes (at the repo root).
func (r *Runner) HasLFS() bool {
	path := filepath.Join(r.Dir, ".gitattributes")

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "filter=lfs") {
			return true
		}
	}

	return false
}

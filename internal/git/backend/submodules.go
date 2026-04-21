// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// StaleSubmoduleDirs walks .git/modules/ and returns every module directory
// whose submodule is no longer live in the repository.
//
// A submodule is live when either:
//   - its name appears in .gitmodules at HEAD; or
//   - .git/config declares submodule.<name>.path and that path has a
//     gitlink (mode 160000) entry at HEAD.
//
// A bare [submodule] stanza in .git/config without a live gitlink or
// .gitmodules entry is treated as stale — git rm removes the gitlink and
// the .gitmodules entry but leaves the .git/config section behind, so the
// module dir lingers despite the config record.
//
// Module directories are identified by the presence of a HEAD file (which
// distinguishes actual module git dirs from the parent path components
// that appear when submodule names contain slashes).
//
// Returns nil if .git/modules/ does not exist.
func (r *Runner) StaleSubmoduleDirs(ctx context.Context) []models.StaleSubmoduleDir {
	modulesDir := filepath.Join(r.gitDir(ctx), "modules")
	if _, err := os.Stat(modulesDir); err != nil {
		return nil
	}

	moduleDirs := findModuleDirs(modulesDir)
	if len(moduleDirs) == 0 {
		return nil
	}

	live := r.liveSubmoduleNames(ctx)

	var stale []models.StaleSubmoduleDir

	for name, path := range moduleDirs {
		if _, ok := live[name]; ok {
			continue
		}

		stale = append(stale, models.StaleSubmoduleDir{
			Name:      name,
			Path:      path,
			SizeBytes: gitDirSize(path),
		})
	}

	return stale
}

// liveSubmoduleNames returns the set of submodule names that the repo
// still uses — i.e., either recorded in .gitmodules at HEAD, or declared
// in .git/config with a path that has a gitlink at HEAD.
func (r *Runner) liveSubmoduleNames(ctx context.Context) map[string]struct{} {
	live := make(map[string]struct{})

	// (1) Names in .gitmodules at HEAD (authoritative for tracked submodules).
	for name := range r.gitmodulesNamesAtHEAD(ctx) {
		live[name] = struct{}{}
	}

	// (2) Names in .git/config whose configured path has a gitlink at HEAD.
	gitlinks := r.gitlinkPathsAtHEAD(ctx)

	for name, path := range r.configuredSubmodulePaths(ctx) {
		if _, ok := gitlinks[path]; ok {
			live[name] = struct{}{}
		}
	}

	return live
}

// gitmodulesNamesAtHEAD parses .gitmodules as it exists at HEAD and returns
// the submodule names recorded there.
func (r *Runner) gitmodulesNamesAtHEAD(ctx context.Context) map[string]struct{} {
	out, err := r.run(ctx, cmdConfigGetRegexpBlob("HEAD:.gitmodules", `^submodule\..+\.path$`)...)
	if err != nil {
		return nil
	}

	return parseSubmoduleNames(out, ".path")
}

// configuredSubmodulePaths returns submodule name → configured path from
// .git/config for every [submodule "<name>"] stanza that has a path set.
func (r *Runner) configuredSubmodulePaths(ctx context.Context) map[string]string {
	out, err := r.run(ctx, cmdConfigGetRegexp(`^submodule\..+\.path$`)...)
	if err != nil {
		return nil
	}

	paths := make(map[string]string)

	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// With --show-scope, output is: "<scope>\tsubmodule.<name>.path <value>"
		if _, rest, ok := strings.Cut(line, "\t"); ok {
			line = rest
		}

		key, value, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}

		name := trimSubmoduleKey(key, ".path")
		if name != "" {
			paths[name] = strings.TrimSpace(value)
		}
	}

	return paths
}

// gitlinkPathsAtHEAD returns the set of paths at HEAD that are gitlinks
// (mode 160000 — i.e. active submodules in the current tree).
func (r *Runner) gitlinkPathsAtHEAD(ctx context.Context) map[string]struct{} {
	out, err := r.run(ctx, cmdLsTreeHEAD()...)
	if err != nil {
		return nil
	}

	paths := make(map[string]struct{})

	for line := range strings.SplitSeq(out, "\n") {
		// Format: "<mode> <type> <sha>\t<path>"
		if !strings.HasPrefix(line, "160000 ") {
			continue
		}

		_, path, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}

		path = strings.TrimSpace(path)
		if path != "" {
			paths[path] = struct{}{}
		}
	}

	return paths
}

// parseSubmoduleNames extracts submodule names from `git config --get-regexp`
// output where the key has the shape `submodule.<name><suffix>`.
// The --show-scope prefix (if present) is stripped.
func parseSubmoduleNames(out, suffix string) map[string]struct{} {
	names := make(map[string]struct{})

	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Strip optional "<scope>\t" prefix from --show-scope output.
		if _, rest, ok := strings.Cut(line, "\t"); ok {
			line = rest
		}

		key, _, _ := strings.Cut(line, " ")

		name := trimSubmoduleKey(key, suffix)
		if name != "" {
			names[name] = struct{}{}
		}
	}

	return names
}

// trimSubmoduleKey turns "submodule.<name><suffix>" into "<name>", or
// returns "" if the key doesn't match.
func trimSubmoduleKey(key, suffix string) string {
	const prefix = "submodule."

	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return ""
	}

	return strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
}

// findModuleDirs returns a map of submodule-name → absolute path for every
// module directory under modulesRoot. A submodule name is the path relative
// to modulesRoot (using forward slashes, matching what git stores in
// .git/config).
func findModuleDirs(modulesRoot string) map[string]string {
	out := make(map[string]string)

	//nolint:errcheck // best-effort walk
	filepath.Walk(modulesRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries
		}

		if !info.IsDir() {
			return nil
		}

		// A module git-dir has a HEAD file.
		if _, headErr := os.Stat(filepath.Join(path, "HEAD")); headErr != nil {
			return nil //nolint:nilerr // not a module dir, keep walking
		}

		rel, relErr := filepath.Rel(modulesRoot, path)
		if relErr != nil || rel == "." {
			return nil //nolint:nilerr // skip root and unreadable
		}

		out[filepath.ToSlash(rel)] = path

		// Don't descend into an identified module dir: nested submodules
		// are handled by the parent module's own .git/modules, not ours.
		return filepath.SkipDir
	})

	return out
}

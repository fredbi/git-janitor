// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindModuleDirs(t *testing.T) {
	root := t.TempDir()

	// Top-level module: .git/modules/simple/HEAD + config
	mustMkdir(t, filepath.Join(root, "simple"))
	mustTouch(t, filepath.Join(root, "simple", "HEAD"))
	mustTouch(t, filepath.Join(root, "simple", "config"))

	// Nested module name: .git/modules/vendor/foo/ is the real git-dir,
	// .git/modules/vendor/ is just a path component.
	mustMkdir(t, filepath.Join(root, "vendor", "foo"))
	mustTouch(t, filepath.Join(root, "vendor", "foo", "HEAD"))

	// Decoy: a random directory with no HEAD should not be reported.
	mustMkdir(t, filepath.Join(root, "not-a-module"))

	got := findModuleDirs(root)

	if _, ok := got["simple"]; !ok {
		t.Errorf("missing 'simple', got keys: %v", keys(got))
	}

	if _, ok := got["vendor/foo"]; !ok {
		t.Errorf("missing 'vendor/foo', got keys: %v", keys(got))
	}

	if _, ok := got["vendor"]; ok {
		t.Errorf("reported parent path 'vendor' as a module dir")
	}

	if _, ok := got["not-a-module"]; ok {
		t.Errorf("reported HEAD-less dir 'not-a-module' as a module dir")
	}

	if len(got) != 2 {
		t.Errorf("expected 2 module dirs, got %d: %v", len(got), keys(got))
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()

	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func mustTouch(t *testing.T, p string) {
	t.Helper()

	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("create %s: %v", p, err)
	}

	_ = f.Close()
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

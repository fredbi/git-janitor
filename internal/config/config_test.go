package config

import (
	"bytes"
	"io/fs"
	"testing"
	"time"
)

func TestLoadDefaults_ScheduleInterval(t *testing.T) {
	cfg, err := loadDefaults()
	if err != nil {
		t.Fatalf("loadDefaults() error: %v", err)
	}

	if cfg.Defaults.RootConfig == nil {
		t.Fatal("Defaults.RootConfig is nil")
	}

	got := cfg.Defaults.RootConfig.ScheduleInterval
	want := 5 * time.Minute

	if got != want {
		t.Errorf("ScheduleInterval = %v, want %v", got, want)
	}
}

func TestLoad_DurationParsing(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    time.Duration
		wantErr bool
	}{
		{
			name: "minutes",
			yaml: `roots:
- path: /tmp/repos
  rootConfig:
    scheduleInterval: 5m
`,
			want: 5 * time.Minute,
		},
		{
			name: "hours",
			yaml: `roots:
- path: /tmp/repos
  rootConfig:
    scheduleInterval: 2h
`,
			want: 2 * time.Hour,
		},
		{
			name: "compound",
			yaml: `roots:
- path: /tmp/repos
  rootConfig:
    scheduleInterval: 1h30m
`,
			want: 1*time.Hour + 30*time.Minute,
		},
		{
			name: "seconds",
			yaml: `roots:
- path: /tmp/repos
  rootConfig:
    scheduleInterval: 90s
`,
			want: 90 * time.Second,
		},
		{
			name:    "invalid duration",
			yaml:    "roots:\n- path: /tmp\n  rootConfig:\n    scheduleInterval: 5mn\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			_, err := load(fakeFS(tt.yaml), "config.yaml", cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("load() error: %v", err)
			}

			if len(cfg.Roots) == 0 {
				t.Fatal("no roots parsed")
			}

			got := cfg.Roots[0].RootConfig.ScheduleInterval
			if got != tt.want {
				t.Errorf("ScheduleInterval = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodeYAML_ExcludesRepos(t *testing.T) {
	cfg := &Config{}
	cfg.AddRoot("", "/tmp/repos", 10*time.Minute)
	cfg.Roots[0].Repos = []Repository{
		{Path: "/tmp/repos/foo"},
	}

	var buf bytes.Buffer
	if err := cfg.EncodeYAML(&buf); err != nil {
		t.Fatalf("EncodeYAML() error: %v", err)
	}

	out := buf.String()
	if bytes.Contains(buf.Bytes(), []byte("foo")) {
		t.Errorf("EncodeYAML output should not contain runtime Repos, got:\n%s", out)
	}
}

func TestRoundTrip(t *testing.T) {
	// Build a config, encode it, decode it, check values survive.
	original := &Config{}
	original.AddRoot("my-projects", "/home/dev/projects", 15*time.Minute)
	original.Roots[0].Repos = []Repository{{Path: "should-be-excluded"}}

	var buf bytes.Buffer
	if err := original.EncodeYAML(&buf); err != nil {
		t.Fatalf("EncodeYAML() error: %v", err)
	}

	restored := &Config{}
	_, err := load(fakeFS(buf.String()), "config.yaml", restored)
	if err != nil {
		t.Fatalf("load() error: %v", err)
	}

	if len(restored.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(restored.Roots))
	}

	got := restored.Roots[0].RootConfig.ScheduleInterval
	want := 15 * time.Minute
	if got != want {
		t.Errorf("round-tripped ScheduleInterval = %v, want %v", got, want)
	}

	if restored.Roots[0].Path != "/home/dev/projects" {
		t.Errorf("round-tripped Path = %q, want %q", restored.Roots[0].Path, "/home/dev/projects")
	}

	if restored.Roots[0].Name != "my-projects" {
		t.Errorf("round-tripped Name = %q, want %q", restored.Roots[0].Name, "my-projects")
	}

	if len(restored.Roots[0].Repos) != 0 {
		t.Errorf("Repos should not survive round-trip, got %d", len(restored.Roots[0].Repos))
	}
}

func TestAddRoot_DefaultName(t *testing.T) {
	cfg := &Config{}

	// When name is empty, it defaults to basename of path.
	idx := cfg.AddRoot("", "/home/dev/projects", 5*time.Minute)
	if cfg.Roots[idx].Name != "projects" {
		t.Errorf("AddRoot empty name: got %q, want %q", cfg.Roots[idx].Name, "projects")
	}

	// When name is provided, it is used as-is.
	idx = cfg.AddRoot("custom", "/home/dev/work", 5*time.Minute)
	if cfg.Roots[idx].Name != "custom" {
		t.Errorf("AddRoot explicit name: got %q, want %q", cfg.Roots[idx].Name, "custom")
	}
}

func TestRootDisplayName(t *testing.T) {
	cfg := &Config{}
	cfg.AddRoot("explicit", "/home/dev/projects", 5*time.Minute)
	cfg.AddRoot("", "/home/dev/work", 5*time.Minute)

	if got := cfg.RootDisplayName(0); got != "explicit" {
		t.Errorf("RootDisplayName(0) = %q, want %q", got, "explicit")
	}

	// Second root has name "work" (set by AddRoot default), so RootDisplayName returns it.
	if got := cfg.RootDisplayName(1); got != "work" {
		t.Errorf("RootDisplayName(1) = %q, want %q", got, "work")
	}

	// Clear the name to test fallback to basename.
	cfg.Roots[1].Name = ""
	if got := cfg.RootDisplayName(1); got != "work" {
		t.Errorf("RootDisplayName(1) with empty Name = %q, want %q", got, "work")
	}

	// Out-of-range returns empty.
	if got := cfg.RootDisplayName(99); got != "" {
		t.Errorf("RootDisplayName(99) = %q, want empty", got)
	}
}

func TestUpdateRootName(t *testing.T) {
	cfg := &Config{}
	cfg.AddRoot("original", "/home/dev/projects", 5*time.Minute)

	if idx := cfg.UpdateRootName(0, "renamed"); idx != 0 {
		t.Fatalf("UpdateRootName(0) returned %d, want 0", idx)
	}

	if cfg.Roots[0].Name != "renamed" {
		t.Errorf("Name after update = %q, want %q", cfg.Roots[0].Name, "renamed")
	}

	// Out-of-range returns -1.
	if idx := cfg.UpdateRootName(5, "nope"); idx != -1 {
		t.Errorf("UpdateRootName(5) = %d, want -1 for out-of-range index", idx)
	}
}

func TestRootMaxDepth_DefaultsWhenZero(t *testing.T) {
	cfg := &Config{}
	cfg.Roots = []LocalRoot{
		{Path: "/p/a", RootConfig: RootConfig{MaxDepth: 0}},  // unset → default
		{Path: "/p/b", RootConfig: RootConfig{MaxDepth: 1}},  // explicit flat
		{Path: "/p/c", RootConfig: RootConfig{MaxDepth: -1}}, // explicit unlimited
	}

	if got := cfg.RootMaxDepth(0); got != DefaultMaxDepth {
		t.Errorf("RootMaxDepth(0) = %d, want %d", got, DefaultMaxDepth)
	}
	if got := cfg.RootMaxDepth(1); got != 1 {
		t.Errorf("RootMaxDepth(1) = %d, want 1", got)
	}
	if got := cfg.RootMaxDepth(2); got != -1 {
		t.Errorf("RootMaxDepth(2) = %d, want -1", got)
	}
	if got := cfg.RootMaxDepth(99); got != DefaultMaxDepth {
		t.Errorf("RootMaxDepth(out-of-range) = %d, want %d", got, DefaultMaxDepth)
	}
}

func TestUpdateRootMaxDepth(t *testing.T) {
	cfg := &Config{}
	cfg.AddRoot("a", "/p/a", time.Minute)

	// AddRoot must initialize MaxDepth to DefaultMaxDepth.
	if got := cfg.Roots[0].RootConfig.MaxDepth; got != DefaultMaxDepth {
		t.Errorf("AddRoot left MaxDepth = %d, want %d", got, DefaultMaxDepth)
	}

	if !cfg.UpdateRootMaxDepth(0, 2) {
		t.Fatal("UpdateRootMaxDepth returned false")
	}
	if got := cfg.Roots[0].RootConfig.MaxDepth; got != 2 {
		t.Errorf("MaxDepth after update = %d, want 2", got)
	}

	if cfg.UpdateRootMaxDepth(99, 3) {
		t.Error("UpdateRootMaxDepth(99) should return false for out-of-range index")
	}
}

func TestRoundTrip_MaxDepth(t *testing.T) {
	original := &Config{}
	original.AddRoot("alpha", "/p/a", time.Minute)
	original.UpdateRootMaxDepth(0, 3)

	var buf bytes.Buffer
	if err := original.EncodeYAML(&buf); err != nil {
		t.Fatalf("EncodeYAML: %v", err)
	}

	restored := &Config{}
	if _, err := load(fakeFS(buf.String()), "config.yaml", restored); err != nil {
		t.Fatalf("load: %v", err)
	}

	if got := restored.Roots[0].RootConfig.MaxDepth; got != 3 {
		t.Errorf("round-tripped MaxDepth = %d, want 3", got)
	}
}

func TestSortRoots_Alphabetical(t *testing.T) {
	cfg := &Config{}

	// Add roots out of order; sort happens inside AddRoot.
	cfg.AddRoot("zeta", "/p/z", time.Minute)
	cfg.AddRoot("alpha", "/p/a", time.Minute)
	cfg.AddRoot("Mike", "/p/m", time.Minute) // case-insensitive

	want := []string{"alpha", "Mike", "zeta"}
	for i, w := range want {
		if got := cfg.Roots[i].Name; got != w {
			t.Errorf("Roots[%d].Name = %q, want %q", i, got, w)
		}
	}
}

func TestSortRoots_FallsBackToBasename(t *testing.T) {
	cfg := &Config{}
	// All names empty — display name comes from basename(path).
	cfg.Roots = []LocalRoot{
		{Path: "/p/zebra"},
		{Path: "/p/apple"},
		{Path: "/p/mango"},
	}
	cfg.SortRoots()

	want := []string{"/p/apple", "/p/mango", "/p/zebra"}
	for i, w := range want {
		if got := cfg.Roots[i].Path; got != w {
			t.Errorf("Roots[%d].Path = %q, want %q", i, got, w)
		}
	}
}

func TestAddRoot_ReturnsNewIndex(t *testing.T) {
	cfg := &Config{}

	// Insertion order: zeta, alpha. After sort, alpha is at 0, zeta at 1.
	zetaIdx := cfg.AddRoot("zeta", "/p/z", time.Minute)
	if zetaIdx != 0 {
		t.Fatalf("first AddRoot returned %d, want 0", zetaIdx)
	}

	alphaIdx := cfg.AddRoot("alpha", "/p/a", time.Minute)
	if alphaIdx != 0 {
		t.Errorf("AddRoot(alpha) returned %d, want 0 (sorts before zeta)", alphaIdx)
	}

	// Confirm zeta moved to index 1.
	if cfg.Roots[1].Name != "zeta" {
		t.Errorf("Roots[1].Name = %q, want %q", cfg.Roots[1].Name, "zeta")
	}
}

func TestUpdateRootName_FollowsSortedPosition(t *testing.T) {
	cfg := &Config{}
	cfg.AddRoot("alpha", "/p/a", time.Minute)
	cfg.AddRoot("beta", "/p/b", time.Minute)

	// Rename alpha → zulu; it should move from index 0 to index 1.
	newIdx := cfg.UpdateRootName(0, "zulu")
	if newIdx != 1 {
		t.Errorf("UpdateRootName returned %d, want 1", newIdx)
	}

	if cfg.Roots[newIdx].Name != "zulu" {
		t.Errorf("Roots[%d].Name = %q, want zulu", newIdx, cfg.Roots[newIdx].Name)
	}
	if cfg.Roots[0].Name != "beta" {
		t.Errorf("Roots[0].Name = %q, want beta", cfg.Roots[0].Name)
	}
}

func TestUpdateRootPath_FollowsSortedPosition(t *testing.T) {
	cfg := &Config{}
	// Construct directly so Name stays empty and the display name comes
	// from basename(Path). AddRoot would default Name to the basename,
	// which would defeat the rename-by-path scenario under test.
	cfg.Roots = []LocalRoot{
		{Path: "/p/alpha"},
		{Path: "/p/beta"},
	}
	cfg.SortRoots()

	// Move alpha → /p/zulu; the display name becomes "zulu" and the entry
	// should move from index 0 to index 1.
	newIdx := cfg.UpdateRootPath(0, "/p/zulu")
	if newIdx != 1 {
		t.Errorf("UpdateRootPath returned %d, want 1", newIdx)
	}
	if cfg.Roots[newIdx].Path != "/p/zulu" {
		t.Errorf("Roots[%d].Path = %q, want /p/zulu", newIdx, cfg.Roots[newIdx].Path)
	}
}

func TestLoadSortsRoots(t *testing.T) {
	yaml := "roots:\n" +
		"  - name: zeta\n    path: /p/z\n" +
		"  - name: alpha\n    path: /p/a\n"

	cfg := &Config{}
	if _, err := load(fakeFS(yaml), "config.yaml", cfg); err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(cfg.Roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(cfg.Roots))
	}
	if cfg.Roots[0].Name != "alpha" {
		t.Errorf("Roots[0].Name = %q, want alpha", cfg.Roots[0].Name)
	}
}

func TestRoundTrip_WithName(t *testing.T) {
	// Verify that a root with a name survives encode/decode,
	// and that an empty-name root also round-trips correctly.
	original := &Config{}
	original.AddRoot("named", "/home/dev/named", 10*time.Minute)
	original.AddRoot("", "/home/dev/unnamed", 20*time.Minute)

	var buf bytes.Buffer
	if err := original.EncodeYAML(&buf); err != nil {
		t.Fatalf("EncodeYAML() error: %v", err)
	}

	restored := &Config{}
	_, err := load(fakeFS(buf.String()), "config.yaml", restored)
	if err != nil {
		t.Fatalf("load() error: %v", err)
	}

	if len(restored.Roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(restored.Roots))
	}

	if restored.Roots[0].Name != "named" {
		t.Errorf("root[0].Name = %q, want %q", restored.Roots[0].Name, "named")
	}

	// The second root was added with empty name, so AddRoot defaulted to "unnamed".
	if restored.Roots[1].Name != "unnamed" {
		t.Errorf("root[1].Name = %q, want %q", restored.Roots[1].Name, "unnamed")
	}
}

const qaOpenInTerminal = "open-in-terminal"

func TestLoadDefaults_QuickActions(t *testing.T) {
	cfg, err := loadDefaults()
	if err != nil {
		t.Fatalf("loadDefaults() error: %v", err)
	}

	if len(cfg.QuickActions) == 0 {
		t.Fatal("expected at least one default quick action, got none")
	}

	var found bool
	for _, qa := range cfg.QuickActions {
		if qa.Name != qaOpenInTerminal {
			continue
		}

		found = true

		if qa.Subject != "repo" {
			t.Errorf("%s Subject = %q, want %q", qaOpenInTerminal, qa.Subject, "repo")
		}

		if len(qa.Command) == 0 || qa.Command[0] != "gnome-terminal" {
			t.Errorf("%s Command[0] = %v, want gnome-terminal", qaOpenInTerminal, qa.Command)
		}

		if len(qa.InitCommands) == 0 || qa.InitCommands[0] != "git status" {
			t.Errorf("%s InitCommands = %v, want [git status]", qaOpenInTerminal, qa.InitCommands)
		}
	}

	if !found {
		t.Errorf("default quick action %s not present", qaOpenInTerminal)
	}
}

func TestQuickActionsForRoot_MergeByName(t *testing.T) {
	cfg := &Config{
		QuickActions: []QuickActionConfig{
			{Name: qaOpenInTerminal, Subject: "repo", Command: []string{"gnome-terminal"}},
			{Name: "list-files", Subject: "repo", Command: []string{"ls"}},
		},
		Roots: []LocalRoot{
			{
				Name: "work",
				Path: "/p/w",
				RootConfig: RootConfig{
					QuickActions: []QuickActionConfig{
						// Override: same name as global open-in-terminal.
						{Name: qaOpenInTerminal, Subject: "repo", Command: []string{"xterm"}},
						// New entry only for this root.
						{Name: "open-editor", Subject: "repo", Command: []string{"nvim"}},
					},
				},
			},
			{
				Name:       "personal",
				Path:       "/p/p",
				RootConfig: RootConfig{}, // no overrides
			},
		},
	}

	// Root 0 (work): override applies + new entry appended.
	merged := cfg.QuickActionsForRoot(0)
	if len(merged) != 3 {
		t.Fatalf("root 0: expected 3 entries, got %d (%v)", len(merged), merged)
	}

	if merged[0].Name != qaOpenInTerminal || merged[0].Command[0] != "xterm" {
		t.Errorf("root 0: open-in-terminal not overridden, got %+v", merged[0])
	}

	if merged[1].Name != "list-files" || merged[1].Command[0] != "ls" {
		t.Errorf("root 0: list-files should be inherited, got %+v", merged[1])
	}

	if merged[2].Name != "open-editor" {
		t.Errorf("root 0: open-editor should be appended, got %+v", merged[2])
	}

	// Root 1 (personal): inherits both globals unchanged.
	merged = cfg.QuickActionsForRoot(1)
	if len(merged) != 2 {
		t.Fatalf("root 1: expected 2 entries, got %d", len(merged))
	}

	if merged[0].Command[0] != "gnome-terminal" {
		t.Errorf("root 1: open-in-terminal not inherited, got %+v", merged[0])
	}

	// Out-of-range root: returns globals.
	merged = cfg.QuickActionsForRoot(99)
	if len(merged) != 2 {
		t.Errorf("out-of-range: expected 2 globals, got %d", len(merged))
	}
}

func TestMergeQuickActions_AddsMissingDefaults(t *testing.T) {
	user := []QuickActionConfig{
		{Name: "user-only", Subject: "repo", Command: []string{"foo"}},
	}
	defaults := []QuickActionConfig{
		{Name: "user-only", Subject: "repo", Command: []string{"DEFAULT-OVERWRITE"}},
		{Name: qaOpenInTerminal, Subject: "repo", Command: []string{"gnome-terminal"}},
	}

	merged := mergeQuickActions(user, defaults)
	if len(merged) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(merged))
	}

	// User entry preserved as-is (default with same name is NOT applied).
	if merged[0].Command[0] != "foo" {
		t.Errorf("user entry overwritten: got %+v", merged[0])
	}

	// Default appended.
	if merged[1].Name != qaOpenInTerminal {
		t.Errorf("default entry not appended: got %+v", merged[1])
	}
}

func TestRoundTrip_QuickActions(t *testing.T) {
	original := &Config{
		QuickActions: []QuickActionConfig{
			{
				Name:        qaOpenInTerminal,
				Subject:     "repo",
				Description: "Open repo in terminal",
				Command:     []string{"gnome-terminal", "--working-directory={{workdir}}"},
			},
		},
	}
	original.AddRoot("work", "/p/w", time.Minute)
	original.Roots[0].RootConfig.QuickActions = []QuickActionConfig{
		{Name: "extra", Subject: "repo", Command: []string{"echo", "hi"}},
	}

	var buf bytes.Buffer
	if err := original.EncodeYAML(&buf); err != nil {
		t.Fatalf("EncodeYAML: %v", err)
	}

	restored := &Config{}
	if _, err := load(fakeFS(buf.String()), "config.yaml", restored); err != nil {
		t.Fatalf("load: %v\nyaml=%s", err, buf.String())
	}

	if len(restored.QuickActions) != 1 || restored.QuickActions[0].Name != qaOpenInTerminal {
		t.Errorf("global quick actions not round-tripped: %+v", restored.QuickActions)
	}

	if len(restored.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(restored.Roots))
	}

	rootQAs := restored.Roots[0].RootConfig.QuickActions
	if len(rootQAs) != 1 || rootQAs[0].Name != "extra" {
		t.Errorf("per-root quick actions not round-tripped: %+v", rootQAs)
	}
}

// fakeFS is a minimal fs.FS that serves a single file from a string.
type fakeFS string

func (f fakeFS) Open(name string) (fs.File, error) {
	return &fakeFile{Reader: bytes.NewReader([]byte(f)), name: name}, nil
}

type fakeFile struct {
	*bytes.Reader

	name string
}

func (f *fakeFile) Stat() (fs.FileInfo, error) {
	return fakeFileInfo{name: f.name, size: int64(f.Len())}, nil
}

func (f *fakeFile) Close() error { return nil }

type fakeFileInfo struct {
	name string
	size int64
}

func (fi fakeFileInfo) Name() string       { return fi.name }
func (fi fakeFileInfo) Size() int64        { return fi.size }
func (fi fakeFileInfo) Mode() fs.FileMode  { return 0o444 }
func (fi fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fi fakeFileInfo) IsDir() bool        { return false }
func (fi fakeFileInfo) Sys() any           { return nil }

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
	cfg.AddRoot("", "/home/dev/projects", 5*time.Minute)
	if cfg.Roots[0].Name != "projects" {
		t.Errorf("AddRoot empty name: got %q, want %q", cfg.Roots[0].Name, "projects")
	}

	// When name is provided, it is used as-is.
	cfg.AddRoot("custom", "/home/dev/work", 5*time.Minute)
	if cfg.Roots[1].Name != "custom" {
		t.Errorf("AddRoot explicit name: got %q, want %q", cfg.Roots[1].Name, "custom")
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

	if !cfg.UpdateRootName(0, "renamed") {
		t.Fatal("UpdateRootName(0) returned false")
	}

	if cfg.Roots[0].Name != "renamed" {
		t.Errorf("Name after update = %q, want %q", cfg.Roots[0].Name, "renamed")
	}

	// Out-of-range returns false.
	if cfg.UpdateRootName(5, "nope") {
		t.Error("UpdateRootName(5) should return false for out-of-range index")
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

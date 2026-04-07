package config

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"go.yaml.in/yaml/v3"
)

//go:embed default_config.yaml
var efs embed.FS

const (
	// configDir is the directory name under os.UserConfigDir.
	configDir = "git-janitor"

	// configFile is the configuration file name.
	configFile = "config.yaml"
)

// Config describes the git-janitor configuration for this local host.
type Config struct {
	Roots    []LocalRoot
	Defaults struct {
		RootConfig *RootConfig
		Rules      *RulesConfig
	}

	// GitHub controls GitHub API integration (requires GITHUB_TOKEN or GH_TOKEN).
	GitHub GitHubConfig
}

// GitHubConfig controls GitHub API integration.
type GitHubConfig struct {
	// Enabled controls whether GitHub API checks are attempted.
	// Default: true.
	Enabled bool

	// SecurityAlerts controls whether security alert APIs are queried.
	// When false, security checks are skipped and the Facts tab shows "not queried".
	// Default: true.
	SecurityAlerts *bool `mapstructure:"securityAlerts,omitempty"`
}

// LocalRoot groups repositories found under a single filesystem root.
type LocalRoot struct {
	Name string
	// Path is the absolute path of the root on this host.
	Path       string
	RootConfig RootConfig

	// Repos is populated at runtime by scanning; it is not persisted.
	Repos []Repository `mapstructure:"-"`
}

// RootConfig holds the configuration for all repositories under a root.
type RootConfig struct {
	// ScheduleInterval is how often the janitor should check this root
	// for stale branches, etc.
	ScheduleInterval time.Duration

	// Rules overrides the default rules for this root.
	// Only the Disable field is used — checks listed here are
	// removed from the defaults.
	Rules *RootRulesOverride `mapstructure:",omitempty"`

	// GitHub overrides the global GitHub config for this root.
	// nil means inherit the global default.
	GitHub *GitHubConfig `mapstructure:",omitempty"`
}

// RulesConfig defines which checks and actions are enabled by default.
type RulesConfig struct {
	// Checks lists the enabled checks with optional parameters.
	Checks []CheckConfig

	// Actions lists the action settings (auto-execute, confirmation, etc.).
	Actions []ActionConfig
}

// CheckConfig configures a single check.
type CheckConfig struct {
	// Name is the registered check name (e.g. "branch-lagging").
	Name string

	// Params holds check-specific configuration overrides.
	// Keys and semantics depend on the check implementation.
	Params map[string]any `mapstructure:",omitempty"`
}

// ActionConfig configures a single action.
type ActionConfig struct {
	// Name is the registered action name (e.g. "run-gc").
	Name string

	// Auto indicates whether the action can be executed without user
	// confirmation. In Phase 1 (UX-driven), non-auto actions show a
	// confirmation popup. In Phase 2, non-auto actions queue as "pending
	// confirmation" rather than executing immediately.
	Auto bool
}

// RootRulesOverride allows per-root customization of the default rules.
type RootRulesOverride struct {
	// Disable lists check names to exclude for this root.
	Disable []string
}

// Repository describes a single git repository discovered on disk.
type Repository struct {
	Path          string
	Kind          RepoKind
	Access        AccessKind
	LastScanned   time.Time
	Remotes       map[string]string
	SCM           SCMKind
	DefaultBranch string
	LastCommit    time.Time
	Activity      float64
}

// RepoKind classifies a repository.
type RepoKind string

const (
	RepoKindNone  RepoKind = "not-git"
	RepoKindClone RepoKind = "clone"
	RepoKindFork  RepoKind = "fork"
	RepoKindOther RepoKind = "other"
)

// AccessKind classifies the access level of a repository.
type AccessKind string

const (
	AccessKindNone    AccessKind = "none"
	AccessKindPublic  AccessKind = "public"
	AccessKindPrivate AccessKind = "private"
)

// SCMKind identifies the source-code management platform hosting the remote.
type SCMKind string

const (
	SCMKindNone   SCMKind = "no-scm"
	SCMKindGithub SCMKind = "github"
	SCMKindGitlab SCMKind = "gitlab"
	SCMKindOther  SCMKind = "other"
)

// EnabledChecks returns the list of check names enabled for the given root,
// after applying the root's disable overrides to the defaults.
// If no rules are configured, returns nil (meaning "run all registered checks").
func (c *Config) EnabledChecks(rootIndex int) []string {
	rules := c.Defaults.Rules
	if rules == nil || len(rules.Checks) == 0 {
		return nil // no config = run all
	}

	// Start with default check names.
	enabled := make([]string, 0, len(rules.Checks))
	for _, ch := range rules.Checks {
		enabled = append(enabled, ch.Name)
	}

	// Apply per-root disable list.
	if rootIndex >= 0 && rootIndex < len(c.Roots) {
		override := c.Roots[rootIndex].RootConfig.Rules
		if override != nil {
			disabled := make(map[string]bool, len(override.Disable))
			for _, name := range override.Disable {
				disabled[name] = true
			}

			filtered := enabled[:0]
			for _, name := range enabled {
				if !disabled[name] {
					filtered = append(filtered, name)
				}
			}

			enabled = filtered
		}
	}

	return enabled
}

// IsActionAuto reports whether the named action is configured for
// auto-execution (no user confirmation needed).
// Returns false if the action is not configured or not set to auto.
func (c *Config) IsActionAuto(name string) bool {
	rules := c.Defaults.Rules
	if rules == nil {
		return false
	}

	for _, a := range rules.Actions {
		if a.Name == name {
			return a.Auto
		}
	}

	return false
}

// GitHubEnabled reports whether GitHub API checks are enabled for the given root.
//
// Per-root override takes precedence over the global default.
// If no override is set, the global GitHub.Enabled value is used.
func (c *Config) GitHubEnabled(rootIndex int) bool {
	if rootIndex >= 0 && rootIndex < len(c.Roots) {
		if override := c.Roots[rootIndex].RootConfig.GitHub; override != nil {
			return override.Enabled
		}
	}

	return c.GitHub.Enabled
}

// GitHubSecurityAlerts reports whether security alert APIs should be queried
// for the given root. Per-root override takes precedence over the global default.
// Default is true when not explicitly set.
func (c *Config) GitHubSecurityAlerts(rootIndex int) bool {
	if rootIndex >= 0 && rootIndex < len(c.Roots) {
		if override := c.Roots[rootIndex].RootConfig.GitHub; override != nil && override.SecurityAlerts != nil {
			return *override.SecurityAlerts
		}
	}

	if c.GitHub.SecurityAlerts != nil {
		return *c.GitHub.SecurityAlerts
	}

	return true // default
}

// DefaultConfigPath returns the full path to the configuration file
// under the user's config directory (e.g. $HOME/.config/git-janitor/config.yaml).
func DefaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determining user config directory: %w", err)
	}

	return filepath.Join(dir, configDir, configFile), nil
}

// Load reads and parses the configuration from the given YAML file path.
//
// Defaults from the embedded default_config.yaml are loaded first,
// then the user file is overlaid on top.
func Load(file string) (*Config, error) {
	cfg, err := loadDefaults()
	if err != nil {
		return nil, fmt.Errorf("loading default config: %w", err)
	}

	fsys := os.DirFS(filepath.Dir(file))
	pth := filepath.Join(".", filepath.Base(file))

	return load(fsys, pth, cfg)
}

// LoadDefault attempts to load the configuration from the default path.
//
// If the file does not exist it returns the embedded defaults and no error.
// Any other I/O or parse error is reported.
//
// The user config is overlaid on top of embedded defaults. Checks and actions
// lists are merged: any default entries not present in the user config are
// appended so that newly added built-in checks are picked up automatically.
func LoadDefault() (*Config, error) {
	defaults, err := loadDefaults()
	if err != nil {
		return nil, fmt.Errorf("loading default config: %w", err)
	}

	path, err := DefaultConfigPath()
	if err != nil {
		return defaults, nil //nolint:nilerr // Cannot determine path — return defaults silently.
	}

	fsys := os.DirFS(filepath.Dir(path))
	pth := filepath.Join(".", filepath.Base(path))

	// Snapshot the default rules before the user config overwrites slices.
	defaultRules := copyRules(defaults.Defaults.Rules)

	result, err := load(fsys, pth, defaults)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return defaults, nil
		}

		return nil, fmt.Errorf("loading config from %s: %w", path, err)
	}

	// Merge: add default checks/actions not present in the user config.
	mergeRules(result, defaultRules)

	return result, nil
}

// copyRules returns a deep copy of a RulesConfig so the original
// is not mutated when mapstructure overwrites slices.
func copyRules(r *RulesConfig) *RulesConfig {
	if r == nil {
		return nil
	}

	cp := &RulesConfig{
		Checks:  make([]CheckConfig, len(r.Checks)),
		Actions: make([]ActionConfig, len(r.Actions)),
	}

	copy(cp.Checks, r.Checks)
	copy(cp.Actions, r.Actions)

	return cp
}

// mergeRules supplements the config's rules with any default checks and actions
// that are not already present. This ensures newly added built-in checks are
// picked up even when the user has an existing config file.
func mergeRules(cfg *Config, defaults *RulesConfig) {
	if defaults == nil || cfg.Defaults.Rules == nil {
		return
	}

	cfg.Defaults.Rules.Checks = mergeChecks(cfg.Defaults.Rules.Checks, defaults.Checks)
	cfg.Defaults.Rules.Actions = mergeActions(cfg.Defaults.Rules.Actions, defaults.Actions)
}

func mergeChecks(user, defaults []CheckConfig) []CheckConfig {
	existing := make(map[string]bool, len(user))
	for _, c := range user {
		existing[c.Name] = true
	}

	for _, c := range defaults {
		if !existing[c.Name] {
			user = append(user, c)
		}
	}

	return user
}

func mergeActions(user, defaults []ActionConfig) []ActionConfig {
	existing := make(map[string]bool, len(user))
	for _, a := range user {
		existing[a.Name] = true
	}

	for _, a := range defaults {
		if !existing[a.Name] {
			user = append(user, a)
		}
	}

	return user
}

// LoadDefaults loads the embedded default_config.yaml into a fresh Config.
func LoadDefaults() (*Config, error) {
	return loadDefaults()
}

func loadDefaults() (*Config, error) {
	return load(efs, "default_config.yaml", &Config{})
}

// load reads a YAML file from fsys, unmarshals it, and decodes into cfg
// using mapstructure with duration/time hooks.
func load(fsys fs.FS, file string, cfg *Config) (*Config, error) {
	content, err := fs.ReadFile(fsys, file)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(time.RFC3339),
		),
		Result: cfg,
	})
	if err != nil {
		return nil, fmt.Errorf("creating decoder: %w", err)
	}

	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	return cfg, nil
}

// IsEmpty reports whether the configuration has no roots defined.
func (c *Config) IsEmpty() bool {
	return len(c.Roots) == 0
}

// AddRoot appends a new root directory to the configuration.
//
// If name is empty it defaults to the base name of path (e.g.
// "/home/dev/projects" → "projects").
func (c *Config) AddRoot(name, path string, interval time.Duration) {
	if name == "" {
		name = filepath.Base(path)
	}

	c.Roots = append(c.Roots, LocalRoot{
		Name: name,
		Path: path,
		RootConfig: RootConfig{
			ScheduleInterval: interval,
		},
	})
}

// UpdateRootName changes the display name of the root at the given index.
//
// It returns false if the index is out of range.
func (c *Config) UpdateRootName(index int, name string) bool {
	if index < 0 || index >= len(c.Roots) {
		return false
	}

	c.Roots[index].Name = name

	return true
}

// RootDisplayName returns the display name for the root at the given index.
//
// If the name is empty, it falls back to the base name of the root's path.
func (c *Config) RootDisplayName(index int) string {
	if index < 0 || index >= len(c.Roots) {
		return ""
	}

	r := c.Roots[index]
	if r.Name != "" {
		return r.Name
	}

	return filepath.Base(r.Path)
}

// UpdateRootPath changes the path of the root at the given index.
//
// It returns false if the index is out of range.
func (c *Config) UpdateRootPath(index int, path string) bool {
	if index < 0 || index >= len(c.Roots) {
		return false
	}

	c.Roots[index].Path = path

	return true
}

// DeleteRoot removes the root at the given index.
//
// It returns false if the index is out of range.
func (c *Config) DeleteRoot(index int) bool {
	if index < 0 || index >= len(c.Roots) {
		return false
	}

	c.Roots = append(c.Roots[:index], c.Roots[index+1:]...)

	return true
}

// UpdateRootInterval changes the schedule interval of the root at the given index.
//
// It returns false if the index is out of range.
func (c *Config) UpdateRootInterval(index int, interval time.Duration) bool {
	if index < 0 || index >= len(c.Roots) {
		return false
	}

	c.Roots[index].RootConfig.ScheduleInterval = interval

	return true
}

// EncodeYAML serializes the configuration as YAML into the provided writer.
//
// Runtime-only fields (Repos) are excluded from the output via mapstructure:"-" tags.
func (c *Config) EncodeYAML(w io.Writer) error {
	var raw map[string]any

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Squash: true,
		Deep:   true,
		Result: &raw,
	})
	if err != nil {
		return fmt.Errorf("creating mapstructure decoder: %w", err)
	}

	if err := dec.Decode(c); err != nil {
		return fmt.Errorf("encoding config to map: %w", err)
	}

	return yaml.NewEncoder(w).Encode(raw)
}

// Save writes the configuration as YAML to the default config path.
//
// It creates the parent directory if it does not exist.
func (c *Config) Save() error {
	path, err := DefaultConfigPath()
	if err != nil {
		return fmt.Errorf("determining config path: %w", err)
	}

	return c.SaveTo(path)
}

// SaveTo writes the configuration as YAML to the given file path.
//
// It creates the parent directory if it does not exist.
func (c *Config) SaveTo(path string) error {
	const readableDirPerm = 0o755
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, readableDirPerm); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating config file %s: %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	if encErr := c.EncodeYAML(f); encErr != nil {
		return fmt.Errorf("writing config to %s: %w", path, encErr)
	}

	return nil
}

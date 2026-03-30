package backend

import (
	"context"
	"strings"
)

// ConfigScope indicates where a git config value is defined.
type ConfigScope string

const (
	ScopeSystem   ConfigScope = "system"
	ScopeGlobal   ConfigScope = "global"
	ScopeLocal    ConfigScope = "local"
	ScopeWorktree ConfigScope = "worktree"
	ScopeCommand  ConfigScope = "command"
	ScopeUnset    ConfigScope = ""
)

// ConfigEntry holds a single git config value with its origin scope.
type ConfigEntry struct {
	// Key is the config key (e.g. "user.email").
	Key string

	// Value is the effective value. Empty if unset.
	Value string

	// Scope indicates where the value is defined (global, local, etc.).
	// Empty (ScopeUnset) if the key is not configured.
	Scope ConfigScope

	// IsLocal reports whether this value is set in the repo's local config
	// (as opposed to inherited from global/system).
	IsLocal bool
}

// RepoConfig holds a curated set of git config values for a repository.
type RepoConfig struct {
	UserEmail  ConfigEntry
	UserName   ConfigEntry
	SigningKey ConfigEntry
	CommitSign ConfigEntry
	TagSign    ConfigEntry
}

// configPattern matches the keys we query in a single git config --get-regexp call.
const configPattern = `^(user\.(email|name|signingkey)|commit\.gpgsign|tag\.gpgsign)$`

// Config queries the curated set of git config values for the repository
// in a single git command, reporting the effective value and whether it's
// local or inherited.
func (r *Runner) Config(ctx context.Context) RepoConfig {
	var rc RepoConfig

	out, err := r.run(ctx, cmdConfigGetRegexp(configPattern)...)
	if err != nil {
		return rc
	}

	// Build a map of key → ConfigEntry from the output.
	// If a key appears at multiple scopes (e.g. global then local),
	// the last one wins (git lists them in precedence order).
	entries := parseConfigEntries(out)

	rc.UserEmail = entries["user.email"]
	rc.UserName = entries["user.name"]
	rc.SigningKey = entries["user.signingkey"]
	rc.CommitSign = entries["commit.gpgsign"]
	rc.TagSign = entries["tag.gpgsign"]

	return rc
}

// parseConfigEntries parses the output of git config --show-scope --get-regexp.
//
// Each line has the format:
//
//	scope\tkey value
//
// If a key appears multiple times (e.g. global + local override), the last
// occurrence wins (highest precedence).
func parseConfigEntries(output string) map[string]ConfigEntry {
	entries := make(map[string]ConfigEntry)

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split: "scope\tkey value"
		scope, rest, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}

		// rest is "key value" — split on first space.
		key, value, _ := strings.Cut(rest, " ")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		s := ConfigScope(strings.TrimSpace(scope))

		entries[key] = ConfigEntry{
			Key:     key,
			Value:   value,
			Scope:   s,
			IsLocal: s == ScopeLocal || s == ScopeWorktree,
		}
	}

	return entries
}

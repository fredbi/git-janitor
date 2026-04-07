// SPDX-License-Identifier: Apache-2.0

package models

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
	Scope ConfigScope

	// IsLocal reports whether this value is set in the repo's local config.
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

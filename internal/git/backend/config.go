package backend

import (
	"context"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// configPattern matches the keys we query in a single git config --get-regexp call.
const configPattern = `^(user\.(email|name|signingkey)|commit\.gpgsign|tag\.gpgsign)$`

// Config queries the curated set of git config values for the repository
// in a single git command, reporting the effective value and whether it's
// local or inherited.
func (r *Runner) Config(ctx context.Context) models.RepoConfig {
	var rc models.RepoConfig

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
func parseConfigEntries(output string) map[string]models.ConfigEntry {
	entries := make(map[string]models.ConfigEntry)

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
		s := models.ConfigScope(strings.TrimSpace(scope))

		entries[key] = models.ConfigEntry{
			Key:     key,
			Value:   value,
			Scope:   s,
			IsLocal: s == models.ScopeLocal || s == models.ScopeWorktree,
		}
	}

	return entries
}

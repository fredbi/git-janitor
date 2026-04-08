// SPDX-License-Identifier: Apache-2.0

// Package prompts builds structured prompts for AI agent actions.
package prompts

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ConflictInfo holds the conflict context provided by the git check.
type ConflictInfo struct {
	Files       []ConflictFile `json:"files"`
	Language    string         `json:"language"`    // "go", "yaml", etc.
	CommitCount int            `json:"commitCount"` // number of commits on the branch
}

// ConflictFile describes a single conflicting file.
type ConflictFile struct {
	Path string `json:"path"`
	Type string `json:"type"` // e.g. "add/add", "content"
}

// ParseConflictInfo parses the JSON-encoded conflict info from action params.
func ParseConflictInfo(raw string) (ConflictInfo, error) {
	var info ConflictInfo
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		return info, fmt.Errorf("parsing conflict info: %w", err)
	}

	return info, nil
}

// EncodeConflictInfo encodes conflict info to JSON for action params.
func EncodeConflictInfo(info ConflictInfo) string {
	data, err := json.Marshal(info)
	if err != nil {
		return "{}"
	}

	return string(data)
}

// BuildResolveConflictsPrompt generates a prompt for the AI agent to resolve
// merge conflicts in a worktree.
func BuildResolveConflictsPrompt(workDir, branchName, targetBranch string, conflicts ConflictInfo) string {
	var b strings.Builder

	b.WriteString("You are resolving merge conflicts in a git repository.\n\n")

	// Context.
	b.WriteString("CONTEXT:\n")
	b.WriteString(fmt.Sprintf("- Repository: %s\n", workDir))
	b.WriteString(fmt.Sprintf("- Branch %q has been squashed and rebased onto %q but has conflicts.\n", branchName, targetBranch))
	b.WriteString("- The working tree contains conflict markers that need resolution.\n\n")

	// Conflicting files.
	b.WriteString("CONFLICTING FILES:\n")

	for _, f := range conflicts.Files {
		if f.Type != "" {
			b.WriteString(fmt.Sprintf("- %s (%s)\n", f.Path, f.Type))
		} else {
			b.WriteString(fmt.Sprintf("- %s\n", f.Path))
		}
	}

	b.WriteString("\n")

	// Strategy.
	b.WriteString("STRATEGY:\n")
	b.WriteString(fmt.Sprintf("- For each conflicting file, resolve the conflict by keeping the intent of\n"))
	b.WriteString(fmt.Sprintf("  the %q branch changes while incorporating the %q updates.\n", branchName, targetBranch))

	// Language-specific hints.
	if conflicts.Language == "go" {
		b.WriteString("- For go.mod: resolve conflicts keeping the higher version of each dependency.\n")
		b.WriteString("  After resolving go.mod, run `go mod tidy` to regenerate go.sum.\n")
		b.WriteString("  Do NOT manually edit go.sum — it is auto-generated.\n")
	}

	b.WriteString("- For YAML/workflow files: prefer the branch version but incorporate\n")
	b.WriteString(fmt.Sprintf("  any new content from %q that doesn't conflict with the branch intent.\n\n", targetBranch))

	// Instructions.
	b.WriteString("INSTRUCTIONS:\n")
	b.WriteString("1. Examine each conflicting file\n")
	b.WriteString("2. Resolve all conflict markers (<<<<<<< ======= >>>>>>>)\n")

	if conflicts.Language == "go" {
		b.WriteString("3. Run `go mod tidy` to regenerate go.sum\n")
		b.WriteString("4. Stage all resolved files: `git add -A`\n")
	} else {
		b.WriteString("3. Stage all resolved files: `git add -A`\n")
	}

	b.WriteString("5. Verify no conflict markers remain:\n")
	b.WriteString("   `grep -r '<<<<<<' . --include='*.go' --include='*.yml' --include='*.yaml' --include='*.md' --include='*.mod'`\n")
	b.WriteString("6. Do NOT commit or push — the caller will handle that.\n")

	return b.String()
}

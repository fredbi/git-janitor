package git

import (
	"bufio"
	"context"
	"fmt"
	"strings"
)

// StatusEntry represents a single entry from git status --porcelain=v2.
type StatusEntry struct {
	// XY is the two-character status code (e.g. "M.", ".M", "A.", "??").
	XY string

	// Path is the file path relative to the repo root.
	Path string

	// OrigPath is set for renames/copies (the source path).
	OrigPath string
}

// IsUntracked reports whether the entry is an untracked file.
func (e StatusEntry) IsUntracked() bool {
	return e.XY == "??"
}

// IsIgnored reports whether the entry is an ignored file.
func (e StatusEntry) IsIgnored() bool {
	return e.XY == "!!"
}

// Status holds the parsed output of git status.
type Status struct {
	// Branch is the current branch name (empty if detached HEAD).
	Branch string

	// OID is the commit hash of HEAD.
	OID string

	// Upstream is the upstream tracking branch (e.g. "origin/main").
	Upstream string

	// AheadBehind holds the ahead/behind counts relative to upstream.
	Ahead  int
	Behind int

	// Entries are the changed/untracked files.
	Entries []StatusEntry
}

// IsDirty reports whether the working tree has any changes.
func (s Status) IsDirty() bool {
	return len(s.Entries) > 0
}

// Status runs git status --porcelain=v2 --branch and parses the output.
func (r *Runner) Status(ctx context.Context) (Status, error) {
	out, err := r.run(ctx, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return Status{}, err
	}

	return parseStatus(out), nil
}

// parseStatus parses the output of git status --porcelain=v2 --branch.
func parseStatus(output string) Status {
	var s Status

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "# branch.oid "):
			s.OID = strings.TrimPrefix(line, "# branch.oid ")

		case strings.HasPrefix(line, "# branch.head "):
			head := strings.TrimPrefix(line, "# branch.head ")
			if head != "(detached)" {
				s.Branch = head
			}

		case strings.HasPrefix(line, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(line, "# branch.upstream ")

		case strings.HasPrefix(line, "# branch.ab "):
			parseAheadBehind(strings.TrimPrefix(line, "# branch.ab "), &s)

		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			// Ordinary (1) or rename/copy (2) changed entry.
			entry := parseChangedEntry(line)
			if entry != nil {
				s.Entries = append(s.Entries, *entry)
			}

		case strings.HasPrefix(line, "? "):
			// Untracked file.
			s.Entries = append(s.Entries, StatusEntry{
				XY:   "??",
				Path: line[2:],
			})

		case strings.HasPrefix(line, "! "):
			// Ignored file.
			s.Entries = append(s.Entries, StatusEntry{
				XY:   "!!",
				Path: line[2:],
			})
		}
	}

	return s
}

// parseAheadBehind parses "+N -M" into Ahead/Behind on a Status.
func parseAheadBehind(s string, st *Status) {
	// Format: "+3 -1"
	var ahead, behind int

	//nolint:errcheck // best-effort parsing
	fmt.Sscanf(s, "+%d -%d", &ahead, &behind)

	st.Ahead = ahead
	st.Behind = behind
}

// parseChangedEntry parses a porcelain v2 ordinary or rename/copy entry.
//
// Format for ordinary (type 1):
//
//	1 XY sub mH mI mW hH hI path
//
// Format for rename/copy (type 2):
//
//	2 XY sub mH mI mW hH hI Xscore path\torigPath
func parseChangedEntry(line string) *StatusEntry {
	// Split into at most 9 or 10 space-delimited fields, keeping the tail intact
	// so that tab-separated paths in rename entries are preserved.
	fields := strings.SplitN(line, " ", 2)
	if len(fields) < 2 {
		return nil
	}

	entryType := fields[0]

	switch entryType {
	case "1":
		// Ordinary entry: 9 space-separated fields (path is field 9).
		parts := strings.SplitN(line, " ", 9)
		if len(parts) < 9 {
			return nil
		}

		return &StatusEntry{
			XY:   parts[1],
			Path: parts[8],
		}

	case "2":
		// Rename/copy entry: 10 space-separated fields (field 9 is Xscore,
		// field 10 is "path\torigPath").
		parts := strings.SplitN(line, " ", 10)
		if len(parts) < 10 {
			return nil
		}

		pathPart := parts[9]
		path, origPath, _ := strings.Cut(pathPart, "\t")

		return &StatusEntry{
			XY:       parts[1],
			Path:     path,
			OrigPath: origPath,
		}
	}

	return nil
}

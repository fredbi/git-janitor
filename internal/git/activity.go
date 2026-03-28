package git

import (
	"bufio"
	"context"
	"strconv"
	"strings"
	"time"
)

// Staleness classification constants.
const (
	StalenessActive  = "active"  // commits in the last 30 days
	StalenessRecent  = "recent"  // commits in the last 90 days
	StalenessStale   = "stale"   // commits in the last 360 days
	StalenessDormant = "dormant" // no commits in the last 360 days
)

// Activity holds commit activity metrics for a repository.
//
// All windows are rolling: 7d, 30d, 90d, 360d from now.
// Counts are on HEAD only (merged activity).
type Activity struct {
	// Commit counts over rolling windows.
	Commits7d   int
	Commits30d  int
	Commits90d  int
	Commits360d int

	// TagsLast360d is the number of tags on the default branch created in the last 360 days.
	// Derived from the tag list — no extra git command.
	TagsLast360d int

	// Staleness is derived from commit counts:
	//   "active"  — Commits30d > 0
	//   "recent"  — Commits90d > 0
	//   "stale"   — Commits360d > 0
	//   "dormant" — otherwise
	Staleness string

	// Authors is populated on-demand via LoadAuthors.
	Authors []AuthorActivity
}

// AuthorActivity holds per-author commit counts.
type AuthorActivity struct {
	Name    string
	Email   string
	Commits int
}

// Activity collects commit activity metrics on HEAD using rolling windows.
func (r *Runner) Activity(ctx context.Context) Activity {
	now := time.Now()

	var a Activity

	a.Commits7d = r.commitCount(ctx, now.AddDate(0, 0, -7))
	a.Commits30d = r.commitCount(ctx, now.AddDate(0, 0, -30))
	a.Commits90d = r.commitCount(ctx, now.AddDate(0, 0, -90))
	a.Commits360d = r.commitCount(ctx, now.AddDate(0, 0, -360))

	a.Staleness = deriveStaleness(a)

	return a
}

// LoadAuthors populates the Authors field with per-author commit counts
// for the given rolling window (e.g. 360 days).
func (r *Runner) LoadAuthors(ctx context.Context, sinceDays int) []AuthorActivity {
	since := time.Now().AddDate(0, 0, -sinceDays)

	out, err := r.run(ctx, cmdShortlog(since.Format("2006-01-02"))...)
	if err != nil {
		return nil
	}

	return parseShortlog(out)
}

// CountTagsInWindow counts how many tags from the given list fall within the last nDays.
func CountTagsInWindow(tags []Tag, nDays int) int {
	cutoff := time.Now().AddDate(0, 0, -nDays)

	var count int

	for i := range tags {
		if tags[i].RemoteOnly {
			continue
		}

		if tags[i].Date.After(cutoff) {
			count++
		}
	}

	return count
}

// commitCount returns the number of commits on HEAD since the given time.
func (r *Runner) commitCount(ctx context.Context, since time.Time) int {
	out, err := r.run(ctx, cmdRevListCount(since.Format("2006-01-02"))...)
	if err != nil {
		return 0
	}

	n, _ := strconv.Atoi(strings.TrimSpace(out)) //nolint:errcheck // best-effort

	return n
}

// deriveStaleness classifies activity based on commit counts.
func deriveStaleness(a Activity) string {
	switch {
	case a.Commits30d > 0:
		return StalenessActive
	case a.Commits90d > 0:
		return StalenessRecent
	case a.Commits360d > 0:
		return StalenessStale
	default:
		return StalenessDormant
	}
}

// parseShortlog parses the output of git shortlog -sne.
// Format: "  <count>\t<Name> <email>"
func parseShortlog(output string) []AuthorActivity {
	var authors []AuthorActivity

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split on tab: "count\tName <email>"
		countStr, rest, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}

		count, err := strconv.Atoi(strings.TrimSpace(countStr))
		if err != nil {
			continue
		}

		name, email := parseNameEmail(rest)

		authors = append(authors, AuthorActivity{
			Name:    name,
			Email:   email,
			Commits: count,
		})
	}

	return authors
}

// parseNameEmail extracts name and email from "Name <email>" format.
func parseNameEmail(s string) (string, string) {
	s = strings.TrimSpace(s)

	open := strings.LastIndex(s, "<")
	close := strings.LastIndex(s, ">")

	if open < 0 || close < 0 || close <= open {
		return s, ""
	}

	return strings.TrimSpace(s[:open]), s[open+1 : close]
}

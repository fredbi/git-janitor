package backend

import (
	"bufio"
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// Activity collects commit activity metrics on HEAD using rolling windows.
func (r *Runner) Activity(ctx context.Context) models.Activity {
	now := time.Now()

	var a models.Activity

	a.Commits7d = r.commitCount(ctx, now.AddDate(0, 0, -7))
	a.Commits30d = r.commitCount(ctx, now.AddDate(0, 0, -30))
	a.Commits90d = r.commitCount(ctx, now.AddDate(0, 0, -90))
	a.Commits360d = r.commitCount(ctx, now.AddDate(0, 0, -360))

	a.Staleness = deriveStaleness(a)

	return a
}

// LoadAuthors populates the Authors field with per-author commit counts
// for the given rolling window (e.g. 360 days).
func (r *Runner) LoadAuthors(ctx context.Context, sinceDays int) []models.AuthorActivity {
	since := time.Now().AddDate(0, 0, -sinceDays)

	out, err := r.run(ctx, cmdShortlog(since.Format("2006-01-02"))...)
	if err != nil {
		return nil
	}

	return parseShortlog(out)
}

// commitCount returns the number of commits on HEAD since the given time.
func (r *Runner) commitCount(ctx context.Context, since time.Time) int {
	out, err := r.run(ctx, cmdRevListCount(since.Format("2006-01-02"))...)
	if err != nil {
		return 0
	}

	n, _ := strconv.Atoi(strings.TrimSpace(out))

	return n
}

// deriveStaleness classifies activity based on commit counts.
func deriveStaleness(a models.Activity) string {
	switch {
	case a.Commits30d > 0:
		return models.StalenessActive
	case a.Commits90d > 0:
		return models.StalenessRecent
	case a.Commits360d > 0:
		return models.StalenessStale
	default:
		return models.StalenessDormant
	}
}

// parseShortlog parses the output of git shortlog -sne.
// Format: "  <count>\t<Name> <email>".
func parseShortlog(output string) []models.AuthorActivity {
	var authors []models.AuthorActivity

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

		authors = append(authors, models.AuthorActivity{
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

	opening := strings.LastIndex(s, "<")
	if opening < 0 {
		return s, ""
	}

	closing := strings.LastIndex(s, ">")
	if closing < 0 || closing <= opening {
		return s, ""
	}

	return strings.TrimSpace(s[:opening]), s[opening+1 : closing]
}

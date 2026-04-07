package backend

import (
	"bufio"
	"context"
	"strings"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

// Stashes runs git stash list and returns all stash entries.
func (r *Runner) Stashes(ctx context.Context) ([]models.Stash, error) {
	out, err := r.run(ctx, cmdStashList()...)
	if err != nil {
		return nil, err
	}

	return parseStashes(out), nil
}

// parseStashes parses the output of git stash list --format="%gD\t%aI\t%gs".
//
// Each line has the form:
//
//	stash@{0}\t2025-04-07T10:30:00+02:00\tOn main: my stash message
//	stash@{1}\t2025-04-06T09:00:00+02:00\tWIP on feature: abc1234 commit message
func parseStashes(output string) []models.Stash {
	var stashes []models.Stash

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		stash := parseStashLine(line)
		if stash != nil {
			stashes = append(stashes, *stash)
		}
	}

	return stashes
}

// parseStashLine parses a single stash list line in tab-separated format.
func parseStashLine(line string) *models.Stash {
	// Expected format: "stash@{N}\tISO-date\tsubject"
	// Subject is either "On <branch>: <message>" or "WIP on <branch>: <hash> <message>"
	parts := strings.SplitN(line, "\t", 3) //nolint:mnd // 3 tab-separated fields

	if len(parts) < 3 { //nolint:mnd // need all 3 fields
		// Fallback: try legacy format (no tabs) for compatibility.
		return parseStashLineLegacy(line)
	}

	ref := parts[0]
	dateStr := parts[1]
	subject := parts[2]

	stash := &models.Stash{Ref: ref}

	// Parse timestamp.
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		stash.LastUpdatedAt = t
	}

	// Parse subject: "On <branch>: <message>" or "WIP on <branch>: <hash> <message>"
	parseStashSubject(stash, subject)

	return stash
}

// parseStashLineLegacy handles the old git stash list format (without --format).
func parseStashLineLegacy(line string) *models.Stash {
	ref, rest, ok := strings.Cut(line, ": ")
	if !ok {
		return nil
	}

	stash := &models.Stash{Ref: ref}
	parseStashSubject(stash, rest)

	return stash
}

// parseStashSubject extracts branch and message from the stash subject string.
func parseStashSubject(stash *models.Stash, subject string) {
	switch {
	case strings.HasPrefix(subject, "On "):
		branchMsg := strings.TrimPrefix(subject, "On ")
		branch, message, _ := strings.Cut(branchMsg, ": ")
		stash.Branch = branch
		stash.Message = message

	case strings.HasPrefix(subject, "WIP on "):
		branchMsg := strings.TrimPrefix(subject, "WIP on ")
		branch, message, _ := strings.Cut(branchMsg, ": ")
		stash.Branch = branch
		stash.Message = "WIP: " + message

	default:
		stash.Message = subject
	}
}

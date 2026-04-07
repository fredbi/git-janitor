package backend

import (
	"bufio"
	"context"
	"strings"

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

// parseStashes parses the output of git stash list.
//
// Each line has the form:
//
//	stash@{0}: On main: my stash message
//	stash@{1}: WIP on feature: abc1234 commit message
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

// parseStashLine parses a single stash list line.
func parseStashLine(line string) *models.Stash {
	// Format: "stash@{N}: On <branch>: <message>"
	// or:     "stash@{N}: WIP on <branch>: <hash> <message>"

	ref, rest, ok := strings.Cut(line, ": ")
	if !ok {
		return nil
	}

	stash := &models.Stash{Ref: ref}

	switch {
	case strings.HasPrefix(rest, "On "):
		// "On <branch>: <message>"
		branchMsg := strings.TrimPrefix(rest, "On ")
		branch, message, _ := strings.Cut(branchMsg, ": ")
		stash.Branch = branch
		stash.Message = message

	case strings.HasPrefix(rest, "WIP on "):
		// "WIP on <branch>: <hash> <message>"
		branchMsg := strings.TrimPrefix(rest, "WIP on ")
		branch, message, _ := strings.Cut(branchMsg, ": ")
		stash.Branch = branch
		stash.Message = "WIP: " + message

	default:
		// Unknown format — store the whole thing as the message.
		stash.Message = rest
	}

	return stash
}

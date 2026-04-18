package backend

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// FetchAll runs git fetch --all to update all remote-tracking branches.
func (r *Runner) FetchAll(ctx context.Context) error {
	_, err := r.run(ctx, cmdFetchAll()...)

	return err
}

// FetchAllTags runs git fetch --all --tags to update all remotes including tags.
func (r *Runner) FetchAllTags(ctx context.Context) error {
	_, err := r.run(ctx, cmdFetchAllTags()...)

	return err
}

// Fetch runs git fetch for a specific remote.
func (r *Runner) Fetch(ctx context.Context, remote string) error {
	_, err := r.run(ctx, cmdFetchRemote(remote)...)

	return err
}

// LastCommitTime returns the author date of the most recent commit on HEAD.
func (r *Runner) LastCommitTime(ctx context.Context) (time.Time, error) {
	out, err := r.run(ctx, cmdLastCommitDate()...)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, strings.TrimSpace(out))
}

// LastCommitMessage returns the subject line of the most recent commit on HEAD.
func (r *Runner) LastCommitMessage(ctx context.Context) string {
	out, err := r.run(ctx, cmdLastCommitMessage()...)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(out)
}

// CommitCount returns the total number of commits reachable from HEAD.
// Callers should skip this on shallow repos — the count would reflect the
// shallow window, not the real history.
func (r *Runner) CommitCount(ctx context.Context) int {
	out, err := r.run(ctx, cmdRevListCountHEAD()...)
	if err != nil {
		return 0
	}

	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0
	}

	return n
}

// FirstCommitTime returns the author date of the earliest commit reachable
// from HEAD (the root commit). When HEAD has multiple root commits (merged
// histories), the earliest date is returned. Callers should skip this on
// shallow repos — the result would be the shallow boundary, not the real
// first commit.
func (r *Runner) FirstCommitTime(ctx context.Context) (time.Time, error) {
	out, err := r.run(ctx, cmdLogRootCommitDates()...)
	if err != nil {
		return time.Time{}, err
	}

	var earliest time.Time

	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		t, err := time.Parse(time.RFC3339, line)
		if err != nil {
			continue
		}

		if earliest.IsZero() || t.Before(earliest) {
			earliest = t
		}
	}

	return earliest, nil
}

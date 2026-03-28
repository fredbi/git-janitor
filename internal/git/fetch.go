package git

import (
	"context"
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

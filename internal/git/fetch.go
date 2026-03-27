package git

import "context"

// FetchAll runs git fetch --all to update all remote-tracking branches.
func (r *Runner) FetchAll(ctx context.Context) error {
	_, err := r.run(ctx, "fetch", "--all")

	return err
}

// Fetch runs git fetch for a specific remote.
func (r *Runner) Fetch(ctx context.Context, remote string) error {
	_, err := r.run(ctx, "fetch", remote)

	return err
}

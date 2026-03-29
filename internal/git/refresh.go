package git

import "context"

// RefreshRepo runs git fetch --all --tags then collects full repo info.
// If the fetch fails (e.g. remote unavailable), local data is still collected
// and the fetch error is recorded in FetchErr.
func RefreshRepo(ctx context.Context, path string) RepoInfo {
	r := NewRunner(path)

	fetchErr := r.FetchAllTags(ctx)

	info := CollectRepoInfo(ctx, r, path)
	info.FetchErr = fetchErr

	return info
}

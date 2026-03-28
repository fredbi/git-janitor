package git

import "context"

// RefreshRepo runs git fetch --all --tags then collects full repo info.
func RefreshRepo(ctx context.Context, path string) RepoInfo {
	r := NewRunner(path)

	if err := r.FetchAllTags(ctx); err != nil {
		return RepoInfo{Path: path, IsGit: true, Err: err}
	}

	return CollectRepoInfo(ctx, r, path)
}

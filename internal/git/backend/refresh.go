package backend

import (
	"context"

	"github.com/fredbi/git-janitor/internal/models"
)

// RefreshRepo runs git fetch --all --tags then collects full repo info.
// If the fetch fails (e.g. remote unavailable), local data is still collected
// and the fetch error is recorded in FetchErr.
func (r *Runner) RefreshRepo(ctx context.Context) *models.RepoInfo {
	fetchErr := r.FetchAllTags(ctx)

	info := r.collectRepoInfo(ctx)
	info.FetchErr = fetchErr

	return info
}

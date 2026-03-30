package engine

/*
// TODO: are these interfaces useful?

// gitCheckEvaluator is implemented by any check that can evaluate git.RepoInfo.
// Both GitCheck and types embedding GitCheck satisfy this via their Evaluate method.
type gitCheckEvaluator interface {
	Evaluate(info *git.RepoInfo) (iter.Seq[models.Alert], error)
}

// githubCheckEvaluator is implemented by any check that can evaluate github.RepoData.
type githubCheckEvaluator interface {
	Evaluate(data *github.RepoData) (iter.Seq[models.Alert], error)
}

// gitActionExecutor is implemented by any action that can execute via git.Runner.
type gitActionExecutor interface {
	Execute(ctx context.Context, r *git.Runner, info *git.RepoInfo, subjects []models.ActionSubject) (models.Result, error)
}

// githubActionExecutor is implemented by any action that can execute via github.Client.
type githubActionExecutor interface {
	Execute(ctx context.Context, client *github.Client, data *github.RepoData, subjects []models.ActionSubject, params []map[string]string) (models.Result, error)
}
*/

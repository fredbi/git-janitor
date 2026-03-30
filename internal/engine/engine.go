package engine

/*
// Engine is the orchestrator that connects configuration rules to check
// evaluation and action execution.
//
// For Phase 1 (manual, UX-driven), it is a thin loop: run matching
// checks, collect alerts, execute actions on user request.
//
// For Phase 2, it will grow into a full scheduler with priority queue,
// rate limiting, and persistence.
type Engine struct {
	options

	History *History
}

// New creates an Engine with empty registries and a default history.
// Use [NewWithBuiltins] from a package that can import the check/action
// packages, or call RegisterAll manually to populate the registries.
func New() *Engine {
	return &Engine{
		Checks:  registries.NewCheckRegistry(),
		Actions: registries.NewActionRegistry(),
		Runners: registries.NewRunnerRegistry(),
		History: NewHistory(500),
	}
}

func (e *Engine) SetConfig(cfg *config.Config) {
	e.cfg = cfg
}

// Evaluate dispatches a check to the appropriate typed Evaluate method
// based on the check's Kind. It uses interface assertions so concrete
// types embedding GitCheck or GitHubmodels.Check are correctly dispatched.
func (e *Engine) Evaluate(ctx context.Context, check ifaces.Check, input ifaces.RepoInfo) (iter.Seq[models.Alert], error) {
	return e.evaluate(ctx, check, input)
}

// EvaluateRepo runs all enabled checks for a git repository and collects
// the resulting alerts. Alerts with SeverityNone are included so the
// caller can determine which checks ran.
//
// The enabledmodels.Checks parameter is the list of check names from config.
// If empty, all registered git checks are run.
func (e *Engine) EvaluateRepo(ctx context.Context,
	info *git.RepoInfo, enabledChecks []string,
) (iter.Seq[models.Alert], error) {
	return e.evaluateRepo(ctx, info, enabledChecks)
}

// EvaluateGitHub runs all enabled GitHub checks for a repository and collects
// the resulting alerts. This parallels [EvaluateRepo] for git checks.
//
// The enabledmodels.Checks parameter is the list of check names from config.
// If empty, or if no github-* checks are in the list, all registered
// GitHub checks are run. This handles the common case where a user's
// config predates the addition of GitHub checks.
func (e *Engine) EvaluateGitHub(ctx context.Context, data *github.RepoData, enabledChecks []string) (iter.Seq[models.Alert], error) {
	return e.evaluateGithub(ctx, data, enabledChecks)
}

// Execute runs a suggested action after validating that the action's
// models.SubjectKind matctx context.Context, runner *git.Runner, info *git.RepoInfo, suggestion models.ActionSuggestion
func (e *Engine) Execute(ctx context.Context, runner *git.Runner, info *git.RepoInfo, suggestion models.ActionSuggestion) (models.Result, error) {
	return e.execute(ctx, runner, info, suggestion)
}

// ExecuteGitHub runs a GitHub ctx context.Context, client *github.Client, data *github.RepoData, suggestion models.ActionSuggestion
func (e *Engine) ExecuteGitHub(ctx context.Context, client *github.Client, data *github.RepoData, suggestion models.ActionSuggestion) (models.Result, error) {
	return e.executeGithub(ctx, client, data, suggestion)
}


func (e *Engine) evaluate(ctx context.Context, check ifaces.Check, input ifaces.RepoInfo) (iter.Seq[models.Alert], error) {
	switch check.Kind() {
	case models.CheckKindGit:
		gc, ok := check.(gitCheckEvaluator)
		if !ok {
			return nil, fmt.Errorf("engine: check %q (kind=git) does not implement gitCheckEvaluator", check.Name())
		}

		info, ok := input.(*git.RepoInfo)
		if !ok {
			return nil, fmt.Errorf("engine: GitCheck %q expects *git.RepoInfo, got %T", check.Name(), input)
		}

		return gc.Evaluate(info)

	case models.CheckKindGitHub:
		gc, ok := check.(githubCheckEvaluator)
		if !ok {
			return nil, fmt.Errorf("engine: check %q (kind=github) does not implement githubCheckEvaluator", check.Name())
		}

		data, ok := input.(*github.RepoData)
		if !ok {
			return nil, fmt.Errorf("engine: GitHubmodels.Check %q expects *github.RepoData, got %T", check.Name(), input)
		}

		return gc.Evaluate(data)

	default:
		return nil, fmt.Errorf("engine: unsupported check kind %d for %q", check.Kind(), check.Name())
	}
}

func (e *Engine) evaluateRepo(_ context.Context, info *git.RepoInfo, enabledChecks []string) (iter.Seq[models.Alert], error) {
	var alerts []models.Alert

	for name, check := range e.Checks.All() {
		// Skip non-git checks.
		if check.Kind() != models.CheckKindGit {
			continue
		}

		// Filter by enabled list (if provided).
		if len(enabledChecks) > 0 && !slices.Contains(enabledChecks, name) {
			continue
		}

		gc, ok := check.(gitCheckEvaluator)
		if !ok {
			continue
		}

		seq, err := gc.Evaluate(info)
		if err != nil {
			alerts = append(alerts, models.Alert{
				CheckName: name,
				Severity:  models.SeverityHigh,
				Summary:   fmt.Sprintf("check %q failed: %v", name, err),
			})

			continue
		}

		if seq == nil {
			alerts = append(alerts, models.Alert{
				CheckName: name,
				Severity:  models.SeverityNone,
			})

			continue
		}

		for alert := range seq {
			alerts = append(alerts, alert)
		}
	}

	// Sort by severity descending (high first).
	slices.SortStableFunc(alerts, func(a, b models.Alert) int {
		return int(b.Severity) - int(a.Severity)
	})

	return slices.Values(alerts), nil
}

func (e *Engine) evaluateGitHub(_ context.Context, data *github.RepoData, enabledChecks []string) (iter.Seq[models.Alert], error) {
	// If the enabled list has no GitHub checks, run all of them.
	hasGitHubChecks := false
	for _, name := range enabledChecks {
		if len(name) > 7 && name[:7] == "github-" {
			hasGitHubChecks = true

			break
		}
	}

	var alerts []models.Alert

	for name, check := range e.Checks.All() {
		if check.Kind() != models.CheckKindGitHub {
			continue
		}

		if hasGitHubChecks && !slices.Contains(enabledChecks, name) {
			continue
		}

		gc, ok := check.(githubCheckEvaluator)
		if !ok {
			continue
		}

		seq, err := gc.Evaluate(data)
		if err != nil {
			alerts = append(alerts, models.Alert{
				CheckName: name,
				Severity:  models.SeverityHigh,
				Summary:   fmt.Sprintf("check %q failed: %v", name, err),
			})

			continue
		}

		if seq == nil {
			alerts = append(alerts, models.Alert{
				CheckName: name,
				Severity:  models.SeverityNone,
			})

			continue
		}

		for alert := range seq {
			alerts = append(alerts, alert)
		}
	}

	slices.SortStableFunc(alerts, func(a, b models.Alert) int {
		return int(b.Severity) - int(a.Severity)
	})

	return slices.Values(alerts), nil
}

func (e *Engine) execute(ctx context.Context, runner *git.Runner, info *git.RepoInfo, suggestion models.ActionSuggestion) (models.Result, error) {
	action, ok := e.Actions.Get(suggestion.ActionName)
	if !ok {
		return models.Result{}, fmt.Errorf("engine: action %q not found in registry", suggestion.ActionName)
	}

	// Validate models.SubjectKind match.
	if action.ApplyTo() != models.SubjectNone && suggestion.SubjectKind != models.SubjectNone &&
		action.ApplyTo() != suggestion.SubjectKind {
		return models.Result{}, fmt.Errorf(
			"engine: action %q applies to %s but suggestion has %s subjects",
			suggestion.ActionName, action.ApplyTo(), suggestion.SubjectKind,
		)
	}

	ga, ok := action.(gitActionExecutor)
	if !ok {
		return models.Result{}, fmt.Errorf("engine: action %q does not implement gitActionExecutor (got %T)", suggestion.ActionName, action)
	}

	return ga.Execute(ctx, runner, info, suggestion.Subjects)
}

func (e *Engine) executeGitHub(ctx context.Context, client *github.Client, data *github.RepoData, suggestion models.ActionSuggestion) (models.Result, error) {
	action, ok := e.Actions.Get(suggestion.ActionName)
	if !ok {
		return models.Result{}, fmt.Errorf("engine: action %q not found in registry", suggestion.ActionName)
	}

	ga, ok := action.(githubActionExecutor)
	if !ok {
		return models.Result{}, fmt.Errorf("engine: action %q does not implement githubActionExecutor (got %T)", suggestion.ActionName, action)
	}

	return ga.Execute(ctx, client, data, suggestion.Subjects, suggestion.Params)
}
*/

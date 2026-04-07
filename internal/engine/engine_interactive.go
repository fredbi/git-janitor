// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/fredbi/git-janitor/internal/config"
	gitbackend "github.com/fredbi/git-janitor/internal/git/backend"
	githubbackend "github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
)

var _ ifaces.Engineer = &Interactive{}

// Interactive engine is an [ifaces.Engineer] that runs checks and actions
// following user interactions.
//
// It is a thin orchestrator: run matching checks, collect alerts,
// execute actions on user request, and record history.
type Interactive struct {
	options

	// githubClient is lazily initialized on first use.
	// nil means not yet initialized (not that GitHub is unavailable).
	githubClient *githubbackend.Client
}

// NewInteractive creates a new Interactive engine.
func NewInteractive(opts ...Option) *Interactive {
	return &Interactive{
		options: optionsWithDefaults(opts),
	}
}

// Evaluate runs all enabled checks against the given repo info and returns
// the resulting alerts sorted by severity (highest first).
//
// Checks that fail produce a high-severity error alert rather than aborting
// the entire evaluation. Checks that return nil produce a SeverityNone
// placeholder so callers can see which checks ran.
func (e *Interactive) Evaluate(
	ctx context.Context, info *models.RepoInfo, _ ...models.EvaluateOption,
) (iter.Seq[models.Alert], error) {
	if info == nil || info.IsEmpty() {
		return slices.Values([]models.Alert{}), nil
	}

	enabledChecks := e.cfg.EnabledChecks(info.RootIndex)

	var alerts []models.Alert

	for name, check := range e.checks.All() {
		if !e.checkEnabled(name, enabledChecks) {
			continue
		}

		// Skip GitHub checks if no platform data is available.
		if check.Kind() == models.CheckKindGitHub && info.Platform == nil {
			continue
		}

		seq, err := check.Evaluate(ctx, info)
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

		alerts = slices.AppendSeq(alerts, seq)
	}

	// Sort by severity descending (highest first).
	slices.SortStableFunc(alerts, func(a, b models.Alert) int {
		return int(b.Severity) - int(a.Severity)
	})

	return slices.Values(alerts), nil
}

// Execute looks up the action by name, validates the subject kind,
// creates the appropriate runner, and dispatches the action.
func (e *Interactive) Execute(ctx context.Context, info *models.RepoInfo, suggestion models.ActionSuggestion) (models.Result, error) {
	action, ok := e.actions.Get(suggestion.ActionName)
	if !ok {
		return models.Result{}, fmt.Errorf("engine: action %q not found in registry", suggestion.ActionName)
	}

	// Validate SubjectKind match.
	if action.ApplyTo() != models.SubjectNone && suggestion.SubjectKind != models.SubjectNone &&
		action.ApplyTo() != suggestion.SubjectKind {
		return models.Result{}, fmt.Errorf(
			"engine: action %q applies to %s but suggestion has %s subjects",
			suggestion.ActionName, action.ApplyTo(), suggestion.SubjectKind,
		)
	}

	// Create the appropriate runner and inject it into context.
	ctx, err := e.withRunnerForAction(ctx, action, info)
	if err != nil {
		return models.Result{}, err
	}

	// Build the parameter list for the action.
	// Actions with ParamPrompt receive per-subject params; others receive subject names.
	subjects := suggestion.SubjectNames()

	var params []string
	if action.ParamPrompt() != "" {
		// Collect params from all subjects (e.g. user-typed description).
		for _, sub := range suggestion.Subjects {
			params = append(params, sub.Params...)
		}
	} else {
		params = subjects
	}

	result, err := action.Execute(ctx, info, params)

	// Record in history regardless of success/failure.
	e.appendHistory(models.HistoryEntry{
		Timestamp:  time.Now(),
		RepoPath:   info.Path,
		ActionName: suggestion.ActionName,
		Subjects:   subjects,
		Params:     params,
		Result:     result,
	})

	return result, err
}

// GetCheck returns a named check from the registry.
func (e *Interactive) GetCheck(name string) (ifaces.Check, bool) { //nolint:ireturn // interface required by ifaces.Engineer
	return e.checks.Get(name)
}

// GetAction returns a named action from the registry.
func (e *Interactive) GetAction(name string) (ifaces.Action, bool) { //nolint:ireturn // interface required by ifaces.Engineer
	return e.actions.Get(name)
}

// Collect gathers or enriches repo info.
//
// When info only has Path set, it performs a git collection.
// When CollectPlatform is requested, it fetches hosting-platform metadata
// (if the repo is on a supported platform and the config allows it).
//
// Results are cached in the persistent store (if configured). Subsequent
// calls for the same repo path return the cached data until the TTL expires.
// Use [models.CollectForceRefresh] to bypass the cache.
func (e *Interactive) Collect(ctx context.Context, info *models.RepoInfo, opts ...models.CollectOption) *models.RepoInfo {
	if info == nil {
		return nil
	}

	hasOpt := makeOptSet(opts)

	// Cache lookup (skip if ForceRefresh or if info already has git data).
	if !hasOpt[models.CollectForceRefresh] && !info.IsGit && info.Path != "" {
		requireFull := !hasOpt[models.CollectFast]
		if cached := e.cacheGet(info.Path, requireFull); cached != nil {
			cached.RootIndex = info.RootIndex

			// Still do platform collection if requested (platform has its own in-memory cache).
			if hasOpt[models.CollectPlatform] {
				return e.collectPlatform(ctx, cached, hasOpt)
			}

			return cached
		}
	}

	// Git collection: when info has no git data yet.
	if !info.IsGit && info.Path != "" {
		runner := gitbackend.NewRunner(info.Path)

		var collected *models.RepoInfo
		if hasOpt[models.CollectFast] {
			collected = runner.CollectRepoInfoFast(ctx)
			collected.CollectLevel = models.CollectLevelFast
		} else {
			collected = runner.CollectRepoInfo(ctx)
			collected.CollectLevel = models.CollectLevelFull
		}

		collected.RootIndex = info.RootIndex
		collected.CollectedAt = time.Now()

		info = collected

		// Cache the result (only if no fatal error).
		if info.Err == nil {
			e.cachePut(info)
		}
	}

	// Platform collection: when explicitly requested.
	if hasOpt[models.CollectPlatform] {
		info = e.collectPlatform(ctx, info, hasOpt)
	}

	return info
}

// Refresh fetches from remotes then re-collects full repo info.
// The result always bypasses and overwrites the cache.
func (e *Interactive) Refresh(ctx context.Context, info *models.RepoInfo) *models.RepoInfo {
	if info == nil || info.Path == "" {
		return info
	}

	runner := gitbackend.NewRunner(info.Path)
	refreshed := runner.RefreshRepo(ctx)
	refreshed.RootIndex = info.RootIndex
	refreshed.CollectedAt = time.Now()
	refreshed.CollectLevel = models.CollectLevelFull

	// Overwrite the cache with fresh data.
	if refreshed.Err == nil {
		e.cachePut(refreshed)
	}

	return refreshed
}

// Reload updates the engine's configuration.
func (e *Interactive) Reload(cfg *config.Config) {
	e.cfg = cfg
}

// collectPlatform fetches hosting-platform metadata if applicable.
func (e *Interactive) collectPlatform(ctx context.Context, info *models.RepoInfo, hasOpt map[models.CollectOption]bool) *models.RepoInfo {
	// Check that the repo is on a supported platform.
	if info.SCM != models.SCMGitHub {
		return info
	}

	// Check per-root config.
	if !e.cfg.GitHubEnabled(info.RootIndex) {
		return info
	}

	// Resolve origin URL.
	originURL := models.OriginFetchURL(info.Remotes)
	if originURL == "" {
		return info
	}

	// Parse owner/repo from URL.
	owner, repo, err := githubbackend.ExtractOwnerRepo(originURL)
	if err != nil {
		return info
	}

	client := e.getGitHubClient()
	if client == nil || !client.Available() {
		return info
	}

	fetchOpts := githubbackend.FetchOptions{
		ForceRefresh:  hasOpt[models.CollectForceRefresh],
		FetchSecurity: e.cfg.GitHubSecurityAlerts(info.RootIndex),
	}

	platform := client.Fetch(ctx, owner, repo, fetchOpts)
	if platform != nil {
		// Inject local default branch for cross-check.
		platform.LocalDefaultBranch = info.DefaultBranch
		info.Platform = platform
	}

	return info
}

// getGitHubClient lazily initializes the GitHub client.
func (e *Interactive) getGitHubClient() *githubbackend.Client {
	if e.githubClient == nil {
		e.githubClient = githubbackend.NewClient()
	}

	return e.githubClient
}

// withRunnerForAction creates a runner appropriate for the action's kind
// and injects it into the context.
func (e *Interactive) withRunnerForAction(ctx context.Context, action ifaces.Action, info *models.RepoInfo) (context.Context, error) {
	switch action.Kind() {
	case models.ActionKindGit:
		runner := gitbackend.NewRunner(info.Path)

		return ifaces.WithRunner(ctx, runner), nil

	case models.ActionKindGitHub:
		client := e.getGitHubClient()
		if client == nil || !client.Available() {
			return ctx, fmt.Errorf("engine: GitHub token not available for action %q", action.Name())
		}

		runner := &githubbackend.Runner{Client: client}

		return ifaces.WithRunner(ctx, runner), nil

	default:
		return ctx, fmt.Errorf("engine: unsupported action kind %d for %q", action.Kind(), action.Name())
	}
}

// checkEnabled reports whether a check should run, based on the enabled list.
func (e *Interactive) checkEnabled(name string, enabledChecks []string) bool {
	if len(enabledChecks) == 0 {
		return true // no config = run all
	}

	return slices.Contains(enabledChecks, name)
}

// makeOptSet builds a fast lookup set from collect options.
func makeOptSet(opts []models.CollectOption) map[models.CollectOption]bool {
	m := make(map[models.CollectOption]bool, len(opts))
	for _, o := range opts {
		m[o] = true
	}

	return m
}

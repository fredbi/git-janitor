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

		// Skip GitHub checks if no platform data is available (neither origin nor upstream).
		if check.Kind() == models.CheckKindGitHub && info.Platform == nil && info.UpstreamPlatform == nil {
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

	// When subjects carry per-subject params, execute once per subject
	// so each gets its own params (e.g. separate WIP branches per stash).
	// Otherwise, execute once with all subject names.
	if hasPerSubjectParams(suggestion) {
		return e.executePerSubject(ctx, info, action, suggestion)
	}

	subjects := suggestion.SubjectNames()
	result, err := action.Execute(ctx, info, subjects)

	// Attach command log from the runner, if available.
	result.CommandLog = extractCommandLog(ctx)

	e.appendHistory(models.HistoryEntry{
		Timestamp:  time.Now(),
		RepoPath:   info.Path,
		ActionName: suggestion.ActionName,
		Subjects:   subjects,
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

// CollectDetails enriches a RepoInfo with on-demand details for the given subjects.
// Only the named subjects are populated; the result is cached.
func (e *Interactive) CollectDetails(ctx context.Context, info *models.RepoInfo, scope models.ActionSuggestion) *models.RepoInfo {
	if info == nil || info.Path == "" {
		return info
	}

	switch scope.SubjectKind {
	case models.SubjectBranch:
		runner := gitbackend.NewRunner(info.Path)
		e.collectBranchDetails(ctx, runner, info, scope)
	case models.SubjectStash:
		runner := gitbackend.NewRunner(info.Path)
		e.collectStashDetails(ctx, runner, info, scope)
	case models.SubjectIssues:
		e.collectIssueList(ctx, info, scope)
	case models.SubjectPullRequests:
		e.collectPullRequestList(ctx, info, scope)
	case models.SubjectWorkflowRuns:
		e.collectWorkflowRunList(ctx, info, scope)
	}

	// Update cache with enriched data.
	info.CollectedAt = time.Now()
	if info.Err == nil {
		e.cachePut(info)
	}

	return info
}

func (e *Interactive) collectBranchDetails(ctx context.Context, runner *gitbackend.Runner, info *models.RepoInfo, scope models.ActionSuggestion) {
	requested := make(map[string]bool, len(scope.Subjects))
	for _, sub := range scope.Subjects {
		requested[sub.Subject] = true
	}

	for i := range info.Branches {
		b := &info.Branches[i]
		if !requested[b.Name] {
			continue
		}

		// Skip if already populated.
		if b.Detail != nil {
			continue
		}

		b.Detail = runner.CollectBranchDetail(ctx, b.Name, info.DefaultBranch)
	}
}

func (e *Interactive) collectStashDetails(ctx context.Context, runner *gitbackend.Runner, info *models.RepoInfo, scope models.ActionSuggestion) {
	requested := make(map[string]bool, len(scope.Subjects))
	for _, sub := range scope.Subjects {
		requested[sub.Subject] = true
	}

	for i := range info.Stashes {
		s := &info.Stashes[i]
		if !requested[s.Ref] {
			continue
		}

		if s.Detail != nil {
			continue
		}

		s.Detail = runner.CollectStashDetail(ctx, s.Ref)
	}
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

// ProviderEnabled reports whether the named provider is available.
// Currently supports "github" (config enabled + token present).
func (e *Interactive) ProviderEnabled(provider string) bool {
	switch provider {
	case "github":
		if !e.cfg.GitHub.Enabled {
			return false
		}

		client := e.getGitHubClient()

		return client != nil && client.Available()
	default:
		return false
	}
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

		// Preserve activity data from the previous Platform if it exists.
		// Activity data is collected separately via CollectDetails and should
		// survive platform metadata refreshes.
		if old := info.Platform; old != nil {
			platform.Issues = old.Issues
			platform.PullRequests = old.PullRequests
			platform.WorkflowRuns = old.WorkflowRuns
		}

		info.Platform = platform
	}

	// Also collect upstream platform data if an upstream remote exists.
	upstreamURL := models.UpstreamFetchURL(info.Remotes)
	if upstreamURL != "" {
		upOwner, upRepo, upErr := githubbackend.ExtractOwnerRepo(upstreamURL)
		if upErr == nil {
			upstream := client.Fetch(ctx, upOwner, upRepo, fetchOpts)
			if upstream != nil {
				upstream.LocalDefaultBranch = info.DefaultBranch
				info.UpstreamPlatform = upstream
			}
		}
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
		runner.StartLogging()

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

// extractCommandLog retrieves the command log from the runner in context, if available.
func extractCommandLog(ctx context.Context) []string {
	r, ok := ifaces.RunnerFromContext(ctx)
	if !ok || r == nil {
		return nil
	}

	if runner, ok := r.(*gitbackend.Runner); ok {
		return runner.Commands()
	}

	return nil
}

// hasPerSubjectParams reports whether any subject in the suggestion carries params.
func hasPerSubjectParams(suggestion models.ActionSuggestion) bool {
	for _, sub := range suggestion.Subjects {
		if len(sub.Params) > 0 {
			return true
		}
	}

	return false
}

// executePerSubject runs the action once per subject, each with its own params.
// Subjects are processed in reverse order so that index-based references
// (like stash@{N}) remain valid as earlier entries are removed.
// Stops on first failure. Returns a combined result message on success.
func (e *Interactive) executePerSubject(
	ctx context.Context,
	info *models.RepoInfo,
	action ifaces.Action,
	suggestion models.ActionSuggestion,
) (models.Result, error) {
	var messages []string

	// Reverse order: process highest indices first so dropping stash@{5}
	// doesn't shift stash@{3} before we process it.
	for i := len(suggestion.Subjects) - 1; i >= 0; i-- {
		sub := suggestion.Subjects[i]
		// Build params: subject name prepended to per-subject params.
		params := append([]string{sub.Subject}, sub.Params...)

		// Snapshot log length to extract only this execution's commands.
		logBefore := len(extractCommandLog(ctx))

		result, err := action.Execute(ctx, info, params)

		if fullLog := extractCommandLog(ctx); len(fullLog) > logBefore {
			result.CommandLog = fullLog[logBefore:]
		}

		// Record each execution in history.
		e.appendHistory(models.HistoryEntry{
			Timestamp:  time.Now(),
			RepoPath:   info.Path,
			ActionName: suggestion.ActionName,
			Subjects:   []string{sub.Subject},
			Params:     sub.Params,
			Result:     result,
		})

		if err != nil {
			return result, err
		}

		if !result.OK {
			return result, nil
		}

		messages = append(messages, result.Message)
	}

	combined := fmt.Sprintf("%d subject(s) processed", len(suggestion.Subjects))
	if len(messages) > 0 {
		combined = messages[len(messages)-1]
	}

	return models.Result{OK: true, Message: combined}, nil
}

// makeOptSet builds a fast lookup set from collect options.
func makeOptSet(opts []models.CollectOption) map[models.CollectOption]bool {
	m := make(map[models.CollectOption]bool, len(opts))
	for _, o := range opts {
		m[o] = true
	}

	return m
}

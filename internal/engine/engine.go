package engine

import (
	"context"
	"fmt"
	"iter"
	"slices"

	"github.com/fredbi/git-janitor/internal/git"
)

// gitCheckEvaluator is implemented by any check that can evaluate git.RepoInfo.
// Both GitCheck and types embedding GitCheck satisfy this via their Evaluate method.
type gitCheckEvaluator interface {
	Evaluate(info *git.RepoInfo) (iter.Seq[Alert], error)
}

// githubCheckEvaluator is implemented by any check that can evaluate GitHubRepoData.
type githubCheckEvaluator interface {
	Evaluate(data *GitHubRepoData) (iter.Seq[Alert], error)
}

// gitActionExecutor is implemented by any action that can execute via git.Runner.
type gitActionExecutor interface {
	Execute(ctx context.Context, r *git.Runner, info *git.RepoInfo, subjects []string) (Result, error)
}

// Engine is the orchestrator that connects configuration rules to check
// evaluation and action execution.
//
// For Phase 1 (manual, UX-driven), it is a thin loop: run matching
// checks, collect alerts, execute actions on user request.
//
// For Phase 2, it will grow into a full scheduler with priority queue,
// rate limiting, and persistence.
type Engine struct {
	Checks  *CheckRegistry
	Actions *ActionRegistry
	History *History
}

// New creates an Engine with empty registries and a default history.
// Use [NewWithBuiltins] from a package that can import the check/action
// packages, or call RegisterAll manually to populate the registries.
func New() *Engine {
	return &Engine{
		Checks:  NewCheckRegistry(),
		Actions: NewActionRegistry(),
		History: NewHistory(500),
	}
}

// Evaluate dispatches a check to the appropriate typed Evaluate method
// based on the check's Kind. It uses interface assertions so concrete
// types embedding GitCheck or GitHubCheck are correctly dispatched.
func (e *Engine) Evaluate(check Check, input RepoInfo) (iter.Seq[Alert], error) {
	switch check.Kind() {
	case CheckKindGit:
		gc, ok := check.(gitCheckEvaluator)
		if !ok {
			return nil, fmt.Errorf("engine: check %q (kind=git) does not implement gitCheckEvaluator", check.Name())
		}

		info, ok := input.(*git.RepoInfo)
		if !ok {
			return nil, fmt.Errorf("engine: GitCheck %q expects *git.RepoInfo, got %T", check.Name(), input)
		}

		return gc.Evaluate(info)

	case CheckKindGitHub:
		gc, ok := check.(githubCheckEvaluator)
		if !ok {
			return nil, fmt.Errorf("engine: check %q (kind=github) does not implement githubCheckEvaluator", check.Name())
		}

		data, ok := input.(*GitHubRepoData)
		if !ok {
			return nil, fmt.Errorf("engine: GitHubCheck %q expects *GitHubRepoData, got %T", check.Name(), input)
		}

		return gc.Evaluate(data)

	default:
		return nil, fmt.Errorf("engine: unsupported check kind %d for %q", check.Kind(), check.Name())
	}
}

// EvaluateRepo runs all enabled checks for a git repository and collects
// the resulting alerts. Alerts with SeverityNone are included so the
// caller can determine which checks ran.
//
// The enabledChecks parameter is the list of check names from config.
// If empty, all registered git checks are run.
func (e *Engine) EvaluateRepo(_ context.Context, info *git.RepoInfo, enabledChecks []string) []Alert {
	var alerts []Alert

	for name, check := range e.Checks.All() {
		// Skip non-git checks.
		if check.Kind() != CheckKindGit {
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
			alerts = append(alerts, Alert{
				CheckName: name,
				Severity:  SeverityHigh,
				Summary:   fmt.Sprintf("check %q failed: %v", name, err),
			})

			continue
		}

		if seq == nil {
			alerts = append(alerts, Alert{
				CheckName: name,
				Severity:  SeverityNone,
			})

			continue
		}

		for alert := range seq {
			alerts = append(alerts, alert)
		}
	}

	// Sort by severity descending (high first).
	slices.SortStableFunc(alerts, func(a, b Alert) int {
		return int(b.Severity) - int(a.Severity)
	})

	return alerts
}

// Execute runs a suggested action after validating that the action's
// SubjectKind matches the suggestion's SubjectKind.
func (e *Engine) Execute(
	ctx context.Context,
	runner *git.Runner,
	info *git.RepoInfo,
	suggestion ActionSuggestion,
) (Result, error) {
	action, ok := e.Actions.Get(suggestion.ActionName)
	if !ok {
		return Result{}, fmt.Errorf("engine: action %q not found in registry", suggestion.ActionName)
	}

	// Validate SubjectKind match.
	if action.ApplyTo() != SubjectNone && suggestion.SubjectKind != SubjectNone &&
		action.ApplyTo() != suggestion.SubjectKind {
		return Result{}, fmt.Errorf(
			"engine: action %q applies to %s but suggestion has %s subjects",
			suggestion.ActionName, action.ApplyTo(), suggestion.SubjectKind,
		)
	}

	ga, ok := action.(gitActionExecutor)
	if !ok {
		return Result{}, fmt.Errorf("engine: action %q does not implement gitActionExecutor (got %T)", suggestion.ActionName, action)
	}

	return ga.Execute(ctx, runner, info, suggestion.Subjects)
}

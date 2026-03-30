// Package engine defines the shared domain types for git-janitor's
// check→alert→action pipeline, and provides the orchestrator that
// connects configuration rules to check evaluation and action execution.
//
// # Domain types
//
// The core types are [Alert], [models.ActionSuggestion], [models.Result], [Assignment],
// and the enums [Severity], [models.SubjectKind], [models.CheckKind], [models.ActionKind].
// These are shared across the checks, actions, and UX packages.
//
// # models.Checks and actions
//
// [models.Check] and [models.Action] are interfaces implemented by provider-specific
// concrete types ([GitCheck], [GitHubmodels.Check], [GitAction], etc.).
// Each provider struct embeds a [describer] for Name/Description and
// provides a typed Evaluate or Execute method.
//
// Concrete check and action implementations live in separate packages
// (internal/checks/git, internal/actions/git, etc.) and register
// themselves into the engine's registries.
//
// # Registries
//
// [models.CheckRegistry] and [models.ActionRegistry] are flat maps keyed by name.
// They serve two purposes: runtime lookup and config-time discovery
// (listing available checks/actions with descriptions for the wizard).
//
// # Engine
//
// [Engine] is the orchestrator. For Phase 1 (manual, UX-driven), it is
// a thin loop: given a RepoInfo and a list of enabled check names from
// config, run all matching checks, collect alerts, and execute actions
// on user request.
//
// For Phase 2 (background scheduling), Engine will grow into a full
// scheduler with priority queue, rate limiting, and persistence.
//
// # History
//
// [History] is an in-memory ring buffer of [HistoryEntry] records,
// tracking executed actions and their results. Phase 2 will persist
// this to a KV store.
package engine

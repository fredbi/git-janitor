// SPDX-License-Identifier: Apache-2.0

// Package engine provides the orchestrator that connects configuration
// to check evaluation, data collection, and action execution.
//
// # Interactive engine
//
// [Interactive] implements [ifaces.Engineer] for Phase 1 (manual, UX-driven).
// It is a thin loop: given a [models.RepoInfo] and the current config,
// run all matching checks, collect alerts, and execute actions on user request.
//
// The engine owns the check and action registries, the configuration,
// and lazily-created backend runners. Callers (the UX layer) interact
// through a pure data-in/data-out interface — no context setup or
// provider-awareness is required.
//
// # History
//
// [History] is an in-memory ring buffer of [HistoryEntry] records,
// tracking executed actions and their results. Phase 2 will persist
// this to a KV store.
package engine

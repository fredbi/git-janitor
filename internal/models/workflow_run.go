// SPDX-License-Identifier: Apache-2.0

package models

import "time"

// WorkflowRun represents a GitHub Actions workflow run summary.
type WorkflowRun struct {
	ID         int64
	Name       string
	Status     string // "queued", "in_progress", "completed"
	Conclusion string // "success", "failure", "cancelled", etc.
	Branch     string
	Event      string // "push", "pull_request", "schedule", etc.
	CreatedAt  time.Time
	HTMLURL    string
	Detail     *WorkflowRunDetail // nil until requested via CollectDetails
}

// WorkflowRunDetail holds on-demand detail information for a workflow run.
type WorkflowRunDetail struct {
	RunNumber  int
	RunAttempt int
	Duration   time.Duration
}

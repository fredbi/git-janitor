// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/agent/prompts"
	"github.com/fredbi/git-janitor/internal/models"
)

// RemoteBranchDiverged detects remote branches on the "upstream" remote
// (the fork) that have diverged from the default branch.
//
// Only suggests rebase for branches where RebaseCheck or MergeCheck confirms
// the operation would succeed, matching the behavior of the local BranchDiverged check.
type RemoteBranchDiverged struct {
	gitCheck
}

func NewRemoteBranchDiverged() RemoteBranchDiverged {
	return RemoteBranchDiverged{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"remote-branch-diverged",
				"detects diverged upstream remote branches that can be rebased or merged",
			),
		},
	}
}

func (c RemoteBranchDiverged) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c RemoteBranchDiverged) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if models.FindRemote(info.Remotes, models.RemoteUpstream) == nil {
		return noAlert(c.Name())
	}

	upstreamPrefix := models.RemoteUpstream + "/"

	// All diverged upstream branches (not merged, not default).
	allDiverged := filterBranches(info, func(b models.Branch) bool {
		if !b.IsRemote || !strings.HasPrefix(b.Name, upstreamPrefix) {
			return false
		}

		branchName := strings.TrimPrefix(b.Name, upstreamPrefix)
		if branchName == info.DefaultBranch {
			return false
		}

		// Not merged AND not just ahead (actually diverged).
		return !b.Merged && !b.AheadOnly
	})

	if len(allDiverged) == 0 {
		return noAlert(c.Name())
	}

	// Filter to those confirmed rebasable.
	rebasable := filterBranches(info, func(b models.Branch) bool {
		if !b.IsRemote || !strings.HasPrefix(b.Name, upstreamPrefix) || b.Merged || b.AheadOnly {
			return false
		}

		branchName := strings.TrimPrefix(b.Name, upstreamPrefix)
		if branchName == info.DefaultBranch {
			return false
		}

		if b.RebaseCheck != nil && (b.RebaseCheck.CanRebase || b.RebaseCheck.CanRebaseSquashed) {
			return true
		}

		// Also accept branches that can be cleanly merged.
		if b.MergeCheck != nil && b.MergeCheck.Clean {
			return true
		}

		return false
	})

	// Split into rebasable (actionable) and stuck (manual attention).
	stuckSet := make(map[string]bool, len(allDiverged))
	for _, s := range allDiverged {
		stuckSet[s.Subject] = true
	}

	for _, s := range rebasable {
		delete(stuckSet, s.Subject)
	}

	var stuck []models.ActionSubject
	for _, s := range allDiverged {
		if stuckSet[s.Subject] {
			stuck = append(stuck, s)
		}
	}

	return func(yield func(models.Alert) bool) {
		// Alert 1: rebasable branches (with action).
		if len(rebasable) > 0 {
			if !yield(models.Alert{
				CheckName: c.Name(),
				Severity:  models.SeverityLow,
				Summary:   fmt.Sprintf("%d upstream branch(es) can be rebased onto %s", len(rebasable), info.DefaultBranch),
				Detail:    subjectsDetail(rebasable),
				Suggestions: []models.ActionSuggestion{{
					ActionName:  "rebase-remote-branch",
					SubjectKind: models.SubjectBranch,
					Subjects:    rebasable,
				}},
			}) {
				return
			}
		}

		// Alert 2: stuck branches — suggest AI agent for conflict resolution.
		if len(stuck) > 0 {
			agentSubjects := buildAgentSubjects(info, stuck)

			alert := models.Alert{
				CheckName: c.Name(),
				Severity:  models.SeverityMedium,
				Summary:   fmt.Sprintf("%d upstream branch(es) diverged with conflicts", len(stuck)),
				Detail:    "These branches cannot be rebased or merged cleanly: " + subjectsDetail(stuck),
			}

			if len(agentSubjects) > 0 {
				alert.Suggestions = []models.ActionSuggestion{{
					ActionName:  "agent-resolve-conflicts",
					SubjectKind: models.SubjectBranch,
					Subjects:    agentSubjects,
				}}
			}

			if !yield(alert) {
				return
			}
		}
	}, nil
}

// buildAgentSubjects creates ActionSubjects with conflict info in Params
// for each stuck branch. The agent action reads these to build its prompt.
func buildAgentSubjects(info *models.RepoInfo, stuck []models.ActionSubject) []models.ActionSubject {
	branchByName := make(map[string]models.Branch, len(info.Branches))
	for _, b := range info.Branches {
		branchByName[b.Name] = b
	}

	// Detect language from go.mod presence.
	language := ""
	for _, b := range info.Branches {
		if !b.IsRemote && b.Name == info.DefaultBranch {
			// Heuristic: if the repo has go.mod, it's a Go project.
			// This is approximate — the actual file check happens at action time.
			language = "go" // TODO: detect more robustly

			break
		}
	}

	var subjects []models.ActionSubject

	for _, s := range stuck {
		b, ok := branchByName[s.Subject]
		if !ok {
			continue
		}

		// Build conflict file list from MergeCheck or RebaseCheck.
		var conflictFiles []prompts.ConflictFile

		if b.MergeCheck != nil {
			for _, f := range b.MergeCheck.Conflicts {
				conflictFiles = append(conflictFiles, prompts.ConflictFile{Path: f})
			}
		}

		if b.RebaseCheck != nil && len(conflictFiles) == 0 {
			for _, f := range b.RebaseCheck.Conflicts {
				conflictFiles = append(conflictFiles, prompts.ConflictFile{Path: f})
			}
		}

		conflictInfo := prompts.ConflictInfo{
			Files:       conflictFiles,
			Language:    language,
			CommitCount: 0, // unknown at check time
		}

		subjects = append(subjects, models.ActionSubject{
			Subject: s.Subject,
			Params:  []string{info.DefaultBranch, prompts.EncodeConflictInfo(conflictInfo)},
		})
	}

	return subjects
}

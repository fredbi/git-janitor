// SPDX-License-Identifier: Apache-2.0

package models

import "testing"

func TestParseSubjectKind(t *testing.T) {
	cases := map[string]struct {
		input string
		want  SubjectKind
		ok    bool
	}{
		"empty":    {"", SubjectNone, true},
		"none":     {"none", SubjectNone, true},
		"repo":     {"repo", SubjectRepo, true},
		"remote":   {"remote", SubjectRemote, true},
		"branch":   {"branch", SubjectBranch, true},
		"stash":    {"stash", SubjectStash, true},
		"tag":      {"tag", SubjectTag, true},
		"file":     {"file", SubjectFile, true},
		"issues":   {"issues", SubjectIssues, true},
		"upper":    {"BRANCH", SubjectBranch, true},
		"trim":     {" stash ", SubjectStash, true},
		"prs":      {"pull-requests", SubjectPullRequests, true},
		"prs_us":   {"pull_requests", SubjectPullRequests, true},
		"workflow": {"workflow_runs", SubjectWorkflowRuns, true},
		"wf_dash":  {"workflow-runs", SubjectWorkflowRuns, true},
		"unknown":  {"banana", 0, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := ParseSubjectKind(tc.input)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

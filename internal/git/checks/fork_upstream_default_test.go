// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestForkUpstreamDefaultBehindLocal(t *testing.T) {
	check := NewForkUpstreamDefaultBehindLocal()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name: "fork lagging — fires medium",
			info: &models.RepoInfo{
				Kind:                       models.RepoKindFork,
				DefaultBranch:              "main",
				UpstreamDefaultBehindLocal: true,
			},
			wantSeverity: models.SeverityMedium,
		},
		{
			name: "fork not lagging — silent",
			info: &models.RepoInfo{
				Kind:                       models.RepoKindFork,
				DefaultBranch:              "main",
				UpstreamDefaultBehindLocal: false,
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "clone kind — silent even if flag set",
			info: &models.RepoInfo{
				Kind:                       models.RepoKindClone,
				DefaultBranch:              "main",
				UpstreamDefaultBehindLocal: true,
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "no default branch — silent",
			info: &models.RepoInfo{
				Kind:                       models.RepoKindFork,
				DefaultBranch:              "",
				UpstreamDefaultBehindLocal: true,
			},
			wantSeverity: models.SeverityNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seq, err := check.Evaluate(context.Background(), tc.info)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got models.Severity

			for a := range seq {
				got = a.Severity
				break
			}

			if got != tc.wantSeverity {
				t.Errorf("severity: got %v, want %v", got, tc.wantSeverity)
			}
		})
	}
}

func TestForkUpstreamDefaultBehindLocalSuggestion(t *testing.T) {
	check := NewForkUpstreamDefaultBehindLocal()
	info := &models.RepoInfo{
		Kind:                       models.RepoKindFork,
		DefaultBranch:              "master",
		UpstreamDefaultBehindLocal: true,
	}

	seq, err := check.Evaluate(context.Background(), info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var alert models.Alert

	for a := range seq {
		alert = a
		break
	}

	if len(alert.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(alert.Suggestions))
	}

	s := alert.Suggestions[0]
	if s.ActionName != "push-local-to-upstream" {
		t.Errorf("action name: got %q, want push-local-to-upstream", s.ActionName)
	}

	if s.SubjectKind != models.SubjectBranch {
		t.Errorf("subject kind: got %v, want SubjectBranch", s.SubjectKind)
	}

	if len(s.Subjects) != 1 || s.Subjects[0].Subject != "master" {
		t.Errorf("subjects: got %v, want [master]", s.Subjects)
	}
}

func TestForkUpstreamDefaultBehindOrigin(t *testing.T) {
	check := NewForkUpstreamDefaultBehindOrigin()

	cases := []struct {
		name         string
		info         *models.RepoInfo
		wantSeverity models.Severity
	}{
		{
			name: "fork upstream behind origin — fires info",
			info: &models.RepoInfo{
				Kind:                        models.RepoKindFork,
				DefaultBranch:               "main",
				UpstreamDefaultBehindOrigin: true,
			},
			wantSeverity: models.SeverityInfo,
		},
		{
			name: "fork upstream in sync with origin — silent",
			info: &models.RepoInfo{
				Kind:                        models.RepoKindFork,
				DefaultBranch:               "main",
				UpstreamDefaultBehindOrigin: false,
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "clone kind — silent even if flag set",
			info: &models.RepoInfo{
				Kind:                        models.RepoKindClone,
				DefaultBranch:               "main",
				UpstreamDefaultBehindOrigin: true,
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "no default branch — silent",
			info: &models.RepoInfo{
				Kind:                        models.RepoKindFork,
				DefaultBranch:               "",
				UpstreamDefaultBehindOrigin: true,
			},
			wantSeverity: models.SeverityNone,
		},
		{
			name: "both flags set — this check still fires independently",
			info: &models.RepoInfo{
				Kind:                        models.RepoKindFork,
				DefaultBranch:               "main",
				UpstreamDefaultBehindLocal:  true,
				UpstreamDefaultBehindOrigin: true,
			},
			wantSeverity: models.SeverityInfo,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seq, err := check.Evaluate(context.Background(), tc.info)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got models.Severity

			for a := range seq {
				got = a.Severity
				break
			}

			if got != tc.wantSeverity {
				t.Errorf("severity: got %v, want %v", got, tc.wantSeverity)
			}
		})
	}
}

func TestForkUpstreamDefaultBehindOriginSuggestion(t *testing.T) {
	check := NewForkUpstreamDefaultBehindOrigin()
	info := &models.RepoInfo{
		Kind:                        models.RepoKindFork,
		DefaultBranch:               "master",
		UpstreamDefaultBehindOrigin: true,
	}

	seq, err := check.Evaluate(context.Background(), info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var alert models.Alert

	for a := range seq {
		alert = a
		break
	}

	if len(alert.Suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(alert.Suggestions))
	}

	s := alert.Suggestions[0]
	if s.ActionName != "push-origin-to-upstream" {
		t.Errorf("action name: got %q, want push-origin-to-upstream", s.ActionName)
	}

	if s.SubjectKind != models.SubjectBranch {
		t.Errorf("subject kind: got %v, want SubjectBranch", s.SubjectKind)
	}

	if len(s.Subjects) != 1 || s.Subjects[0].Subject != "master" {
		t.Errorf("subjects: got %v, want [master]", s.Subjects)
	}
}

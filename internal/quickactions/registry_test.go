// SPDX-License-Identifier: Apache-2.0

package quickactions

import (
	"slices"
	"strings"
	"testing"

	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/models"
)

func TestBuildRegistry_NilConfig(t *testing.T) {
	reg, err := BuildRegistry(nil)
	if err != nil {
		t.Fatalf("BuildRegistry(nil): %v", err)
	}
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
	if reg.Len() != 0 {
		t.Errorf("Len() = %d, want 0", reg.Len())
	}
}

func TestBuildRegistry_NoRoots_ExposesGlobals(t *testing.T) {
	cfg := &config.Config{
		QuickActions: []config.QuickActionConfig{
			{Name: "open-in-terminal", Subject: "repo", Command: []string{"true"}},
		},
	}

	reg, err := BuildRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}

	qa, ok := LookupForRoot(reg, 0, "open-in-terminal")
	if !ok {
		t.Fatal("expected open-in-terminal under root 0")
	}
	if qa.DisplayName() != "open-in-terminal" {
		t.Errorf("DisplayName() = %q", qa.DisplayName())
	}
}

func TestBuildRegistry_PerRootOverride(t *testing.T) {
	cfg := &config.Config{
		QuickActions: []config.QuickActionConfig{
			{Name: "open-in-terminal", Subject: "repo", Command: []string{"gnome-terminal"}},
		},
		Roots: []config.LocalRoot{
			{Name: "work", Path: "/p/w"},
			{
				Name: "personal",
				Path: "/p/p",
				RootConfig: config.RootConfig{
					QuickActions: []config.QuickActionConfig{
						{Name: "open-in-terminal", Subject: "repo", Command: []string{"xterm"}},
						{Name: "edit", Subject: "repo", Command: []string{"nvim"}},
					},
				},
			},
		},
	}
	cfg.SortRoots()

	reg, err := BuildRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}

	personalIdx := slices.IndexFunc(cfg.Roots, func(r config.LocalRoot) bool {
		return r.Name == "personal"
	})
	workIdx := slices.IndexFunc(cfg.Roots, func(r config.LocalRoot) bool {
		return r.Name == "work"
	})

	// Work root inherits the global gnome-terminal entry.
	workQA, ok := LookupForRoot(reg, workIdx, "open-in-terminal")
	if !ok {
		t.Fatal("work: open-in-terminal missing")
	}
	if workQA.Command()[0] != "gnome-terminal" {
		t.Errorf("work: command = %v, want gnome-terminal", workQA.Command())
	}

	// Personal root overrides with xterm and gains an extra entry.
	personalQA, ok := LookupForRoot(reg, personalIdx, "open-in-terminal")
	if !ok {
		t.Fatal("personal: open-in-terminal missing")
	}
	if personalQA.Command()[0] != "xterm" {
		t.Errorf("personal: command = %v, want xterm", personalQA.Command())
	}

	editQA, ok := LookupForRoot(reg, personalIdx, "edit")
	if !ok {
		t.Fatal("personal: edit missing")
	}
	if editQA.Command()[0] != "nvim" {
		t.Errorf("personal: edit command = %v, want nvim", editQA.Command())
	}
}

func TestBuildRegistry_AggregatesProblems(t *testing.T) {
	cfg := &config.Config{
		QuickActions: []config.QuickActionConfig{
			{Name: "ok", Subject: "repo", Command: []string{"true"}},
			{Name: "no-cmd", Subject: "repo", Command: nil},                   // ErrEmptyCommand
			{Name: "bad-subject", Subject: "moon", Command: []string{"true"}}, // ErrUnknownSubject
		},
	}

	reg, err := BuildRegistry(cfg)
	if err == nil {
		t.Fatal("expected aggregated error, got nil")
	}
	if !strings.Contains(err.Error(), "no-cmd") || !strings.Contains(err.Error(), "bad-subject") {
		t.Errorf("error should mention bad entries, got: %v", err)
	}

	// The valid entry is still registered.
	if _, ok := LookupForRoot(reg, 0, "ok"); !ok {
		t.Error("valid entry should be registered despite sibling failures")
	}
}

func TestIterateForRoot_FiltersBySubject(t *testing.T) {
	cfg := &config.Config{
		QuickActions: []config.QuickActionConfig{
			{Name: "open-repo", Subject: "repo", Command: []string{"true"}},
			{Name: "open-branch", Subject: "branch", Command: []string{"true"}},
			{Name: "open-stash", Subject: "stash", Command: []string{"true"}},
		},
	}

	reg, err := BuildRegistry(cfg)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}

	var repoNames []string
	for qa := range IterateForRoot(reg, 0, models.SubjectRepo) {
		repoNames = append(repoNames, qa.DisplayName())
	}

	if len(repoNames) != 1 || repoNames[0] != "open-repo" {
		t.Errorf("repo filter: got %v", repoNames)
	}

	// SubjectNone returns everything.
	var allNames []string
	for qa := range IterateForRoot(reg, 0, models.SubjectNone) {
		allNames = append(allNames, qa.DisplayName())
	}
	if len(allNames) != 3 {
		t.Errorf("none filter: got %d, want 3", len(allNames))
	}
}

func TestIterateForRoot_NilRegistry(t *testing.T) {
	var n int
	for range IterateForRoot(nil, 0, models.SubjectRepo) {
		n++
	}
	if n != 0 {
		t.Errorf("nil registry should yield no entries, got %d", n)
	}
}

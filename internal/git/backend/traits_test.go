package backend

import (
	"context"
	"testing"
)

func TestIntegration_IsShallow(t *testing.T) {
	r := &Runner{Dir: "."}

	shallow := r.IsShallow(context.Background())
	t.Logf("IsShallow=%v", shallow)

	// This repo is not a shallow clone.
	if shallow {
		t.Error("expected IsShallow=false for this repo")
	}
}

func TestIntegration_HasSubmodules(t *testing.T) {
	r := &Runner{Dir: "."}

	has := r.HasSubmodules()
	t.Logf("HasSubmodules=%v", has)

	// This repo does not have submodules.
	if has {
		t.Error("expected HasSubmodules=false for this repo")
	}
}

func TestIntegration_HasLFS(t *testing.T) {
	r := &Runner{Dir: "."}

	has := r.HasLFS()
	t.Logf("HasLFS=%v", has)

	// This repo does not use LFS.
	if has {
		t.Error("expected HasLFS=false for this repo")
	}
}

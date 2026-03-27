package git

import (
	"testing"
)

func TestParseStashes_Standard(t *testing.T) {
	input := "stash@{0}: On main: save work in progress\n" +
		"stash@{1}: WIP on feature: abc1234 half-done refactor\n" +
		"stash@{2}: On develop: quick save\n"

	stashes := parseStashes(input)

	if len(stashes) != 3 {
		t.Fatalf("got %d stashes, want 3", len(stashes))
	}

	// stash@{0}: regular "On" format.
	if stashes[0].Ref != "stash@{0}" {
		t.Errorf("stash[0].Ref = %q", stashes[0].Ref)
	}

	if stashes[0].Branch != "main" {
		t.Errorf("stash[0].Branch = %q, want %q", stashes[0].Branch, "main")
	}

	if stashes[0].Message != "save work in progress" {
		t.Errorf("stash[0].Message = %q", stashes[0].Message)
	}

	// stash@{1}: "WIP on" format.
	if stashes[1].Branch != "feature" {
		t.Errorf("stash[1].Branch = %q, want %q", stashes[1].Branch, "feature")
	}

	if stashes[1].Message != "WIP: abc1234 half-done refactor" {
		t.Errorf("stash[1].Message = %q", stashes[1].Message)
	}
}

func TestParseStashes_Empty(t *testing.T) {
	stashes := parseStashes("")

	if len(stashes) != 0 {
		t.Errorf("got %d stashes, want 0", len(stashes))
	}
}

func TestParseStashLine_OnFormat(t *testing.T) {
	s := parseStashLine("stash@{0}: On main: my message")
	if s == nil {
		t.Fatal("parseStashLine returned nil")
	}

	if s.Ref != "stash@{0}" {
		t.Errorf("Ref = %q", s.Ref)
	}

	if s.Branch != "main" {
		t.Errorf("Branch = %q", s.Branch)
	}

	if s.Message != "my message" {
		t.Errorf("Message = %q", s.Message)
	}
}

func TestParseStashLine_WIPFormat(t *testing.T) {
	s := parseStashLine("stash@{3}: WIP on develop: abc123 some commit")
	if s == nil {
		t.Fatal("parseStashLine returned nil")
	}

	if s.Branch != "develop" {
		t.Errorf("Branch = %q", s.Branch)
	}

	if s.Message != "WIP: abc123 some commit" {
		t.Errorf("Message = %q", s.Message)
	}
}

func TestParseStashLine_UnknownFormat(t *testing.T) {
	s := parseStashLine("stash@{0}: something unusual here")
	if s == nil {
		t.Fatal("parseStashLine returned nil")
	}

	if s.Message != "something unusual here" {
		t.Errorf("Message = %q", s.Message)
	}
}

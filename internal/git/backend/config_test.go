package backend

import (
	"context"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestParseConfigEntries(t *testing.T) {
	input := "global\tuser.name Frédéric BIDON\n" +
		"global\tuser.email fred@example.com\n" +
		"local\tuser.email local@override.com\n" +
		"global\tcommit.gpgsign true\n"

	entries := parseConfigEntries(input)

	// user.email should be the local override (last wins).
	email := entries["user.email"]
	if email.Value != "local@override.com" {
		t.Errorf("user.email = %q, want local@override.com", email.Value)
	}

	if email.Scope != models.ScopeLocal {
		t.Errorf("user.email scope = %q, want local", email.Scope)
	}

	if !email.IsLocal {
		t.Error("user.email should be IsLocal=true")
	}

	// user.name should be global.
	name := entries["user.name"]
	if name.Value != "Frédéric BIDON" {
		t.Errorf("user.name = %q, want Frédéric BIDON", name.Value)
	}

	if name.IsLocal {
		t.Error("user.name should be IsLocal=false")
	}

	// Unqueried key should be absent.
	if _, found := entries["tag.gpgsign"]; found {
		t.Error("tag.gpgsign should not be in the result")
	}
}

func TestIntegration_Config(t *testing.T) {
	r := &Runner{Dir: "."}

	cfg := r.Config(context.Background())

	entries := []struct {
		name  string
		entry models.ConfigEntry
	}{
		{"user.email", cfg.UserEmail},
		{"user.name", cfg.UserName},
		{"user.signingkey", cfg.SigningKey},
		{"commit.gpgsign", cfg.CommitSign},
		{"tag.gpgsign", cfg.TagSign},
	}

	for _, e := range entries {
		if e.entry.Value != "" {
			t.Logf("%-20s = %-30s scope=%-8s local=%v", e.name, e.entry.Value, e.entry.Scope, e.entry.IsLocal)
		} else {
			t.Logf("%-20s (unset)", e.name)
		}
	}

	// At minimum, user.email should be configured.
	if cfg.UserEmail.Value == "" {
		t.Log("warning: user.email is not configured")
	}
}

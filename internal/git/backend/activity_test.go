package backend

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestDeriveStaleness(t *testing.T) {
	tests := []struct {
		name string
		a    models.Activity
		want string
	}{
		{"active", models.Activity{Commits30d: 5}, models.StalenessActive},
		{"recent", models.Activity{Commits90d: 3}, models.StalenessRecent},
		{"stale", models.Activity{Commits360d: 1}, models.StalenessStale},
		{"dormant", models.Activity{}, models.StalenessDormant},
		// 30d > 0 wins even if others are set.
		{"active wins", models.Activity{Commits7d: 2, Commits30d: 10, Commits90d: 20, Commits360d: 50}, models.StalenessActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveStaleness(tt.a)
			if got != tt.want {
				t.Errorf("deriveStaleness() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCountTagsInWindow(t *testing.T) {
	now := time.Now()

	tags := []models.Tag{
		{Name: "v1.0.0", Date: now.AddDate(0, 0, -10)},                   // 10 days ago — in window
		{Name: "v0.9.0", Date: now.AddDate(0, 0, -100)},                  // 100 days ago — in window
		{Name: "v0.1.0", Date: now.AddDate(-2, 0, 0)},                    // 2 years ago — out
		{Name: "v0.8.0", Date: now.AddDate(0, 0, -50), RemoteOnly: true}, // remote-only — skipped
	}

	got := models.CountTagsInWindow(tags, 360)
	if got != 2 {
		t.Errorf("models.CountTagsInWindow(360) = %d, want 2", got)
	}

	got = models.CountTagsInWindow(tags, 30)
	if got != 1 {
		t.Errorf("models.CountTagsInWindow(30) = %d, want 1", got)
	}
}

func TestParseShortlog(t *testing.T) {
	input := "    42\tAlice Smith <alice@example.com>\n" +
		"    17\tBob Jones <bob@example.com>\n" +
		"     3\tCharlie <charlie@example.com>\n"

	authors := parseShortlog(input)

	if len(authors) != 3 {
		t.Fatalf("got %d authors, want 3", len(authors))
	}

	if authors[0].Name != "Alice Smith" || authors[0].Email != "alice@example.com" || authors[0].Commits != 42 {
		t.Errorf("author[0] = %+v", authors[0])
	}

	if authors[1].Commits != 17 {
		t.Errorf("author[1].Commits = %d, want 17", authors[1].Commits)
	}
}

func TestParseNameEmail(t *testing.T) {
	tests := []struct {
		input     string
		wantName  string
		wantEmail string
	}{
		{"Alice <alice@example.com>", "Alice", "alice@example.com"},
		{"Bob Smith <bob@co.uk>", "Bob Smith", "bob@co.uk"},
		{"noangle", "noangle", ""},
	}

	for _, tt := range tests {
		name, email := parseNameEmail(tt.input)
		if name != tt.wantName || email != tt.wantEmail {
			t.Errorf("parseNameEmail(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, email, tt.wantName, tt.wantEmail)
		}
	}
}

func TestIntegration_Activity(t *testing.T) {
	// Run on this repo itself.
	repoPath := "/home/fred/src/github.com/fredbi/git-janitor"
	if _, err := os.Stat(repoPath + "/.git"); err != nil {
		t.Skipf("repo not available: %s", repoPath)
	}

	r := &Runner{Dir: repoPath}
	ctx := context.Background()

	a := r.Activity(ctx)

	t.Logf("commits: 7d=%d 30d=%d 90d=%d 360d=%d staleness=%s",
		a.Commits7d, a.Commits30d, a.Commits90d, a.Commits360d, a.Staleness)

	// This repo should have some commits in the last 360 days.
	if a.Commits360d == 0 {
		t.Error("expected some commits in the last 360 days")
	}

	if a.Staleness == "" {
		t.Error("staleness should not be empty")
	}
}

func TestIntegration_LoadAuthors(t *testing.T) {
	repoPath := "/home/fred/src/github.com/fredbi/git-janitor"
	if _, err := os.Stat(repoPath + "/.git"); err != nil {
		t.Skipf("repo not available: %s", repoPath)
	}

	r := &Runner{Dir: repoPath}
	ctx := context.Background()

	authors := r.LoadAuthors(ctx, 360)

	t.Logf("authors (360d): %d", len(authors))

	for _, a := range authors {
		t.Logf("  %s <%s>: %d commits", a.Name, a.Email, a.Commits)
	}
}

package git

import (
	"testing"
)

func TestParseRemotes_Standard(t *testing.T) {
	input := "origin\thttps://github.com/user/repo.git (fetch)\n" +
		"origin\thttps://github.com/user/repo.git (push)\n" +
		"upstream\thttps://github.com/org/repo.git (fetch)\n" +
		"upstream\tgit@github.com:org/repo.git (push)\n"

	remotes := parseRemotes(input)

	if len(remotes) != 2 {
		t.Fatalf("got %d remotes, want 2", len(remotes))
	}

	if remotes[0].Name != "origin" {
		t.Errorf("remotes[0].Name = %q, want %q", remotes[0].Name, "origin")
	}

	if remotes[0].FetchURL != "https://github.com/user/repo.git" {
		t.Errorf("remotes[0].FetchURL = %q, want https://github.com/user/repo.git", remotes[0].FetchURL)
	}

	if remotes[0].PushURL != "https://github.com/user/repo.git" {
		t.Errorf("remotes[0].PushURL = %q, want https://github.com/user/repo.git", remotes[0].PushURL)
	}

	if remotes[1].Name != "upstream" {
		t.Errorf("remotes[1].Name = %q, want %q", remotes[1].Name, "upstream")
	}

	if remotes[1].FetchURL != "https://github.com/org/repo.git" {
		t.Errorf("remotes[1].FetchURL = %q", remotes[1].FetchURL)
	}

	if remotes[1].PushURL != "git@github.com:org/repo.git" {
		t.Errorf("remotes[1].PushURL = %q", remotes[1].PushURL)
	}
}

func TestParseRemotes_Empty(t *testing.T) {
	remotes := parseRemotes("")

	if len(remotes) != 0 {
		t.Errorf("got %d remotes, want 0", len(remotes))
	}
}

func TestParseRemotes_SSH(t *testing.T) {
	input := "origin\tgit@github.com:user/repo.git (fetch)\n" +
		"origin\tgit@github.com:user/repo.git (push)\n"

	remotes := parseRemotes(input)

	if len(remotes) != 1 {
		t.Fatalf("got %d remotes, want 1", len(remotes))
	}

	if remotes[0].FetchURL != "git@github.com:user/repo.git" {
		t.Errorf("FetchURL = %q", remotes[0].FetchURL)
	}
}

func TestParseRemotes_PreservesOrder(t *testing.T) {
	input := "beta\thttps://example.com/beta.git (fetch)\n" +
		"beta\thttps://example.com/beta.git (push)\n" +
		"alpha\thttps://example.com/alpha.git (fetch)\n" +
		"alpha\thttps://example.com/alpha.git (push)\n"

	remotes := parseRemotes(input)

	if len(remotes) != 2 {
		t.Fatalf("got %d remotes, want 2", len(remotes))
	}

	// Order should match input, not alphabetical.
	if remotes[0].Name != "beta" || remotes[1].Name != "alpha" {
		t.Errorf("order not preserved: got [%s, %s]", remotes[0].Name, remotes[1].Name)
	}
}

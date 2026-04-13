package backend

import (
	"regexp"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestDeriveSCM_BuiltInHeuristics(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want models.RepoSCM
	}{
		{"github.com HTTPS", "https://github.com/user/repo.git", models.SCMGitHub},
		{"github.com SSH", "git@github.com:user/repo.git", models.SCMGitHub},
		{"github enterprise", "https://github.mycorp.com/org/repo.git", models.SCMGitHub},
		{"gitlab.com HTTPS", "https://gitlab.com/group/project.git", models.SCMGitLab},
		{"gitlab.com SSH", "git@gitlab.com:group/project.git", models.SCMGitLab},
		{"gitlab enterprise", "https://gitlab.mycorp.com/group/sub/project.git", models.SCMGitLab},
		{"unknown host", "https://bitbucket.org/user/repo.git", models.SCMOther},
		{"no host match", "https://scm-premium.saas.cagip.group.gca/group/project.git", models.SCMOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remotes := []models.Remote{
				{Name: "origin", FetchURL: tt.url},
			}
			got := DeriveSCM(remotes, nil)
			if got != tt.want {
				t.Errorf("DeriveSCM(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestDeriveSCM_CustomOverrides(t *testing.T) {
	overrides := []SCMOverride{
		{Pattern: regexp.MustCompile(`scm-premium\.saas\.cagip`), SCM: models.SCMGitLab},
		{Pattern: regexp.MustCompile(`git\.mycorp\.com`), SCM: models.SCMGitHub},
	}

	tests := []struct {
		name string
		url  string
		want models.RepoSCM
	}{
		{
			"enterprise gitlab matched by override",
			"https://scm-premium.saas.cagip.group.gca/group/project.git",
			models.SCMGitLab,
		},
		{
			"enterprise gitlab SSH matched by override",
			"git@scm-premium.saas.cagip.group.gca:group/sub/project.git",
			models.SCMGitLab,
		},
		{
			"enterprise github matched by override",
			"https://git.mycorp.com/org/repo.git",
			models.SCMGitHub,
		},
		{
			"override takes precedence over built-in",
			"https://gitlab.com/group/project.git",
			models.SCMGitLab, // still matched by built-in
		},
		{
			"unmatched host falls through to other",
			"https://bitbucket.org/user/repo.git",
			models.SCMOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remotes := []models.Remote{
				{Name: "origin", FetchURL: tt.url},
			}
			got := DeriveSCM(remotes, overrides)
			if got != tt.want {
				t.Errorf("DeriveSCM(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestDeriveSCM_OverrideTakesPrecedenceOverBuiltIn(t *testing.T) {
	// Override that reclassifies a "gitlab.*" host as GitHub (unusual but tests precedence).
	overrides := []SCMOverride{
		{Pattern: regexp.MustCompile(`gitlab\.example\.com`), SCM: models.SCMGitHub},
	}

	remotes := []models.Remote{
		{Name: "origin", FetchURL: "https://gitlab.example.com/group/project.git"},
	}

	got := DeriveSCM(remotes, overrides)
	if got != models.SCMGitHub {
		t.Errorf("override should take precedence: got %q, want %q", got, models.SCMGitHub)
	}
}

func TestDeriveSCM_NoRemotes(t *testing.T) {
	got := DeriveSCM(nil, nil)
	if got != models.SCMOther {
		t.Errorf("no remotes: got %q, want %q", got, models.SCMOther)
	}
}

func TestCompileSCMOverrides(t *testing.T) {
	t.Run("valid patterns", func(t *testing.T) {
		patterns := map[string]string{
			"gitlab": `scm-premium\.saas`,
			"github": `git\.mycorp`,
		}

		overrides, err := CompileSCMOverrides(patterns)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(overrides) != 2 {
			t.Fatalf("got %d overrides, want 2", len(overrides))
		}
	})

	t.Run("unknown SCM name", func(t *testing.T) {
		patterns := map[string]string{
			"bitbucket": `bitbucket\.org`,
		}

		_, err := CompileSCMOverrides(patterns)
		if err == nil {
			t.Fatal("expected error for unknown SCM name")
		}
	})

	t.Run("invalid regex", func(t *testing.T) {
		patterns := map[string]string{
			"gitlab": `[invalid`,
		}

		_, err := CompileSCMOverrides(patterns)
		if err == nil {
			t.Fatal("expected error for invalid regex")
		}
	})

	t.Run("nil map", func(t *testing.T) {
		overrides, err := CompileSCMOverrides(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if overrides != nil {
			t.Fatalf("expected nil overrides for nil input, got %v", overrides)
		}
	})
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/repo.git", "github.com"},
		{"git@github.com:user/repo.git", "github.com"},
		{"ssh://git@gitlab.example.com/group/project.git", "gitlab.example.com"},
		{"https://scm-premium.saas.cagip.group.gca/group/project.git", "scm-premium.saas.cagip.group.gca"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := ExtractHost(tt.url)
			if got != tt.want {
				t.Errorf("ExtractHost(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"testing"
)

func TestExtractOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "SSH with .git suffix",
			url:       "git@github.com:fredbi/git-janitor.git",
			wantOwner: "fredbi",
			wantRepo:  "git-janitor",
		},
		{
			name:      "SSH without .git suffix",
			url:       "git@github.com:fredbi/git-janitor",
			wantOwner: "fredbi",
			wantRepo:  "git-janitor",
		},
		{
			name:      "HTTPS with .git suffix",
			url:       "https://github.com/go-openapi/runtime.git",
			wantOwner: "go-openapi",
			wantRepo:  "runtime",
		},
		{
			name:      "HTTPS without .git suffix",
			url:       "https://github.com/go-openapi/runtime",
			wantOwner: "go-openapi",
			wantRepo:  "runtime",
		},
		{
			name:      "ssh:// scheme",
			url:       "ssh://git@github.com/google/go-github.git",
			wantOwner: "google",
			wantRepo:  "go-github",
		},
		{
			name:      "org with dashes",
			url:       "git@github.com:my-org/my-repo.git",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
		},
		{
			name:      "not a GitHub URL (gitlab SSH)",
			url:       "git@gitlab.com:user/repo.git",
			wantOwner: "user",
			wantRepo:  "repo",
		},
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
		},
		{
			name:    "bare path",
			url:     "/tmp/repo",
			wantErr: true,
		},
		{
			name:    "owner only",
			url:     "https://github.com/fredbi",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ExtractOwnerRepo(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got owner=%q repo=%q", owner, repo)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if owner != tt.wantOwner {
				t.Errorf("owner: got %q, want %q", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("repo: got %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

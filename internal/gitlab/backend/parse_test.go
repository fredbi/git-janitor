// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"testing"
)

func TestExtractProjectPath(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "SSH with .git suffix, 2 segments",
			url:  "git@gitlab.com:mygroup/myproject.git",
			want: "mygroup/myproject",
		},
		{
			name: "SSH without .git suffix",
			url:  "git@gitlab.com:mygroup/myproject",
			want: "mygroup/myproject",
		},
		{
			name: "SSH with nested groups",
			url:  "git@gitlab.ca.cib:group/subgroup/project.git",
			want: "group/subgroup/project",
		},
		{
			name: "SSH with deeply nested groups",
			url:  "git@gitlab.example.com:a/b/c/d/project.git",
			want: "a/b/c/d/project",
		},
		{
			name: "HTTPS with .git suffix",
			url:  "https://gitlab.com/mygroup/myproject.git",
			want: "mygroup/myproject",
		},
		{
			name: "HTTPS without .git suffix",
			url:  "https://gitlab.com/mygroup/myproject",
			want: "mygroup/myproject",
		},
		{
			name: "HTTPS with nested groups",
			url:  "https://gitlab.ca.cib/group/subgroup/project.git",
			want: "group/subgroup/project",
		},
		{
			name: "ssh:// scheme with nested groups",
			url:  "ssh://git@gitlab.example.com/group/subgroup/project.git",
			want: "group/subgroup/project",
		},
		{
			name: "HTTPS self-hosted",
			url:  "https://code.internal.corp/team/service.git",
			want: "team/service",
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
			name:    "single segment (no namespace)",
			url:     "https://gitlab.com/project-only",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractProjectPath(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "SSH SCP-style",
			url:  "git@gitlab.ca.cib:group/project.git",
			want: "https://gitlab.ca.cib",
		},
		{
			name: "HTTPS",
			url:  "https://gitlab.example.com/group/project.git",
			want: "https://gitlab.example.com",
		},
		{
			name: "HTTPS with port",
			url:  "https://gitlab.example.com:8443/group/project.git",
			want: "https://gitlab.example.com:8443",
		},
		{
			name: "ssh:// scheme",
			url:  "ssh://git@gitlab.example.com/group/project.git",
			want: "https://gitlab.example.com",
		},
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBaseURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOwnerAndRepo(t *testing.T) {
	tests := []struct {
		path      string
		wantOwner string
		wantRepo  string
	}{
		{"group/project", "group", "project"},
		{"group/subgroup/project", "group/subgroup", "project"},
		{"a/b/c/d/project", "a/b/c/d", "project"},
		{"project", "", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			owner, repo := OwnerAndRepo(tt.path)
			if owner != tt.wantOwner {
				t.Errorf("owner: got %q, want %q", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("repo: got %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

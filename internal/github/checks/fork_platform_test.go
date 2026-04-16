// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

// Remote convention: origin = source/canonical repo, upstream = user's fork.
func TestForkPlatform(t *testing.T) {
	userFork := &models.PlatformInfo{FullName: "fredbi/testify", IsFork: true}
	intermediate := &models.PlatformInfo{FullName: "go-openapi/testify", IsFork: true}
	canonical := &models.PlatformInfo{FullName: "stretchr/testify", IsFork: false}

	tests := []struct {
		name string
		info *models.RepoInfo
		want *models.PlatformInfo
	}{
		{
			name: "standard: origin=source, upstream=user-fork → return upstream",
			info: &models.RepoInfo{Platform: canonical, UpstreamPlatform: userFork},
			want: userFork,
		},
		{
			name: "fork of a fork: origin=intermediate-fork, upstream=user-fork → return upstream",
			info: &models.RepoInfo{Platform: intermediate, UpstreamPlatform: userFork},
			want: userFork,
		},
		{
			name: "no upstream, origin is user's fork → return origin",
			info: &models.RepoInfo{Platform: userFork},
			want: userFork,
		},
		{
			name: "no upstream, origin is canonical → nil",
			info: &models.RepoInfo{Platform: canonical},
			want: nil,
		},
		{
			name: "upstream present but not a fork → nil (do not fall back to origin)",
			info: &models.RepoInfo{Platform: userFork, UpstreamPlatform: canonical},
			want: nil,
		},
		{
			name: "nothing populated → nil",
			info: &models.RepoInfo{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := forkPlatform(tt.info)
			if got != tt.want {
				t.Errorf("forkPlatform() = %v, want %v", got, tt.want)
			}
		})
	}
}

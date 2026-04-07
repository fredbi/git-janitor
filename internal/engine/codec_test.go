// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestCodec_RoundTrip_Basic(t *testing.T) {
	now := time.Now()

	info := &models.RepoInfo{
		RootIndex:     2,
		Path:          "/home/user/repos/myrepo",
		IsGit:         true,
		CollectedAt:   now,
		CollectLevel:  models.CollectLevelFull,
		DefaultBranch: "main",
		SCM:           models.SCMGitHub,
		Kind:          models.RepoKindClone,
		LastCommit:    now.Add(-time.Hour),
		Branches: []models.Branch{
			{
				Name:       "main",
				IsCurrent:  true,
				Upstream:   "origin/main",
				Ahead:      1,
				Behind:     2,
				LastCommit: now.Add(-time.Hour),
				Hash:       "abc123",
			},
		},
		Remotes: []models.Remote{
			{Name: "origin", FetchURL: "https://github.com/user/repo.git", PushURL: "https://github.com/user/repo.git"},
		},
		Stashes: []models.Stash{
			{Ref: "stash@{0}", Branch: "main", Message: "WIP"},
		},
	}

	data, err := marshalRepoInfo(info)
	if err != nil {
		t.Fatalf("marshalRepoInfo: %v", err)
	}

	got, err := unmarshalRepoInfo(data)
	if err != nil {
		t.Fatalf("unmarshalRepoInfo: %v", err)
	}

	if got.Path != info.Path {
		t.Errorf("Path = %q, want %q", got.Path, info.Path)
	}

	if got.IsGit != info.IsGit {
		t.Errorf("IsGit = %v, want %v", got.IsGit, info.IsGit)
	}

	if got.RootIndex != info.RootIndex {
		t.Errorf("RootIndex = %d, want %d", got.RootIndex, info.RootIndex)
	}

	if got.CollectLevel != info.CollectLevel {
		t.Errorf("CollectLevel = %d, want %d", got.CollectLevel, info.CollectLevel)
	}

	if got.DefaultBranch != info.DefaultBranch {
		t.Errorf("DefaultBranch = %q, want %q", got.DefaultBranch, info.DefaultBranch)
	}

	// Verify time round-trips with nanosecond precision.
	if !got.CollectedAt.Equal(info.CollectedAt) {
		t.Errorf("CollectedAt = %v, want %v", got.CollectedAt, info.CollectedAt)
	}

	if !got.LastCommit.Equal(info.LastCommit) {
		t.Errorf("LastCommit = %v, want %v", got.LastCommit, info.LastCommit)
	}

	if len(got.Branches) != 1 {
		t.Fatalf("Branches len = %d, want 1", len(got.Branches))
	}

	if got.Branches[0].Name != "main" {
		t.Errorf("Branch[0].Name = %q, want %q", got.Branches[0].Name, "main")
	}

	if got.Branches[0].Ahead != 1 {
		t.Errorf("Branch[0].Ahead = %d, want 1", got.Branches[0].Ahead)
	}

	if len(got.Remotes) != 1 {
		t.Fatalf("Remotes len = %d, want 1", len(got.Remotes))
	}

	if got.Remotes[0].FetchURL != info.Remotes[0].FetchURL {
		t.Errorf("Remote[0].FetchURL = %q, want %q", got.Remotes[0].FetchURL, info.Remotes[0].FetchURL)
	}
}

func TestCodec_ErrorsAreZeroed(t *testing.T) {
	info := &models.RepoInfo{
		Path:     "/repo",
		IsGit:    true,
		Err:      errors.New("fatal: not a git repo"),
		FetchErr: errors.New("fetch failed"),
	}

	data, err := marshalRepoInfo(info)
	if err != nil {
		t.Fatalf("marshalRepoInfo: %v", err)
	}

	got, err := unmarshalRepoInfo(data)
	if err != nil {
		t.Fatalf("unmarshalRepoInfo: %v", err)
	}

	// Errors are intentionally zeroed — they are transient state.
	if got.Err != nil {
		t.Errorf("Err should be nil after round-trip, got %v", got.Err)
	}

	if got.FetchErr != nil {
		t.Errorf("FetchErr should be nil after round-trip, got %v", got.FetchErr)
	}
}

func TestCodec_RoundTrip_WithPlatformInfo(t *testing.T) {
	now := time.Now()

	info := &models.RepoInfo{
		Path:  "/repo",
		IsGit: true,
		Platform: &models.PlatformInfo{
			Owner:       "user",
			Repo:        "myrepo",
			FullName:    "user/myrepo",
			HTMLURL:     "https://github.com/user/myrepo",
			Description: "A test repo",
			IsFork:      true,
			StarCount:   42,
			CreatedAt:   now.Add(-time.Hour * 24 * 365),
			UpdatedAt:   now,
		},
	}

	data, err := marshalRepoInfo(info)
	if err != nil {
		t.Fatalf("marshalRepoInfo: %v", err)
	}

	got, err := unmarshalRepoInfo(data)
	if err != nil {
		t.Fatalf("unmarshalRepoInfo: %v", err)
	}

	if got.Platform == nil {
		t.Fatal("Platform should be non-nil after round-trip")
	}

	if got.Platform.Owner != "user" {
		t.Errorf("Platform.Owner = %q, want %q", got.Platform.Owner, "user")
	}

	if got.Platform.StarCount != 42 {
		t.Errorf("Platform.StarCount = %d, want 42", got.Platform.StarCount)
	}

	if !got.Platform.IsFork {
		t.Error("Platform.IsFork should be true")
	}
}

func TestCodec_PlatformErrorZeroed(t *testing.T) {
	info := &models.RepoInfo{
		Path:  "/repo",
		IsGit: true,
		Platform: &models.PlatformInfo{
			Owner: "user",
			Repo:  "myrepo",
			Err:   errors.New("rate limited"),
		},
	}

	data, err := marshalRepoInfo(info)
	if err != nil {
		t.Fatalf("marshalRepoInfo: %v", err)
	}

	got, err := unmarshalRepoInfo(data)
	if err != nil {
		t.Fatalf("unmarshalRepoInfo: %v", err)
	}

	if got.Platform == nil {
		t.Fatal("Platform should be non-nil")
	}

	// Platform.Err is intentionally zeroed.
	if got.Platform.Err != nil {
		t.Errorf("Platform.Err should be nil after round-trip, got %v", got.Platform.Err)
	}
}

func TestCodec_RoundTrip_WithHealthAndSize(t *testing.T) {
	info := &models.RepoInfo{
		Path:  "/repo",
		IsGit: true,
		Health: &models.HealthReport{
			OK:            true,
			LooseObjects:  10,
			PackedObjects: 500,
			GCAdvised:     true,
			GCReasons:     []string{"too many loose objects"},
		},
		Size: &models.RepoSize{
			GitDirBytes: 1024 * 1024,
		},
	}

	data, err := marshalRepoInfo(info)
	if err != nil {
		t.Fatalf("marshalRepoInfo: %v", err)
	}

	got, err := unmarshalRepoInfo(data)
	if err != nil {
		t.Fatalf("unmarshalRepoInfo: %v", err)
	}

	if got.Health == nil {
		t.Fatal("Health should be non-nil")
	}

	if !got.Health.GCAdvised {
		t.Error("Health.GCAdvised should be true")
	}

	if got.Health.LooseObjects != 10 {
		t.Errorf("Health.LooseObjects = %d, want 10", got.Health.LooseObjects)
	}

	if got.Size == nil {
		t.Fatal("Size should be non-nil")
	}

	if got.Size.GitDirBytes != 1024*1024 {
		t.Errorf("Size.GitDirBytes = %d, want %d", got.Size.GitDirBytes, 1024*1024)
	}
}

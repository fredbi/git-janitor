package backend

import (
	"context"
	"os"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestParseTags(t *testing.T) {
	input := "tag|v1.2.3|aaa1111|bbb2222|2026-01-15T10:00:00+00:00|Release 1.2.3|signed\n" +
		"commit|v1.2.4-beta.1|ccc3333||2026-02-01T12:00:00+00:00||unsigned\n" +
		"commit|1.0.0|ddd4444||2025-06-15T08:00:00+00:00|First release|unsigned\n" +
		"commit|not-semver|eee5555||2024-03-01T00:00:00+00:00|Random tag|unsigned\n"

	tags := parseTags(input)

	if len(tags) != 4 {
		t.Fatalf("got %d tags, want 4", len(tags))
	}

	// v1.2.3: annotated, signed, semver with v prefix.
	v123 := tags[0]
	if !v123.Annotated {
		t.Error("v1.2.3 should be annotated")
	}

	if !v123.Signed {
		t.Error("v1.2.3 should be signed")
	}

	if !v123.IsSemver || !v123.HasVPrefix {
		t.Error("v1.2.3 should be semver with v prefix")
	}

	if v123.SemverMajor != 1 || v123.SemverMinor != 2 || v123.SemverPatch != 3 {
		t.Errorf("v1.2.3 version = %d.%d.%d", v123.SemverMajor, v123.SemverMinor, v123.SemverPatch)
	}

	if v123.TargetHash != "bbb2222" {
		t.Errorf("v1.2.3 TargetHash = %q, want bbb2222", v123.TargetHash)
	}

	if v123.IsPrerelease {
		t.Error("v1.2.3 should not be prerelease")
	}

	// v1.2.4-beta.1: lightweight, prerelease.
	beta := tags[1]
	if beta.Annotated {
		t.Error("v1.2.4-beta.1 should be lightweight")
	}

	if !beta.IsSemver || !beta.IsPrerelease {
		t.Error("v1.2.4-beta.1 should be semver prerelease")
	}

	if beta.SemverPrerelease != "beta.1" {
		t.Errorf("prerelease = %q, want beta.1", beta.SemverPrerelease)
	}

	// 1.0.0: semver without v prefix.
	v100 := tags[2]
	if !v100.IsSemver || v100.HasVPrefix {
		t.Error("1.0.0 should be semver without v prefix")
	}

	// not-semver: not semver.
	ns := tags[3]
	if ns.IsSemver {
		t.Error("not-semver should not be semver")
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v1.0.1", -1},
		{"v1.0.1", "v1.0.0", 1},
		{"v1.0.0", "v1.0.0", 0},
		{"v1.2.3", "v2.0.0", -1},
		{"v1.9.0", "v1.10.0", -1},
		{"v1.2.3-beta.1", "v1.2.3", -1},     // prerelease < release
		{"v1.2.3", "v1.2.3-beta.1", 1},      // release > prerelease
		{"v1.2.3-alpha", "v1.2.3-beta", -1}, // alpha < beta
		{"v1.2.3-beta.1", "v1.2.3-beta.2", -1},
	}

	for _, tt := range tests {
		a := models.Tag{Name: tt.a}
		parseSemver(&a)
		b := models.Tag{Name: tt.b}
		parseSemver(&b)

		got := models.CompareSemver(a, b)
		if got != tt.want {
			t.Errorf("models.CompareSemver(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDeriveTagSummary(t *testing.T) {
	tags := parseTags(
		"commit|v2.0.0|aaa||2026-01-01T00:00:00+00:00||unsigned\n" +
			"commit|v1.9.0|bbb||2025-12-01T00:00:00+00:00||unsigned\n" +
			"commit|v2.1.0-rc.1|ccc||2026-03-01T00:00:00+00:00||unsigned\n" +
			"commit|old-tag|ddd||2020-01-01T00:00:00+00:00||unsigned\n",
	)

	lastDate, lastSemver, lastSemverDate := models.DeriveTagSummary(tags)

	// Most recent date is v2.1.0-rc.1 (March 2026).
	if lastDate.Year() != 2026 || lastDate.Month() != 3 {
		t.Errorf("lastTagDate = %v, want 2026-03", lastDate)
	}

	// Highest semver is v2.1.0-rc.1 (2.1.0 prerelease > 2.0.0).
	if lastSemver != "v2.1.0-rc.1" {
		t.Errorf("lastSemverTag = %q, want v2.1.0-rc.1", lastSemver)
	}

	if lastSemverDate.Year() != 2026 || lastSemverDate.Month() != 3 {
		t.Errorf("lastSemverDate = %v, want 2026-03", lastSemverDate)
	}
}

func TestIntegration_Tags(t *testing.T) {
	// Test on go-swagger if available (has real tags).
	repoPath := "/home/fred/src/github.com/go-swagger/go-swagger"
	if _, err := os.Stat(repoPath); err != nil {
		t.Skipf("external repo not available: %s", repoPath)
	}

	r := &Runner{Dir: repoPath}
	ctx := context.Background()

	defaultBranch, _ := r.DefaultBranch(ctx)
	t.Logf("default branch: %s", defaultBranch)

	tags, err := r.Tags(ctx, defaultBranch)
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}

	t.Logf("total tags: %d", len(tags))

	var semverCount, annotatedCount, signedCount, localOnlyCount, remoteOnlyCount int

	for _, tg := range tags {
		if tg.IsSemver {
			semverCount++
		}

		if tg.Annotated {
			annotatedCount++
		}

		if tg.Signed {
			signedCount++
		}

		if tg.LocalOnly {
			localOnlyCount++
		}

		if tg.RemoteOnly {
			remoteOnlyCount++
		}
	}

	t.Logf("semver=%d annotated=%d signed=%d localOnly=%d remoteOnly=%d",
		semverCount, annotatedCount, signedCount, localOnlyCount, remoteOnlyCount)

	lastDate, lastSemver, lastSemverDate := models.DeriveTagSummary(tags)
	t.Logf("lastTagDate=%s lastSemverTag=%s lastSemverDate=%s",
		lastDate.Format("2006-01-02"), lastSemver, lastSemverDate.Format("2006-01-02"))
}

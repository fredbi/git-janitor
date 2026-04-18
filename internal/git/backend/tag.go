package backend

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fredbi/git-janitor/internal/models"
)

var (
	// semverWithV matches "v1.2.3" or "v1.2.3-pre.1" or "v1.2.3+build".
	semverWithV = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.]+))?(?:\+[a-zA-Z0-9.]+)?$`)

	// semverWithoutV matches "1.2.3" or "1.2.3-pre.1".
	semverWithoutV = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.]+))?(?:\+[a-zA-Z0-9.]+)?$`)
)

// LatestTagSummary returns (LastTagDate, LastSemverTag, LastSemverDate)
// using only a single cheap `git tag -l` call.
//
// Unlike [Runner.Tags] it skips per-tag reachability checks (one git
// subprocess per tag) and the `ls-remote` sync check (a network-bound
// call), so it is safe to use in the fast-path collection. Zero values
// are returned when the repository has no tags or the command fails.
func (r *Runner) LatestTagSummary(ctx context.Context) (time.Time, string, time.Time) {
	out, err := r.run(ctx, cmdTagList()...)
	if err != nil {
		return time.Time{}, "", time.Time{}
	}

	return models.DeriveTagSummary(parseTags(out))
}

// Tags lists all tags with metadata.
//
// It fetches tag info in a single git command, then enriches with:
//   - semver parsing
//   - default branch reachability
//   - local vs remote sync status
func (r *Runner) Tags(ctx context.Context, defaultBranch string) ([]models.Tag, error) {
	// Get all tags with metadata in one command.
	out, err := r.run(ctx, cmdTagList()...)
	if err != nil {
		return nil, err
	}

	tags := parseTags(out)

	// Enrich: default branch reachability.
	if defaultBranch != "" {
		r.markTagsOnDefaultBranch(ctx, tags, defaultBranch)
	}

	// Enrich: local vs remote sync.
	tags = r.markTagSyncStatus(ctx, tags)

	return tags, nil
}

// parseTags parses the output of git tag -l --format.
func parseTags(output string) []models.Tag {
	var tags []models.Tag

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 7)
		if len(parts) < 7 {
			continue
		}

		objType := parts[0]
		name := parts[1]
		hash := parts[2]
		deref := parts[3] // dereferenced commit hash (empty for lightweight)
		dateStr := parts[4]
		message := parts[5]
		signedStr := parts[6]

		t := models.Tag{
			Name:      name,
			Hash:      hash,
			Message:   message,
			Annotated: objType == "tag",
			Signed:    signedStr == "signed",
		}

		// Target hash: for annotated tags, deref points to the commit.
		if deref != "" {
			t.TargetHash = deref
		} else {
			t.TargetHash = hash
		}

		// Parse date.
		t.Date, _ = time.Parse(time.RFC3339, strings.TrimSpace(dateStr))

		// Parse semver.
		parseSemver(&t)

		tags = append(tags, t)
	}

	return tags
}

// parseSemver extracts semver components from the tag name.
func parseSemver(t *models.Tag) {
	var matches []string

	if m := semverWithV.FindStringSubmatch(t.Name); m != nil {
		matches = m[1:]
		t.HasVPrefix = true
	} else if m := semverWithoutV.FindStringSubmatch(t.Name); m != nil {
		matches = m[1:]
	}

	if matches == nil {
		return
	}

	t.IsSemver = true

	// regex guarantees digits
	_, _ = fmt.Sscanf(matches[0], "%d", &t.SemverMajor)
	_, _ = fmt.Sscanf(matches[1], "%d", &t.SemverMinor)
	_, _ = fmt.Sscanf(matches[2], "%d", &t.SemverPatch)

	if len(matches) > 3 && matches[3] != "" {
		t.IsPrerelease = true
		t.SemverPrerelease = matches[3]
	}
}

// markTagsOnDefaultBranch checks which tags are reachable from the default branch.
func (r *Runner) markTagsOnDefaultBranch(ctx context.Context, tags []models.Tag, defaultBranch string) {
	for i := range tags {
		_, err := r.run(ctx, cmdIsAncestor(tags[i].TargetHash, defaultBranch)...)
		tags[i].OnDefaultBranch = err == nil
	}
}

// markTagSyncStatus compares local tags against origin remote tags.
// Returns the enriched slice (may include appended remote-only tags).
func (r *Runner) markTagSyncStatus(ctx context.Context, tags []models.Tag) []models.Tag {
	remoteTags := r.remoteTagSet(ctx)
	if remoteTags == nil {
		return tags
	}

	localSet := make(map[string]bool, len(tags))
	for i := range tags {
		localSet[tags[i].Name] = true

		if !remoteTags[tags[i].Name] {
			tags[i].LocalOnly = true
		}
	}

	// Find remote-only tags and append them.
	for name := range remoteTags {
		if !localSet[name] {
			tags = append(tags, models.Tag{
				Name:       name,
				RemoteOnly: true,
			})
		}
	}

	return tags
}

// remoteTagSet fetches the set of tag names from the origin remote.
func (r *Runner) remoteTagSet(ctx context.Context) map[string]bool {
	out, err := r.run(ctx, cmdLsRemoteTags("origin")...)
	if err != nil {
		return nil
	}

	tags := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()

		// Skip dereferenced entries (hash\trefs/tags/name^{}).
		if strings.HasSuffix(line, "^{}") {
			continue
		}

		// Format: "hash\trefs/tags/name"
		_, ref, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}

		name := strings.TrimPrefix(ref, "refs/tags/")
		tags[name] = true
	}

	return tags
}

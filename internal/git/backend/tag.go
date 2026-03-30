package backend

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Tag represents a git tag with metadata.
type Tag struct {
	// Name is the tag name (e.g. "v1.2.3").
	Name string

	// Hash is the object hash the tag points to.
	// For annotated tags, this is the tag object hash.
	Hash string

	// TargetHash is the commit hash the tag ultimately points to.
	// For lightweight tags, same as Hash. For annotated tags, the dereferenced commit.
	TargetHash string

	// Date is the tagger date (annotated) or commit date (lightweight).
	Date time.Time

	// Message is the tag message (empty for lightweight tags).
	Message string

	// Annotated is true for annotated tags (objecttype == "tag").
	Annotated bool

	// Signed is true if the tag has a GPG/SSH signature.
	Signed bool

	// IsSemver is true if the tag matches semver pattern (with or without v prefix).
	IsSemver bool

	// HasVPrefix is true if the semver tag starts with "v" (e.g. "v1.2.3").
	HasVPrefix bool

	// IsPrerelease is true if the semver tag has a prerelease suffix (e.g. "v1.2.3-beta.1").
	IsPrerelease bool

	// OnDefaultBranch is true if the tagged commit is reachable from the default branch.
	OnDefaultBranch bool

	// LocalOnly is true if the tag exists locally but not on the origin remote.
	LocalOnly bool

	// RemoteOnly is true if the tag exists on the origin remote but not locally.
	RemoteOnly bool

	// SemverMajor, SemverMinor, SemverPatch hold the parsed version components.
	SemverMajor int
	SemverMinor int
	SemverPatch int

	// SemverPrerelease holds the prerelease suffix (e.g. "beta.1").
	SemverPrerelease string
}

var (
	// semverWithV matches "v1.2.3" or "v1.2.3-pre.1" or "v1.2.3+build".
	semverWithV = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.]+))?(?:\+[a-zA-Z0-9.]+)?$`)

	// semverWithoutV matches "1.2.3" or "1.2.3-pre.1".
	semverWithoutV = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.]+))?(?:\+[a-zA-Z0-9.]+)?$`)
)

// Tags lists all tags with metadata.
//
// It fetches tag info in a single git command, then enriches with:
//   - semver parsing
//   - default branch reachability
//   - local vs remote sync status
func (r *Runner) Tags(ctx context.Context, defaultBranch string) ([]Tag, error) {
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
func parseTags(output string) []Tag {
	var tags []Tag

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

		t := Tag{
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
func parseSemver(t *Tag) {
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
func (r *Runner) markTagsOnDefaultBranch(ctx context.Context, tags []Tag, defaultBranch string) {
	for i := range tags {
		_, err := r.run(ctx, cmdIsAncestor(tags[i].TargetHash, defaultBranch)...)
		tags[i].OnDefaultBranch = err == nil
	}
}

// markTagSyncStatus compares local tags against origin remote tags.
// Returns the enriched slice (may include appended remote-only tags).
func (r *Runner) markTagSyncStatus(ctx context.Context, tags []Tag) []Tag {
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
			tags = append(tags, Tag{
				Name:       name,
				RemoteOnly: true,
			})
		}
	}

	return tags
}

// CompareSemver returns:
//
//	-1 if a < b
//	 0 if a == b
//	+1 if a > b
//
// Comparison is major, then minor, then patch. Prerelease tags sort
// before the corresponding release (1.2.3-beta < 1.2.3).
func CompareSemver(a, b Tag) int {
	if a.SemverMajor != b.SemverMajor {
		return cmpInt(a.SemverMajor, b.SemverMajor)
	}

	if a.SemverMinor != b.SemverMinor {
		return cmpInt(a.SemverMinor, b.SemverMinor)
	}

	if a.SemverPatch != b.SemverPatch {
		return cmpInt(a.SemverPatch, b.SemverPatch)
	}

	// Prerelease sorts before release.
	if a.IsPrerelease && !b.IsPrerelease {
		return -1
	}

	if !a.IsPrerelease && b.IsPrerelease {
		return 1
	}

	// Both prerelease or both release: compare prerelease strings lexicographically.
	if a.SemverPrerelease < b.SemverPrerelease {
		return -1
	}

	if a.SemverPrerelease > b.SemverPrerelease {
		return 1
	}

	return 0
}

func cmpInt(a, b int) int {
	if a < b {
		return -1
	}

	return 1
}

// DeriveTagSummary computes LastTagDate, LastSemverTag, and LastSemverDate from a tag list.
func DeriveTagSummary(tags []Tag) (lastTagDate time.Time, lastSemverTag string, lastSemverDate time.Time) {
	var bestSemver *Tag

	for i := range tags {
		t := &tags[i]

		// Skip remote-only tags (we don't have their date).
		if t.RemoteOnly {
			continue
		}

		// Last tag by date (any tag).
		if t.Date.After(lastTagDate) {
			lastTagDate = t.Date
		}

		// Last semver tag by version ordering (non-prerelease preferred).
		if !t.IsSemver {
			continue
		}

		if bestSemver == nil || CompareSemver(*t, *bestSemver) > 0 {
			bestSemver = t
		}
	}

	if bestSemver != nil {
		lastSemverTag = bestSemver.Name
		lastSemverDate = bestSemver.Date
	}

	return lastTagDate, lastSemverTag, lastSemverDate
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

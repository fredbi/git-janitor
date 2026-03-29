package gitchecks

import (
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckLargeFiles detects files in HEAD exceeding the size threshold.
type CheckLargeFiles struct {
	engine.GitCheck
}

// Evaluate inspects the FileStats from RepoInfo.
func (c CheckLargeFiles) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.FileStats == nil || len(info.FileStats.LargeFiles) == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	limit := 5
	if len(info.FileStats.LargeFiles) < limit {
		limit = len(info.FileStats.LargeFiles)
	}

	var lines []string

	for _, f := range info.FileStats.LargeFiles[:limit] {
		lines = append(lines, fmt.Sprintf("%s (%s)", f.Path, humanizeBytes(f.Size)))
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityLow,
		Summary:   fmt.Sprintf("%d large file(s) in HEAD", len(info.FileStats.LargeFiles)),
		Detail:    strings.Join(lines, "; "),
	}), nil
}

// CheckBinaryFiles detects binary files tracked in HEAD.
type CheckBinaryFiles struct {
	engine.GitCheck
}

// Evaluate inspects the FileStats from RepoInfo.
func (c CheckBinaryFiles) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.FileStats == nil || len(info.FileStats.BinaryFiles) == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	limit := 10
	if len(info.FileStats.BinaryFiles) < limit {
		limit = len(info.FileStats.BinaryFiles)
	}

	paths := strings.Join(info.FileStats.BinaryFiles[:limit], ", ")

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   fmt.Sprintf("%d binary file(s) tracked in HEAD", len(info.FileStats.BinaryFiles)),
		Detail:    paths,
	}), nil
}

// humanizeBytes formats a byte count into a human-readable string.
func humanizeBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)

	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

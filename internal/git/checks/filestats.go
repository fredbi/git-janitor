package checks

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// LargeFiles detects files in HEAD exceeding the size threshold.
type LargeFiles struct {
	gitCheck
}

func NewLargeFiles() LargeFiles {
	return LargeFiles{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"filestats-large-files",
				"detects files in HEAD exceeding the size threshold",
			),
		},
	}
}

// Evaluate inspects the FileStats from RepoInfo.
func (c LargeFiles) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c LargeFiles) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.FileStats == nil || len(info.FileStats.LargeFiles) == 0 {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}
	const maxReported = 5
	limit := min(len(info.FileStats.LargeFiles), maxReported)
	lines := make([]string, 0, limit)
	for _, f := range info.FileStats.LargeFiles[:limit] { // the slice is sorted: the top largest ones are captured
		lines = append(lines, fmt.Sprintf("%s (%s)", f.Path, humanizeBytes(f.Size)))
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityLow,
		Summary:   fmt.Sprintf("%d large file(s) in HEAD", len(info.FileStats.LargeFiles)),
		Detail:    strings.Join(lines, "; "),
	}), nil
}

// BinaryFiles detects binary files tracked in HEAD.
type BinaryFiles struct {
	gitCheck
}

func NewBinaryFiles() BinaryFiles {
	return BinaryFiles{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"filestats-binary",
				"detects binary files tracked in HEAD",
			),
		},
	}
}

// Evaluate inspects the FileStats from RepoInfo.
func (c BinaryFiles) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c BinaryFiles) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.FileStats == nil || len(info.FileStats.BinaryFiles) == 0 {
		return noAlert(c.Name())
	}

	const maxReported = 10
	limit := min(len(info.FileStats.BinaryFiles), maxReported)
	paths := strings.Join(info.FileStats.BinaryFiles[:limit], ", ")

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   fmt.Sprintf("%d binary file(s) tracked in HEAD", len(info.FileStats.BinaryFiles)),
		Detail:    paths,
	}), nil
}

// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"
	"sort"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// branchUniqueMinBytes hides branches that contribute too little to matter.
const (
	branchUniqueMinBytes  int64 = 1 * 1024 * 1024 // 1 MiB
	branchUniqueMaxListed       = 10
)

// BranchUniqueSize reports local branches that uniquely hold significant
// on-disk storage — i.e., deleting them would make that many bytes
// unreachable, and a subsequent deep-clean would reclaim the space.
//
// Informational only: the user decides whether a branch represents live
// work worth keeping or abandoned effort worth discarding. No action is
// suggested.
type BranchUniqueSize struct {
	gitCheck
}

func NewBranchUniqueSize() BranchUniqueSize {
	return BranchUniqueSize{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"branch-unique-size",
				"reports local branches holding significant unique storage (info-only)",
			),
		},
	}
}

// Evaluate inspects Branch.UniqueBytes on each local non-default branch.
func (c BranchUniqueSize) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c BranchUniqueSize) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	type entry struct {
		name  string
		bytes int64
	}

	var (
		candidates []entry
		total      int64
	)

	for _, b := range info.Branches {
		if b.IsRemote || b.Name == info.DefaultBranch {
			continue
		}

		if b.UniqueBytes < branchUniqueMinBytes {
			continue
		}

		candidates = append(candidates, entry{name: b.Name, bytes: b.UniqueBytes})
		total += b.UniqueBytes
	}

	if len(candidates) == 0 {
		return noAlert(c.Name())
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].bytes > candidates[j].bytes
	})

	limit := min(len(candidates), branchUniqueMaxListed)
	lines := make([]string, 0, limit)

	for _, e := range candidates[:limit] {
		lines = append(lines, fmt.Sprintf("%s (%s)", e.name, models.FormatBytes(e.bytes)))
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary: fmt.Sprintf("%d local branch(es) uniquely holding %s",
			len(candidates), models.FormatBytes(total)),
		Detail: strings.Join(lines, "; "),
	}), nil
}

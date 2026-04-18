// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// ForkUpstreamDefaultBehindLocal detects, on fork-kind repos (origin and
// upstream remotes pointing at distinct repositories), when the upstream
// remote's default branch is strictly behind the local default branch. The
// suggested corrective action is to push the local default branch to
// upstream without --force.
type ForkUpstreamDefaultBehindLocal struct {
	gitCheck
}

func NewForkUpstreamDefaultBehindLocal() ForkUpstreamDefaultBehindLocal {
	return ForkUpstreamDefaultBehindLocal{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"fork-upstream-default-behind-local",
				"detects when the upstream remote's default branch is strictly behind the local default branch (fork repos only)",
			),
		},
	}
}

func (c ForkUpstreamDefaultBehindLocal) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c ForkUpstreamDefaultBehindLocal) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Kind != models.RepoKindFork || !info.UpstreamDefaultBehindLocal || info.DefaultBranch == "" {
		return noAlert(c.Name())
	}

	suggestion := branchSuggestion("push-local-to-upstream", simpleSubject(info.DefaultBranch))

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityMedium,
		Summary:     fmt.Sprintf("upstream/%s is behind local %s — fork is lagging local", info.DefaultBranch, info.DefaultBranch),
		Detail:      info.DefaultBranch,
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

// ForkUpstreamDefaultBehindOrigin detects, on fork-kind repos, when the
// upstream remote's default branch is strictly behind the origin remote's
// default branch. Informational: the user's fork is not up to date with the
// canonical source, independent of the local checkout.
type ForkUpstreamDefaultBehindOrigin struct {
	gitCheck
}

func NewForkUpstreamDefaultBehindOrigin() ForkUpstreamDefaultBehindOrigin {
	return ForkUpstreamDefaultBehindOrigin{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"fork-upstream-default-behind-origin",
				"detects when the upstream remote's default branch is strictly behind the origin remote's default branch (fork repos only)",
			),
		},
	}
}

func (c ForkUpstreamDefaultBehindOrigin) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c ForkUpstreamDefaultBehindOrigin) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Kind != models.RepoKindFork || !info.UpstreamDefaultBehindOrigin || info.DefaultBranch == "" {
		return noAlert(c.Name())
	}

	suggestion := branchSuggestion("push-origin-to-upstream", simpleSubject(info.DefaultBranch))

	return singleAlert(models.Alert{
		CheckName:   c.Name(),
		Severity:    models.SeverityInfo,
		Summary:     fmt.Sprintf("upstream/%s is behind origin/%s — fork is lagging origin", info.DefaultBranch, info.DefaultBranch),
		Detail:      info.DefaultBranch,
		Suggestions: []models.ActionSuggestion{suggestion},
	}), nil
}

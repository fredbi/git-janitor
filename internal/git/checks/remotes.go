package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// RemoteNoOrigin detects repos with no remote named "origin",
// or repos with no remotes at all.
type RemoteNoOrigin struct {
	gitCheck
}

func NewRemoteNoOrigin() RemoteNoOrigin {
	return RemoteNoOrigin{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"remote-no-origin",
				"detects repos with no remote named origin",
			),
		},
	}
}

// Evaluate checks that an "origin" remote exists.
func (c RemoteNoOrigin) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c RemoteNoOrigin) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	if len(info.Remotes) == 0 {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityHigh,
			Summary:   "no remotes configured",
			Detail:    "this repository has no remote — it cannot sync with any server",
		}), nil
	}

	origin := models.FindRemote(info.Remotes, models.RemoteOrigin)
	if origin != nil {
		return noAlert(c.Name())
	}

	// There are remotes, but none named "origin".
	// If there's exactly one remote, suggest renaming it.
	if len(info.Remotes) == 1 {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityMedium,
			Summary:   fmt.Sprintf("remote %q should be named %q", info.Remotes[0].Name, models.RemoteOrigin),
			Detail:    fmt.Sprintf("single remote %q exists but is not named %q", info.Remotes[0].Name, models.RemoteOrigin),
			Suggestions: []models.ActionSuggestion{{
				ActionName:  "rename-remote",
				SubjectKind: models.SubjectRemote,
				Subjects:    simpleSubject(info.Remotes[0].Name, models.RemoteOrigin),
			}},
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityMedium,
		Summary:   "no remote named \"origin\"",
		Detail:    fmt.Sprintf("%d remotes exist but none is named %q", len(info.Remotes), models.RemoteOrigin),
	}), nil
}

// RemoteMisnamedUpstream detects forks where a distinct remote exists
// but is not named "upstream" (e.g., typo like "upstram").
type RemoteMisnamedUpstream struct {
	gitCheck
}

func NewRemoteMisnamedUpstream() RemoteMisnamedUpstream {
	return RemoteMisnamedUpstream{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"remote-misnamed-upstream",
				"detects fork repos where the upstream remote has an incorrect name",
			),
		},
	}
}

// Evaluate checks that for fork repos, the distinct remote is named "upstream".
func (c RemoteMisnamedUpstream) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c RemoteMisnamedUpstream) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	// Only relevant for forks (repos with distinct remote URLs).
	if info.Kind != models.RepoKindFork {
		return noAlert(c.Name())
	}

	// If "upstream" remote exists, we're fine.
	if models.FindRemote(info.Remotes, models.RemoteUpstream) != nil {
		return noAlert(c.Name())
	}

	// Find the distinct remote (the one that's not origin).
	misnamed, found := models.HasDistinctRemote(info.Remotes, backend.NormalizeURL)
	if !found {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityMedium,
		Summary:   fmt.Sprintf("fork remote %q should be named %q", misnamed, models.RemoteUpstream),
		Detail:    fmt.Sprintf("remote %q has a different URL from origin — likely a fork source that should be named %q", misnamed, models.RemoteUpstream),
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "rename-remote",
			SubjectKind: models.SubjectRemote,
			Subjects:    simpleSubject(misnamed, models.RemoteUpstream),
		}},
	}), nil
}

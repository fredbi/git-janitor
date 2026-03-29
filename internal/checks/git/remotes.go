package gitchecks

import (
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckRemoteNoOrigin detects repos with no remote named "origin",
// or repos with no remotes at all.
type CheckRemoteNoOrigin struct {
	engine.GitCheck
}

// Evaluate checks that an "origin" remote exists.
func (c CheckRemoteNoOrigin) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if len(info.Remotes) == 0 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityHigh,
			Summary:   "no remotes configured",
			Detail:    "this repository has no remote — it cannot sync with any server",
		}), nil
	}

	origin := git.FindRemote(info.Remotes, git.RemoteOrigin)
	if origin != nil {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	// There are remotes, but none named "origin".
	// If there's exactly one remote, suggest renaming it.
	if len(info.Remotes) == 1 {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityMedium,
			Summary:   fmt.Sprintf("remote %q should be named %q", info.Remotes[0].Name, git.RemoteOrigin),
			Detail:    fmt.Sprintf("single remote %q exists but is not named %q", info.Remotes[0].Name, git.RemoteOrigin),
			Suggestions: []engine.ActionSuggestion{{
				ActionName:  "rename-remote",
				SubjectKind: engine.SubjectRemote,
				Subjects:    []string{info.Remotes[0].Name, git.RemoteOrigin},
			}},
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityMedium,
		Summary:   "no remote named \"origin\"",
		Detail:    fmt.Sprintf("%d remotes exist but none is named %q", len(info.Remotes), git.RemoteOrigin),
	}), nil
}

// CheckRemoteMisnamedUpstream detects forks where a distinct remote exists
// but is not named "upstream" (e.g., typo like "upstram").
type CheckRemoteMisnamedUpstream struct {
	engine.GitCheck
}

// Evaluate checks that for fork repos, the distinct remote is named "upstream".
func (c CheckRemoteMisnamedUpstream) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	// Only relevant for forks (repos with distinct remote URLs).
	if info.Kind != git.KindFork {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	// If "upstream" remote exists, we're fine.
	if git.FindRemote(info.Remotes, git.RemoteUpstream) != nil {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	// Find the distinct remote (the one that's not origin).
	misnamed, found := git.HasDistinctRemote(info.Remotes)
	if !found {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityMedium,
		Summary:   fmt.Sprintf("fork remote %q should be named %q", misnamed, git.RemoteUpstream),
		Detail:    fmt.Sprintf("remote %q has a different URL from origin — likely a fork source that should be named %q", misnamed, git.RemoteUpstream),
		Suggestions: []engine.ActionSuggestion{{
			ActionName:  "rename-remote",
			SubjectKind: engine.SubjectRemote,
			Subjects:    []string{misnamed, git.RemoteUpstream},
		}},
	}), nil
}

package gitchecks

import (
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// CheckConfigNoEmail detects repositories with no user.email configured.
type CheckConfigNoEmail struct {
	engine.GitCheck
}

// Evaluate inspects the RepoConfig from RepoInfo.
func (c CheckConfigNoEmail) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.Config == nil || info.Config.UserEmail.Value != "" {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityMedium,
		Summary:   "user.email is not configured",
		Detail:    "user.email is not configured (scope: unset)",
	}), nil
}

// CheckConfigUnsigned detects repositories where commit signing is not enabled.
type CheckConfigUnsigned struct {
	engine.GitCheck
}

// Evaluate inspects the RepoConfig from RepoInfo.
func (c CheckConfigUnsigned) Evaluate(info *git.RepoInfo) (iter.Seq[engine.Alert], error) {
	if info.Config == nil || info.Config.CommitSign.Value == "true" {
		return singleAlert(engine.Alert{
			CheckName: c.Name(),
			Severity:  engine.SeverityNone,
		}), nil
	}

	scope := string(info.Config.CommitSign.Scope)
	if scope == "" {
		scope = "unset"
	}

	return singleAlert(engine.Alert{
		CheckName: c.Name(),
		Severity:  engine.SeverityInfo,
		Summary:   "commit signing is not enabled",
		Detail:    fmt.Sprintf("commit.gpgsign=%q (scope: %s)", info.Config.CommitSign.Value, scope),
	}), nil
}

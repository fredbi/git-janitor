package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/git/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// ConfigNoEmail detects repositories with no user.email configured.
type ConfigNoEmail struct {
	gitCheck
}

func NewConfigNoEmail() ConfigNoEmail {
	return ConfigNoEmail{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"config-no-email",
				"detects repositories with no user.email configured",
			),
		},
	}
}

// Evaluate inspects the RepoConfig from RepoInfo.
func (c ConfigNoEmail) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c ConfigNoEmail) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Config == nil || info.Config.UserEmail.Value != "" {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityMedium,
		Summary:   "user.email is not configured",
		Detail:    "user.email is not configured (scope: unset)",
	}), nil
}

// ConfigUnsigned detects repositories where commit signing is not enabled.
type ConfigUnsigned struct {
	gitCheck
}

func NewConfigUnsigned() ConfigUnsigned {
	return ConfigUnsigned{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"config-unsigned",
				"detects repositories where commit signing is not enabled",
			),
		},
	}
}

// Evaluate inspects the RepoConfig from RepoInfo.
func (c ConfigUnsigned) Evaluate(ctx context.Context) (iter.Seq[models.Alert], error) {
	info, err := repoInfoCtx(ctx)
	if err != nil {
		return nil, err
	}

	return c.evaluate(info)
}

func (c ConfigUnsigned) evaluate(info *backend.RepoInfo) (iter.Seq[models.Alert], error) {
	if info.Config == nil || info.Config.CommitSign.Value == "true" {
		return singleAlert(models.Alert{
			CheckName: c.Name(),
			Severity:  models.SeverityNone,
		}), nil
	}

	scope := string(info.Config.CommitSign.Scope)
	if scope == "" {
		scope = "unset"
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityInfo,
		Summary:   "commit signing is not enabled",
		Detail:    fmt.Sprintf("commit.gpgsign=%q (scope: %s)", info.Config.CommitSign.Value, scope),
	}), nil
}

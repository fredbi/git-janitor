// SPDX-License-Identifier: Apache-2.0

package checks

import (
	"context"
	"fmt"
	"iter"

	"github.com/fredbi/git-janitor/internal/models"
)

// RemoteCredentials detects remotes with embedded passwords or tokens in the URL.
// Credentials in URLs are a security risk — they may be logged, cached, or
// leaked via git config. Use credential helpers or environment variables instead.
type RemoteCredentials struct {
	gitCheck
}

func NewRemoteCredentials() RemoteCredentials {
	return RemoteCredentials{
		gitCheck: gitCheck{
			Describer: models.NewDescriber(
				"remote-credentials-in-url",
				"detects remotes with passwords or tokens embedded in the URL",
			),
		},
	}
}

func (c RemoteCredentials) Evaluate(_ context.Context, info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	return c.evaluate(info)
}

func (c RemoteCredentials) evaluate(info *models.RepoInfo) (iter.Seq[models.Alert], error) {
	var subjects []models.ActionSubject

	for _, rm := range info.Remotes {
		if models.URLHasCredentials(rm.FetchURL) {
			cleanURL := models.StripURLCredentials(rm.FetchURL)
			subjects = append(subjects, models.ActionSubject{
				Subject: rm.Name,
				Params:  []string{cleanURL},
			})
		}
	}

	if len(subjects) == 0 {
		return noAlert(c.Name())
	}

	return singleAlert(models.Alert{
		CheckName: c.Name(),
		Severity:  models.SeverityHigh,
		Summary:   fmt.Sprintf("%d remote(s) with credentials in URL", len(subjects)),
		Detail:    "Remote URLs should not contain passwords or tokens. Use credential helpers or environment variables instead.",
		Suggestions: []models.ActionSuggestion{{
			ActionName:  "strip-remote-credentials",
			SubjectKind: models.SubjectRemote,
			Subjects:    subjects,
		}},
	}), nil
}

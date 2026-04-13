// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fredbi/git-janitor/internal/models"
)

// OpenInBrowser opens the GitLab project URL in the default browser.
type OpenInBrowser struct {
	gitlabAction
}

// NewOpenInBrowser creates a new OpenInBrowser action.
func NewOpenInBrowser() OpenInBrowser {
	return OpenInBrowser{
		gitlabAction: gitlabAction{
			Describer: models.NewDescriber(
				"gitlab-open-in-browser",
				"open the GitLab project in a browser",
			),
		},
	}
}

func (OpenInBrowser) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a OpenInBrowser) Execute(ctx context.Context, repoInfo *models.RepoInfo, subjects []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	url := repoInfo.Platform.HTMLURL
	if len(subjects) > 0 && subjects[0] != "" {
		url = subjects[0]
	}

	if url == "" {
		return models.Result{}, errors.New("no URL available")
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return models.Result{}, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return models.Result{
			OK:      false,
			Message: fmt.Sprintf("failed to open browser: %v", err),
		}, err
	}

	return models.Result{
		OK:      true,
		Message: fmt.Sprintf("opened %s in browser", url),
	}, nil
}

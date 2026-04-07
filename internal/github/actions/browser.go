// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fredbi/git-janitor/internal/github/backend"
	"github.com/fredbi/git-janitor/internal/models"
)

// OpenInBrowser opens a URL in the default browser.
// The URL is passed as the first subject.
type OpenInBrowser struct {
	githubAction
}

func NewOpenInBrowser() OpenInBrowser {
	return OpenInBrowser{
		githubAction: githubAction{
			Describer: models.NewDescriber(
				"open-in-browser",
				"launch a browser and open the URL",
			),
		},
	}
}

func (OpenInBrowser) ApplyTo() models.SubjectKind { return models.SubjectRepo }

func (a OpenInBrowser) Execute(ctx context.Context, repoInfo *models.RepoInfo, subjects []string) (models.Result, error) {
	if repoInfo.Platform == nil {
		return models.Result{}, errors.New("no platform info available")
	}

	runner, err := runnerCtx(ctx)
	if err != nil {
		return models.Result{}, err
	}

	return a.execute(ctx, runner, repoInfo.Platform, subjects)
}

func (a OpenInBrowser) execute(ctx context.Context, _ *backend.Runner, _ *models.PlatformInfo, subjects []string) (models.Result, error) {
	if len(subjects) == 0 {
		return models.Result{}, errors.New("no URL provided")
	}

	url := subjects[0]

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

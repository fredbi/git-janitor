// SPDX-License-Identifier: Apache-2.0

package gitactions

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/git"
)

// ActionOpenInBrowser opens a URL in the default browser.
// The URL is passed as the first subject.
type ActionOpenInBrowser struct {
	engine.GitAction
}

func (ActionOpenInBrowser) ApplyTo() engine.SubjectKind { return engine.SubjectRepo }

func (a ActionOpenInBrowser) Execute(
	_ context.Context,
	_ *git.Runner,
	_ *git.RepoInfo,
	subjects []string,
) (engine.Result, error) {
	if len(subjects) == 0 {
		return engine.Result{}, fmt.Errorf("no URL provided")
	}

	url := subjects[0]

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return engine.Result{}, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return engine.Result{
			OK:      false,
			Message: fmt.Sprintf("failed to open browser: %v", err),
		}, err
	}

	return engine.Result{
		OK:      true,
		Message: fmt.Sprintf("opened %s in browser", url),
	}, nil
}

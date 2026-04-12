// SPDX-License-Identifier: Apache-2.0

package quickactions

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// runPreCommands executes each pre-command synchronously in the working
// directory identified by params["workdir"]. Placeholder substitution is
// applied to each command string before execution.
//
// Each command is passed to "bash -c" so it can use shell features (pipes,
// variables, etc.). If any command exits non-zero, the remaining commands
// are skipped and the error is returned.
func runPreCommands(ctx context.Context, params map[string]string, commands []string) error {
	wd := params["workdir"]
	if wd != "" {
		wd = filepath.Clean(wd)
	}

	for _, raw := range commands {
		cmd := substitute(raw, params)

		proc := exec.CommandContext(ctx, "bash", "-c", cmd)
		if wd != "" {
			proc.Dir = wd
		}

		output, err := proc.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s: %w\n%s", cmd, err, output)
		}
	}

	return nil
}

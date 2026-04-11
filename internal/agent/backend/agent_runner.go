// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fredbi/git-janitor/internal/config"
)

const defaultAgentTimeout = 10 * time.Minute

// Runner invokes an AI agent CLI tool as a subprocess.
type Runner struct {
	Command []string
	Model   string
	Timeout time.Duration
	Env     config.AgentEnvConfig
	WorkDir string // working directory for the subprocess
}

// NewRunner creates a Runner from the agent config and a working directory.
func NewRunner(cfg config.AgentConfig, workDir string) *Runner {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultAgentTimeout
	}

	return &Runner{
		Command: cfg.Command,
		Model:   cfg.Model,
		Timeout: timeout,
		Env:     cfg.Env,
		WorkDir: workDir,
	}
}

// CommandString returns a human-readable representation of the agent invocation
// for inclusion in command logs. The prompt is replaced by a truncated summary.
func (r *Runner) CommandString(prompt string) string {
	parts := make([]string, 0, len(r.Command)+4) //nolint:mnd // base + flags + prompt summary
	parts = append(parts, r.Command...)
	parts = append(parts, "--print")

	if r.Model != "" {
		parts = append(parts, "--model", r.Model)
	}

	const maxPromptLen = 80

	summary := prompt
	if len(summary) > maxPromptLen {
		summary = summary[:maxPromptLen] + "..."
	}

	// Replace newlines for single-line log entry.
	summary = strings.ReplaceAll(summary, "\n", " ")
	parts = append(parts, fmt.Sprintf("%q", summary))

	return "agent: " + strings.Join(parts, " ")
}

// Run invokes the agent CLI with the given prompt and returns its output.
// The prompt is passed via the --print flag (for claude) or stdin, depending
// on the command structure.
func (r *Runner) Run(ctx context.Context, prompt string) (string, error) {
	if len(r.Command) == 0 {
		return "", fmt.Errorf("agent: no command configured")
	}

	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultAgentTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build the command: command[0] is the binary, rest are base args.
	// Append --print and --model flags, then the prompt as the last argument.
	args := make([]string, 0, len(r.Command)+4) //nolint:mnd // base + flags + prompt
	args = append(args, r.Command[1:]...)
	args = append(args, "--print")

	if r.Model != "" {
		args = append(args, "--model", r.Model)
	}

	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, r.Command[0], args...)
	cmd.Dir = r.WorkDir
	cmd.Env = r.buildEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return "", fmt.Errorf("agent %s: %w: %s", r.Command[0], err, detail)
		}

		return "", fmt.Errorf("agent %s: %w", r.Command[0], err)
	}

	return stdout.String(), nil
}

// buildEnv constructs the environment for the subprocess:
// inherit current env, add configured vars, remove configured vars.
func (r *Runner) buildEnv() []string {
	env := os.Environ()

	// Remove specified vars.
	removeSet := make(map[string]bool, len(r.Env.Remove))
	for _, k := range r.Env.Remove {
		removeSet[k] = true
	}

	filtered := env[:0]

	for _, e := range env {
		k, _, _ := strings.Cut(e, "=")
		if !removeSet[k] {
			filtered = append(filtered, e)
		}
	}

	// Add specified vars.
	for k, v := range r.Env.Add {
		filtered = append(filtered, k+"="+v)
	}

	return filtered
}

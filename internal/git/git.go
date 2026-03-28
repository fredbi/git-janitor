package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Runner executes git CLI commands against a specific repository directory.
type Runner struct {
	// Dir is the working directory for git commands (the repo root).
	Dir string

	// Timeout overrides the default command timeout.
	// Zero means use the default (30s).
	Timeout time.Duration
}

// NewRunner returns a Runner for the given repository directory.
func NewRunner(dir string) *Runner {
	return &Runner{Dir: dir}
}

// run executes a git command and returns its stdout output.
//
// Stderr is captured and included in the error on failure.
func (r *Runner) run(ctx context.Context, args ...string) (string, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.Dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, detail)
		}

		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return stdout.String(), nil
}

// runWithStdin executes a git command with the given string as stdin.
func (r *Runner) runWithStdin(ctx context.Context, stdin string, args ...string) (string, error) {
	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.Dir
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, detail)
		}

		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return stdout.String(), nil
}

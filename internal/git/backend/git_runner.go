package backend

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

	// CmdLog records every git command executed by this runner.
	// Enable with StartLogging(). Retrieve with Commands().
	CmdLog []string
	logging bool
}

// NewRunner returns a Runner for the given repository directory.
func NewRunner(dir string) *Runner {
	return &Runner{Dir: dir}
}

// StartLogging enables command logging. All subsequent git commands
// executed by this runner (and worktree runners it creates) are recorded.
func (r *Runner) StartLogging() {
	r.logging = true
	r.CmdLog = nil
}

// Commands returns the recorded command log.
func (r *Runner) Commands() []string {
	return r.CmdLog
}

// AppendLog adds a custom entry to the command log (e.g. agent invocations).
func (r *Runner) AppendLog(entry string) {
	if r.logging {
		r.CmdLog = append(r.CmdLog, entry)
	}
}

// Run executes a git command and returns its stdout output.
//
// Stderr is captured and included in the error on failure.
func (r *Runner) Run(ctx context.Context, args ...string) (string, error) {
	return r.run(ctx, args...)
}

func (r *Runner) run(ctx context.Context, args ...string) (string, error) {
	if r.logging {
		r.CmdLog = append(r.CmdLog, "git "+strings.Join(args, " "))
	}

	timeout := r.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.Dir
	cmd.Env = append(cmd.Environ(),
		"LC_ALL=C",              // force English output for parsing
		"GIT_TERMINAL_PROMPT=0", // never prompt for credentials (fail fast instead of hanging)
	)

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
	cmd.Env = append(cmd.Environ(),
		"LC_ALL=C",
		"GIT_TERMINAL_PROMPT=0",
	)
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

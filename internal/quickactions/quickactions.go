// SPDX-License-Identifier: Apache-2.0

// Package quickactions implements user-configured shell commands that can be
// launched on demand from the TUI via the quick-actions popup (Ctrl+K).
//
// Unlike checks and actions, quick actions are not built into the binary:
// they are declared entirely in configuration and registered at runtime.
// A single generic [QuickAction] type covers every entry — the only thing
// that varies is its command, subject, and placeholder substitutions.
//
// Quick actions are spawned detached: git-janitor starts the process, sets
// its working directory, and immediately returns. There is no waiting, no
// stdio capture, and no result reporting. The expectation is that the
// command opens its own terminal, editor, or other foreground UI.
package quickactions

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fredbi/git-janitor/internal/models"
)

// ErrEmptyCommand is returned when a [QuickAction] is constructed with an
// empty command list.
var ErrEmptyCommand = errors.New("quickactions: command must have at least one element")

// ErrUnknownSubject is returned when a [QuickAction] is constructed with a
// subject string that does not match any known [models.SubjectKind].
var ErrUnknownSubject = errors.New("quickactions: unknown subject kind")

// QuickAction is a single self-describing shell command exposed to the
// quick-actions popup.
//
// The qualified name combines the owning root key with the user-given name
// (for example "0/open-in-terminal") so that several roots may declare
// distinct entries that share a display name.
type QuickAction struct {
	rootKey      string
	name         string
	description  string
	subject      models.SubjectKind
	command      []string
	preCommands  []string
	initCommands []string
}

// Params groups the construction parameters for a [QuickAction] so the
// New() call signature stays manageable as we add more fields.
type Params struct {
	RootKey      string
	Subject      string
	Name         string
	Description  string
	Command      []string
	PreCommands  []string
	InitCommands []string
}

// New constructs a [QuickAction] from the given parameters.
func New(p Params) (*QuickAction, error) {
	if len(p.Command) == 0 {
		return nil, ErrEmptyCommand
	}

	kind, ok := parseSubject(p.Subject)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSubject, p.Subject)
	}

	return &QuickAction{
		rootKey:      p.RootKey,
		name:         p.Name,
		description:  p.Description,
		subject:      kind,
		command:      p.Command,
		preCommands:  p.PreCommands,
		initCommands: p.InitCommands,
	}, nil
}

// InitCommands returns a copy of the init commands (for tests).
func (q *QuickAction) InitCommands() []string {
	out := make([]string, len(q.initCommands))
	copy(out, q.initCommands)

	return out
}

// PreCommands returns a copy of the pre commands (for tests).
func (q *QuickAction) PreCommands() []string {
	out := make([]string, len(q.preCommands))
	copy(out, q.preCommands)

	return out
}

// Name returns the qualified registry key ("{rootKey}/{name}").
func (q *QuickAction) Name() string {
	return q.rootKey + "/" + q.name
}

// DisplayName returns the bare action name (without the root prefix), which
// is what the popup shows to the user.
func (q *QuickAction) DisplayName() string {
	return q.name
}

// RootKey returns the owning root key for this quick action.
func (q *QuickAction) RootKey() string {
	return q.rootKey
}

// Description returns the human-readable hint shown in the popup list.
func (q *QuickAction) Description() string {
	return q.description
}

// Subject returns the [models.SubjectKind] this quick action operates on.
func (q *QuickAction) Subject() models.SubjectKind {
	return q.subject
}

// Command returns a copy of the configured command and its arguments,
// without any placeholder substitution. Mostly useful for tests.
func (q *QuickAction) Command() []string {
	out := make([]string, len(q.command))
	copy(out, q.command)

	return out
}

// Run substitutes placeholders in the command's arguments using the supplied
// params and spawns the resulting command detached from git-janitor.
//
// When the quick action has [InitCommands], a temporary bash init script is
// generated with a standard preamble (source ~/.bashrc, set terminal title
// via PS1, cd into the working directory) followed by the user commands.
// The {{init-file}} placeholder in the command args is then resolved to the
// temp file path.
//
// The first element of the command is looked up via [exec.LookPath] (i.e.
// against $PATH). The process is started with [os.exec.Cmd.Start]; git-janitor
// does not call Wait, so the child becomes a zombie until reaped by init when
// it eventually exits. Stdio is redirected to /dev/null so it cannot
// interfere with the TUI.
//
// The provided context is currently consulted only for cancellation between
// substitution and Start; it is not propagated to the child.
func (q *QuickAction) Run(ctx context.Context, params map[string]string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(q.command) == 0 {
		return ErrEmptyCommand
	}

	// Execute pre-commands synchronously in the working directory before
	// spawning the terminal. If any pre-command fails, abort.
	if len(q.preCommands) > 0 {
		if err := runPreCommands(ctx, params, q.preCommands); err != nil {
			return fmt.Errorf("quickactions: pre-command failed: %w", err)
		}
	}

	// When the pre-commands created a worktree, redirect the init script's
	// working directory (and the spawned process's cwd) to the worktree
	// path so the terminal opens there instead of the main repo.
	if wt, ok := params["worktree"]; ok && wt != "" {
		if info, statErr := os.Stat(wt); statErr == nil && info.IsDir() {
			params = copyParams(params)
			params["workdir"] = wt
		}
	}

	// When init-commands are configured, generate a temp init script and
	// inject its path as the {{init-file}} placeholder value.
	if len(q.initCommands) > 0 {
		initFile, err := writeInitScript(params, q.initCommands)
		if err != nil {
			return fmt.Errorf("quickactions: creating init script: %w", err)
		}

		params = copyParams(params)
		params["init-file"] = initFile
	}

	bin, err := exec.LookPath(q.command[0])
	if err != nil {
		return fmt.Errorf("quickactions: looking up %q: %w", q.command[0], err)
	}

	args := make([]string, 0, len(q.command)-1)
	for _, arg := range q.command[1:] {
		args = append(args, substitute(arg, params))
	}

	// Detach the spawned process from the caller's context: this is a
	// "fire-and-forget" run, so cancelling ctx must not kill the child
	// (which is typically a long-lived terminal window).
	//nolint:noctx // detached on purpose; ctx must not kill the child
	cmd := exec.Command(bin, args...)

	// Detach from git-janitor's stdio so the child cannot scribble on the TUI.
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err == nil {
		cmd.Stdin = devNull
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}

	// Set working directory from {{workdir}} when present so that commands
	// like "gnome-terminal" without --working-directory still land in the
	// right place. We pass it through filepath.Clean so trailing slashes
	// don't trip up some shells.
	if wd, ok := params["workdir"]; ok && wd != "" {
		cmd.Dir = filepath.Clean(wd)
	}

	if startErr := cmd.Start(); startErr != nil {
		if devNull != nil {
			_ = devNull.Close()
		}

		return fmt.Errorf("quickactions: starting %q: %w", bin, startErr)
	}

	// Reap the process state asynchronously so we don't leak zombies.
	// We deliberately ignore the result.
	go func() {
		_ = cmd.Wait()
		if devNull != nil {
			_ = devNull.Close()
		}
	}()

	return nil
}

// copyParams returns a shallow copy of a params map so the caller's
// original is not mutated when we inject {{init-file}}.
func copyParams(m map[string]string) map[string]string {
	cp := make(map[string]string, len(m)+1)
	maps.Copy(cp, m)

	return cp
}

// substitute replaces {{key}} occurrences in arg with params[key]. Unknown
// keys are left untouched so the user can spot typos in the spawned command.
func substitute(arg string, params map[string]string) string {
	if !strings.Contains(arg, "{{") {
		return arg
	}

	out := arg
	for k, v := range params {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}

	return out
}

// parseSubject converts the YAML string form of a subject into a
// [models.SubjectKind].
func parseSubject(s string) (models.SubjectKind, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "none":
		return models.SubjectNone, true
	case "repo":
		return models.SubjectRepo, true
	case "remote":
		return models.SubjectRemote, true
	case "branch":
		return models.SubjectBranch, true
	case "stash":
		return models.SubjectStash, true
	case "tag":
		return models.SubjectTag, true
	case "issues":
		return models.SubjectIssues, true
	case "pull_requests", "pull-requests", "pullrequests":
		return models.SubjectPullRequests, true
	case "workflow_runs", "workflow-runs", "workflowruns":
		return models.SubjectWorkflowRuns, true
	default:
		return 0, false
	}
}

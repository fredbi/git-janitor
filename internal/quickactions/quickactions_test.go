// SPDX-License-Identifier: Apache-2.0

package quickactions

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestNew_ErrEmptyCommand(t *testing.T) {
	_, err := New(Params{RootKey: "0", Subject: "repo", Name: "noop"})
	if !errors.Is(err, ErrEmptyCommand) {
		t.Fatalf("expected ErrEmptyCommand, got %v", err)
	}
}

func TestNew_ErrUnknownSubject(t *testing.T) {
	_, err := New(Params{RootKey: "0", Subject: "weird", Name: "noop", Command: []string{"true"}})
	if !errors.Is(err, ErrUnknownSubject) {
		t.Fatalf("expected ErrUnknownSubject, got %v", err)
	}
}

func TestNew_QualifiedName(t *testing.T) {
	qa, err := New(Params{
		RootKey: "3", Subject: "repo", Name: "open-in-terminal",
		Description: "Open repo", Command: []string{"gnome-terminal"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if qa.Name() != "3/open-in-terminal" {
		t.Errorf("Name() = %q, want %q", qa.Name(), "3/open-in-terminal")
	}

	if qa.DisplayName() != "open-in-terminal" {
		t.Errorf("DisplayName() = %q, want %q", qa.DisplayName(), "open-in-terminal")
	}

	if qa.RootKey() != "3" {
		t.Errorf("RootKey() = %q, want %q", qa.RootKey(), "3")
	}

	if qa.Subject() != models.SubjectRepo {
		t.Errorf("Subject() = %v, want SubjectRepo", qa.Subject())
	}
}

func TestParseSubject(t *testing.T) {
	cases := map[string]struct {
		input string
		want  models.SubjectKind
		ok    bool
	}{
		"empty":    {"", models.SubjectNone, true},
		"repo":     {"repo", models.SubjectRepo, true},
		"branch":   {"branch", models.SubjectBranch, true},
		"upper":    {"BRANCH", models.SubjectBranch, true},
		"trim":     {" stash ", models.SubjectStash, true},
		"prs":      {"pull-requests", models.SubjectPullRequests, true},
		"workflow": {"workflow_runs", models.SubjectWorkflowRuns, true},
		"unknown":  {"banana", 0, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok := parseSubject(tc.input)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSubstitute(t *testing.T) {
	params := map[string]string{
		"repo":    "git-janitor",
		"workdir": "/home/fred/src/git-janitor",
		"subject": "main",
	}

	cases := []struct {
		in   string
		want string
	}{
		{"--title={{repo}}", "--title=git-janitor"},
		{"--working-directory={{workdir}}", "--working-directory=/home/fred/src/git-janitor"},
		{"plain-arg", "plain-arg"}, // no placeholders
		{"{{subject}}", "main"},
		{"{{unknown}}", "{{unknown}}"}, // left as-is
		{"{{repo}}-{{subject}}", "git-janitor-main"},
	}

	for _, tc := range cases {
		got := substitute(tc.in, params)
		if got != tc.want {
			t.Errorf("substitute(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRun_LookupFails(t *testing.T) {
	qa, err := New(Params{
		RootKey: "0", Subject: "repo", Name: "missing",
		Command: []string{"git-janitor-no-such-binary-xyz"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := qa.Run(context.Background(), nil); err == nil {
		t.Error("expected lookup error, got nil")
	}
}

func TestRun_SuccessDetached(t *testing.T) {
	qa, err := New(Params{RootKey: "0", Subject: "repo", Name: "noop", Command: []string{"true"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := qa.Run(context.Background(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_CancelledContext(t *testing.T) {
	qa, err := New(Params{RootKey: "0", Subject: "repo", Name: "noop", Command: []string{"true"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := qa.Run(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

const testMutated = "MUTATED"

func TestCommand_ReturnsCopy(t *testing.T) {
	qa, err := New(Params{RootKey: "0", Subject: "repo", Name: "noop", Command: []string{"echo", "hi"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got := qa.Command()
	got[0] = testMutated

	if qa.Command()[0] != "echo" {
		t.Errorf("Command() did not return a copy: internal state mutated")
	}
}

func TestWriteInitScript_PreambleAndCommands(t *testing.T) {
	params := map[string]string{
		"repo":    "my-repo",
		"workdir": "/home/dev/my-repo",
	}

	path, err := writeInitScript(params, []string{"git status", "echo hello"})
	if err != nil {
		t.Fatalf("writeInitScript: %v", err)
	}

	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading init script: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "source ~/.bashrc") {
		t.Error("init script missing .bashrc source")
	}

	if !strings.Contains(content, "__gj_title") {
		t.Error("init script missing __gj_title function")
	}

	if !strings.Contains(content, `__gj_title "my-repo"`) {
		t.Errorf("init script missing title call, got:\n%s", content)
	}

	if !strings.Contains(content, `cd "/home/dev/my-repo"`) {
		t.Errorf("init script missing cd, got:\n%s", content)
	}

	if !strings.Contains(content, "git status") {
		t.Error("init script missing 'git status'")
	}

	if !strings.Contains(content, "echo hello") {
		t.Error("init script missing 'echo hello'")
	}
}

func TestWriteInitScript_SubstitutesPlaceholders(t *testing.T) {
	params := map[string]string{
		"repo":    "my-repo",
		"workdir": "/home/dev/my-repo",
		"branch":  "feat/foo",
	}

	path, err := writeInitScript(params, []string{"git checkout {{branch}}"})
	if err != nil {
		t.Fatalf("writeInitScript: %v", err)
	}

	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	if !strings.Contains(string(data), "git checkout feat/foo") {
		t.Errorf("placeholder not substituted:\n%s", string(data))
	}
}

func TestRun_WithInitCommands(t *testing.T) {
	qa, err := New(Params{
		RootKey: "0", Subject: "repo", Name: "test",
		Command:      []string{"cat", "{{init-file}}"},
		InitCommands: []string{"echo hi"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	params := map[string]string{
		"repo":    "test-repo",
		"workdir": "/tmp",
	}

	if err := qa.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, ok := params["init-file"]; ok {
		t.Error("Run mutated the caller's params map")
	}
}

func TestRun_WithPreCommands(t *testing.T) {
	qa, err := New(Params{
		RootKey: "0", Subject: "repo", Name: "test",
		Command:     []string{"true"},
		PreCommands: []string{"echo pre-command-ran"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	params := map[string]string{"workdir": "/tmp"}

	if err := qa.Run(context.Background(), params); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_PreCommandFails(t *testing.T) {
	qa, err := New(Params{
		RootKey: "0", Subject: "repo", Name: "test",
		Command:     []string{"true"},
		PreCommands: []string{"false"}, // exits non-zero
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	params := map[string]string{"workdir": "/tmp"}

	if err := qa.Run(context.Background(), params); err == nil {
		t.Error("expected pre-command failure, got nil")
	}
}

func TestRun_PreCommandSubstitution(t *testing.T) {
	qa, err := New(Params{
		RootKey: "0", Subject: "repo", Name: "test",
		Command:     []string{"true"},
		PreCommands: []string{"test -d {{workdir}}"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	params := map[string]string{"workdir": "/tmp"}

	if err := qa.Run(context.Background(), params); err != nil {
		t.Fatalf("Run with valid workdir: %v", err)
	}
}

func TestInitCommands_Accessor(t *testing.T) {
	qa, err := New(Params{
		RootKey: "0", Subject: "repo", Name: "test",
		Command: []string{"true"}, InitCommands: []string{"git status", "ls"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cmds := qa.InitCommands()
	if len(cmds) != 2 || cmds[0] != "git status" || cmds[1] != "ls" {
		t.Errorf("InitCommands() = %v", cmds)
	}

	cmds[0] = testMutated
	if qa.InitCommands()[0] != "git status" {
		t.Error("InitCommands() did not return a copy")
	}
}

func TestPreCommands_Accessor(t *testing.T) {
	qa, err := New(Params{
		RootKey: "0", Subject: "repo", Name: "test",
		Command: []string{"true"}, PreCommands: []string{"mkdir /tmp/test"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cmds := qa.PreCommands()
	if len(cmds) != 1 || cmds[0] != "mkdir /tmp/test" {
		t.Errorf("PreCommands() = %v", cmds)
	}

	cmds[0] = testMutated
	if qa.PreCommands()[0] != "mkdir /tmp/test" {
		t.Error("PreCommands() did not return a copy")
	}
}

// SPDX-License-Identifier: Apache-2.0

package quickactions

import (
	"fmt"
	"os"
	"strings"
)

// initScriptPrefix is the temp file name prefix used by [writeInitScript].
const initScriptPrefix = "git-janitor-init-"

// bashPreamble is sourced at the top of every generated init script.
// It loads the user's .bashrc (so aliases, PATH, prompt, etc. are available),
// then defines a title() helper that patches PS1 to persistently set the
// terminal title — surviving prompt redraws that would otherwise overwrite
// a raw ANSI escape.
const bashPreamble = `#!/usr/bin/env bash

# --- git-janitor quick-action init script ---

# Load user environment.
[ -f ~/.bashrc ] && source ~/.bashrc

# Patch PS1 to set a persistent terminal title.
# If PS1 contains a title-escape sequence (\a), replace the title portion.
# Otherwise prepend a standard \033]0;TITLE\007 escape.
function __gj_title() {
    case "$PS1" in
        *\\a*)
            local prefix=${PS1%%\\a*}
            local search=${prefix##*;}
            local esearch="${search//\\/\\\\}"
            PS1="${PS1/$esearch/$@}"
            ;;
        *)
            PS1="\\[\\033]0;$@\\007\\]$PS1"
            ;;
    esac
}
`

// writeInitScript generates a temporary bash init script with the standard
// preamble followed by title/cd commands and the user's init commands. The
// placeholder values for {{repo}} and {{workdir}} are read from params.
//
// The caller is responsible for ensuring the file is eventually cleaned up,
// but in practice it lives in $TMPDIR and is tiny, so OS-level temp reaping
// is sufficient.
func writeInitScript(params map[string]string, initCommands []string) (string, error) {
	var b strings.Builder

	b.WriteString(bashPreamble)

	// Set terminal title from the repo display name.
	if title, ok := params["repo"]; ok && title != "" {
		fmt.Fprintf(&b, "__gj_title %q\n", title)
	}

	// cd into the working directory.
	if wd, ok := params["workdir"]; ok && wd != "" {
		fmt.Fprintf(&b, "cd %q\n", wd)
	}

	b.WriteString("\n")

	// Append user init commands (already substituted with placeholders by
	// the caller if needed — here we write them verbatim).
	for _, cmd := range initCommands {
		subst := substitute(cmd, params)
		b.WriteString(subst)
		b.WriteString("\n")
	}

	f, err := os.CreateTemp("", initScriptPrefix)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	path := f.Name()

	if _, err := f.WriteString(b.String()); err != nil {
		_ = f.Close()

		return "", fmt.Errorf("writing init script: %w", err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("closing init script: %w", err)
	}

	return path, nil
}

// SPDX-License-Identifier: Apache-2.0

package ux

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// copyToClipboard copies text to the system clipboard.
//
// It tries OSC 52 first (works on modern terminals without external tools),
// then falls back to xclip/xsel/wl-copy if available.
func copyToClipboard(text string) error {
	// Try command-line tools first — they give reliable feedback
	// (OSC 52 is fire-and-forget, can't tell if the terminal supports it).
	if err := clipboardViaTool(text); err == nil {
		return nil
	}

	// Fall back to OSC 52.
	return osc52Copy(text)
}

// clipboardViaTool tries xclip, xsel, then wl-copy.
func clipboardViaTool(text string) error {
	tools := []struct {
		name string
		args []string
	}{
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
		{"wl-copy", nil},
	}

	for _, t := range tools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			continue
		}

		cmd := exec.Command(path, t.args...)
		cmd.Stdin = strings.NewReader(text)

		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no clipboard tool available (tried xclip, xsel, wl-copy)")
}

// osc52Copy writes an OSC 52 escape sequence to stderr, which instructs
// the terminal emulator to copy the given text to the system clipboard.
//
// Works on kitty, alacritty, wezterm, iTerm2, Windows Terminal, foot, etc.
// Does NOT work on gnome-terminal or some older terminals.
func osc52Copy(text string) error {
	b64 := base64.StdEncoding.EncodeToString([]byte(text))

	// OSC 52 ; c ; <base64> BEL
	seq := fmt.Sprintf("\x1b]52;c;%s\x07", b64)

	// Detect tmux and wrap in passthrough DCS.
	if isTmux() {
		seq = fmt.Sprintf("\x1bPtmux;\x1b%s\x1b\\", seq)
	}

	_, err := fmt.Fprint(os.Stderr, seq)

	return err
}

func isTmux() bool {
	return strings.HasPrefix(os.Getenv("TERM_PROGRAM"), "tmux") ||
		os.Getenv("TMUX") != ""
}

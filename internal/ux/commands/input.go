// Copyright 2024 Frederic Bidon. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package commands

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Command represents a parsed user Command.
type Command struct {
	Name string
	Args []string
}

// Input wraps a bubbles/textinput for the Command zone.
type Input struct {
	Theme  *uxtypes.Theme
	Input  textinput.Model
	Width  int
	Height int
}

// New creates a new Input with sensible defaults.
func New(theme *uxtypes.Theme) Input {
	ti := textinput.New()
	ti.Placeholder = "Type a command (e.g. /help)"
	ti.Prompt = " > "
	ti.CharLimit = 256

	return Input{Theme: theme, Input: ti}
}

// SetSize updates the dimensions of the input widget.
func (c *Input) SetSize(w, h int) {
	c.Width = w
	c.Height = h
	c.Input.Width = w - 6 // account for prompt and border
}

// Focus gives keyboard focus to the text input.
func (c *Input) Focus() tea.Cmd {
	return c.Input.Focus()
}

// Blur removes keyboard focus from the text input.
func (c *Input) Blur() {
	c.Input.Blur()
}

// Update forwards a tea.Msg to the underlying text input.
func (c *Input) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	c.Input, cmd = c.Input.Update(msg)

	return cmd
}

// View renders the command input with a rounded border.
func (c *Input) View(focused bool) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(c.Width - 2)

	t := c.Theme
	if focused {
		border = border.BorderForeground(t.Tertiary)
	} else {
		border = border.BorderForeground(t.Dim)
	}

	return border.Render(c.Input.View())
}

// Submit parses the current input as a Command, clears it, and returns the Command.
// Returns nil if the input is empty.
func (c *Input) Submit() *Command {
	raw := strings.TrimSpace(c.Input.Value())
	c.Input.SetValue("")

	if raw == "" {
		return nil
	}

	parts := strings.Fields(raw)
	name := parts[0]

	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	return &Command{Name: name, Args: args}
}

// Sentinel values returned by ExecuteCommand to signal the caller.
const (
	CommandShowHelp   = "\x00help"   // signals the caller to open the help popup
	CommandShowConfig = "\x00config" // signals the caller to open the config wizard
	CommandScanRoots  = "\x00scan"   // signals the caller to scan configured roots
	CommandThemePrefix = "\x00theme:" // signals the caller to switch theme (prefix)
)

// ExecuteCommand processes a Command and returns a status message.
//
// Special sentinel values are returned when the caller must take action
// (e.g. open the help popup, switch theme). The "/themes" command returns
// [CommandThemePrefix] + "list" so the caller can resolve available theme names
// without creating an import cycle.
func ExecuteCommand(cmd *Command) string {
	if cmd == nil {
		return ""
	}

	switch cmd.Name {
	case "/help":
		return CommandShowHelp
	case "/quit", "/exit":
		return "quit"
	case "/config":
		return CommandShowConfig
	case "/scan":
		return CommandScanRoots
	case "/clear":
		return "Cleared."
	case "/theme":
		// No argument: cycle to next theme.
		return CommandThemePrefix
	case "/themes":
		// Return a sentinel so the caller can resolve theme names.
		return CommandThemePrefix + "list"
	default:
		if strings.HasPrefix(cmd.Name, "/theme") && len(cmd.Args) > 0 {
			return CommandThemePrefix + cmd.Args[0]
		}

		return "Unknown command: " + cmd.Name + ". Type /help for available commands."
	}
}

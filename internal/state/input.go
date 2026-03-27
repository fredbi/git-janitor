package state

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// command represents a parsed user command.
type command struct {
	name string
	args []string
}

// commandResult is a tea.Msg sent after a command is executed.
type commandResult struct {
	output string
}

// commandInput wraps a bubbles/textinput for the command zone.
type commandInput struct {
	input  textinput.Model
	width  int
	height int
}

func newCommandInput() commandInput {
	ti := textinput.New()
	ti.Placeholder = "Type a command (e.g. /help)"
	ti.Prompt = " > "
	ti.CharLimit = 256

	return commandInput{input: ti}
}

func (c *commandInput) SetSize(w, h int) {
	c.width = w
	c.height = h
	c.input.Width = w - 6 // account for prompt and border
}

func (c *commandInput) Focus() tea.Cmd {
	return c.input.Focus()
}

func (c *commandInput) Blur() {
	c.input.Blur()
}

func (c *commandInput) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)

	return cmd
}

func (c *commandInput) View(focused bool) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(c.width - 2)

	if focused {
		border = border.BorderForeground(lipgloss.Color("212"))
	} else {
		border = border.BorderForeground(lipgloss.Color("241"))
	}

	return border.Render(c.input.View())
}

// Submit parses the current input as a command, clears it, and returns the command.
// Returns nil if the input is empty.
func (c *commandInput) Submit() *command {
	raw := strings.TrimSpace(c.input.Value())
	c.input.SetValue("")

	if raw == "" {
		return nil
	}

	parts := strings.Fields(raw)
	name := parts[0]

	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	return &command{name: name, args: args}
}

const commandShowHelp = "\x00help"     // sentinel value: signals the caller to open the help popup
const commandShowConfig = "\x00config" // sentinel value: signals the caller to open the config wizard
const commandScanRoots = "\x00scan"    // sentinel value: signals the caller to scan configured roots

// executeCommand processes a command and returns a status message.
// Returns [commandShowHelp] when the help popup should be opened.
func executeCommand(cmd *command) string {
	if cmd == nil {
		return ""
	}

	switch cmd.name {
	case "/help":
		return commandShowHelp
	case "/quit", "/exit":
		return "quit"
	case "/config":
		return commandShowConfig
	case "/scan":
		return commandScanRoots
	case "/clear":
		return "Cleared."
	default:
		return "Unknown command: " + cmd.name + ". Type /help for available commands."
	}
}

package help

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

const helpText = `Git Janitor — Help
══════════════════

Navigation
──────────
  Tab / Shift+Tab    Cycle focus between Repositories, Right pane, and Input
  j / k  or  ↑ / ↓  Scroll within the focused list
  /                  Jump to command input (pre-fills "/")
  Esc                Close popup, clear filter, or leave command input
  q                  Quit (when not typing a command)
  Ctrl+C / Ctrl+Q    Quit immediately
  Ctrl+R             Fetch (git fetch --all --tags) and refresh selected repo
  Ctrl+H             Open this help popup

Mouse
─────
  Left-click          Focus a panel (Repos, Right pane, or Input)
  Left-click tab      Switch to the clicked tab (works in both panels)

Repositories Panel (left)
─────────────────────────
  Root Tabs:
    Ctrl+A / ← / →    Cycle through root tabs
    Left-click tab     Switch to the clicked root tab

  Filter (below tab bar):
    Type any text to filter repos (interpreted as RE2 regexp, case-insensitive).
    Plain text works as "path contains". Esc clears the filter.
    j / k / ↑ / ↓     Navigate the filtered list (passed through to list)

  Each configured root directory is shown as a separate tab.
  Non-git directories are highlighted in yellow.

Right pane Tabs
───────────────
  Ctrl+A / ← / →    Cycle through tabs (when right pane is focused)

  The right pane has five Tabs:
  • Facts     — quick recap: path, kind, SCM, last commit, branch,
                upstream, working tree status, remotes, stashes
  • Branches  — scrollable list of local and remote branches
  • Alerts    — notifications about the selected repository
  • Actions   — available cleanup operations
  • Recent    — log of recently performed actions

  Facts and Branches auto-populate when you select a repo, and refresh
  after Ctrl+R (fetch).

Commands (type in the input bar)
────────
  /help              Show this help popup
  /config            Open the configuration wizard
  /scan              Scan all configured roots for repositories
  /theme             Cycle to the next color theme
  /theme <name>      Switch to a specific theme
  /themes            List available theme names
  /clear             Clear the status bar message
  /quit              Quit the application

  Available themes:
    default, dracula, gruvbox, tokyo-night,
    solarized, nord, catppuccin

Configuration Wizard
────────────────────
  Opens automatically on first launch (no config), or via /config.

  Root List:
    ↑/↓ or j/k    Navigate the root list
    Enter          Edit the selected root (choose field: Path, Name, or Interval)
    [A]            Add a new root directory
    [D]            Delete the selected root
    [S]            Save changes to disk
    Esc            Close the wizard

  Edit Root:
    ↑/↓            Select field to edit (Path, Name, or Interval)
    Enter           Open the selected field for editing
    [S]            Save changes (available after edits)
    Esc            Go back to root list

  Add Root Flow (via [A] or when no roots exist):
    Step 1: Path     — absolute path to a directory containing git repos
    Step 2: Name     — display name (defaults to directory name)
    Step 3: Interval — check frequency (e.g. "24h", "30m")
    Step 4: Review
      [S/Enter]  Save and close
      [A]        Save and add another
      [N]        Cancel
      Esc        Back to root list (or close if no roots)

  Config file: ~/.config/git-janitor/config.yaml

────────────────────
Press Esc or Ctrl+H to close this help.
`

// Popup is a scrollable overlay that displays help text.
type Popup struct {
	Viewport viewport.Model
	Visible  bool
	Width    int
	Height   int
}

// New creates a new Popup.
func New() Popup {
	vp := viewport.New(0, 0)
	vp.SetContent(helpText)

	return Popup{Viewport: vp}
}

// SetSize recalculates the popup dimensions (centered, ~80% of terminal).
func (h *Popup) SetSize(termWidth, termHeight int) {
	h.Width = termWidth * 3 / 4
	if h.Width < 40 {
		h.Width = min(40, termWidth)
	}

	h.Height = termHeight * 3 / 4
	if h.Height < 10 {
		h.Height = min(10, termHeight)
	}

	// Account for border (2 lines top/bottom, 2 cols left/right).
	h.Viewport.Width = h.Width - 4
	h.Viewport.Height = h.Height - 4
	h.Viewport.SetContent(helpText)
}

// Toggle toggles the visibility of the help popup.
func (h *Popup) Toggle() {
	h.Visible = !h.Visible
	if h.Visible {
		h.Viewport.GotoTop()
	}
}

// Show makes the help popup visible.
func (h *Popup) Show() {
	h.Visible = true
	h.Viewport.GotoTop()
}

// Hide hides the help popup.
func (h *Popup) Hide() {
	h.Visible = false
}

// Update handles messages for the help popup viewport.
func (h *Popup) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	h.Viewport, cmd = h.Viewport.Update(msg)

	return cmd
}

// View renders the help popup overlay.
func (h *Popup) View(termWidth, termHeight int) string {
	if !h.Visible {
		return ""
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(uxtypes.CurrentTheme.Secondary).
		Width(h.Width-2).
		Height(h.Height-2).
		Padding(0, 1)

	popup := border.Render(h.Viewport.View())

	// Center the popup on screen.
	return lipgloss.Place(
		termWidth, termHeight,
		lipgloss.Center, lipgloss.Center,
		popup,
	)
}

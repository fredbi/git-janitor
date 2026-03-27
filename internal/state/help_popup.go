package state

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const helpText = `Git Janitor — Help
══════════════════

Navigation
──────────
  Tab / Shift+Tab   Cycle focus between Repositories, Right pane, and Input
  j / k  or  ↑ / ↓  Scroll within the focused list
  /                  Jump to command input (pre-fills "/")
  Esc                Close popup or leave command input
  q                  Quit (when not typing a command)
  Ctrl+C / Ctrl+Q    Quit immediately

Mouse
─────
  Left-click          Focus a panel (Repos, Right pane, or Input)
  Left-click tab      Switch to the clicked tab (works in both panels)
  Click help popup    Dismiss the help popup

Repositories Panel (left) — Root Tabs
─────────────────────────────────────
  Ctrl+A             Cycle through root tabs (when repos panel is focused)
  ← / h              Previous root tab (when repos panel is focused)
  → / l              Next root tab (when repos panel is focused)
  Left-click tab     Switch to the clicked root tab
  j / k  or  ↑ / ↓  Navigate within the repository list

  Each configured root directory is shown as a separate tab.
  The tab title is the root's display name (defaults to directory name).
  When there are too many root tabs to fit, excess tabs are elided with "..."
  and appear as the user navigates to them.

Right Pane Tabs
───────────────
  Ctrl+A             Cycle through Alerts, Actions, and Recent tabs (when right pane is focused)
  ← / h              Previous tab (when right pane is focused)
  → / l              Next tab (when right pane is focused)

  The right pane has three overlapping tabs:
  • Alerts   — notifications about the selected repository (shown by default)
  • Actions  — available cleanup operations for the selected repository
  • Recent   — log of recently performed actions

Help
────
  Ctrl+H             Open this help popup
  /help              Open this help popup (from command input)

Commands (type in the input bar)
────────
  /help              Show this help popup
  /config            Open the configuration wizard to add/edit root directories
  /scan              Scan all configured roots for git repositories
  /clear             Clear the status bar message
  /quit              Quit the application

Configuration Wizard
────────────────────
  The wizard opens automatically on first launch if no config exists.
  It can also be opened at any time with /config.

  Root List (shown when roots already exist):
    • ↑/↓ or j/k  Navigate the root list.
    • Enter        Edit the selected root (choose field: Name or Interval).
    • [A]          Add a new root directory.
    • [S]          Save changes (shown when modifications exist).
    • [Esc]        Close the wizard.

  Edit Root (after pressing Enter on a root):
    • ↑/↓          Select which field to edit (Name or Interval).
    • Enter        Open the selected field for editing.
    • [Esc]        Go back to root list.

  Add Root Flow (shown when no roots exist, or via [A]):
    Step 1: Enter the absolute path to a directory containing git repos.
    Step 2: Enter a display name (defaults to directory name if left empty).
    Step 3: Set a check interval (e.g. "24h", "30m"). Default from config.
    Step 4: Review and confirm.
      • [Y/Enter]  Save the root and close the wizard.
      • [A]        Save the root and add another.
      • [N]        Cancel this entry.
      • [Esc]      Go back to root list (or close if no roots).

  Configuration is saved to: ~/.config/git-janitor/config.yaml

Alerts Tab (right, default)
───────────────────────────
  A scrollable table of alert messages for the selected repository.
  Columns:
  • Severity   — colored bullet: red (high), orange (medium), yellow (low)
  • Message    — alert title and detail
  • Scheduled  — [✓] if a resolution action is already scheduled, [ ] otherwise

Actions Tab (right)
───────────────────
  Displays available cleanup actions:
  • Clean stale branches
  • Remove merged branches
  • Run git gc to optimize storage
  • Prune stale remote-tracking references

Recent Tab (right)
──────────────────
  Displays a log of recently performed actions, such as:
  • Deleted branches
  • Completed git gc runs
  • Pruned remote references

Status Bar
──────────
  The bottom bar shows the result of the last command or contextual information.

────────────────────
Press Esc or Ctrl+H to close this help.
`

// helpPopup is a scrollable overlay that displays help text.
type helpPopup struct {
	viewport viewport.Model
	visible  bool
	width    int
	height   int
}

func newHelpPopup() helpPopup {
	vp := viewport.New(0, 0)
	vp.SetContent(helpText)

	return helpPopup{viewport: vp}
}

// SetSize recalculates the popup dimensions (centered, ~80% of terminal).
func (h *helpPopup) SetSize(termWidth, termHeight int) {
	h.width = termWidth * 3 / 4
	if h.width < 40 {
		h.width = min(40, termWidth)
	}

	h.height = termHeight * 3 / 4
	if h.height < 10 {
		h.height = min(10, termHeight)
	}

	// Account for border (2 lines top/bottom, 2 cols left/right).
	h.viewport.Width = h.width - 4
	h.viewport.Height = h.height - 4
	h.viewport.SetContent(helpText)
}

func (h *helpPopup) Toggle() {
	h.visible = !h.visible
	if h.visible {
		h.viewport.GotoTop()
	}
}

func (h *helpPopup) Show() {
	h.visible = true
	h.viewport.GotoTop()
}

func (h *helpPopup) Hide() {
	h.visible = false
}

func (h *helpPopup) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	h.viewport, cmd = h.viewport.Update(msg)

	return cmd
}

func (h *helpPopup) View(termWidth, termHeight int) string {
	if !h.visible {
		return ""
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(h.width-2).
		Height(h.height-2).
		Padding(0, 1)

	popup := border.Render(h.viewport.View())

	// Center the popup on screen.
	return lipgloss.Place(
		termWidth, termHeight,
		lipgloss.Center, lipgloss.Center,
		popup,
	)
}

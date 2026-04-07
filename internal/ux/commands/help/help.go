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
  g / G              Jump to top / bottom
  PgUp / PgDn        Page up / page down
  /                  Jump to command input (pre-fills "/")
  Esc / q            Close popup, clear filter, or leave command input
  Ctrl+C / Ctrl+Q    Quit
  Ctrl+R             Fetch (git fetch --all --tags) and refresh selected repo
  Ctrl+H             Contextual help for the focused panel/tab

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

  The right pane has six tabs:
  • Facts     — quick recap: path, kind, SCM, last commit, branch,
                upstream, working tree status, remotes, stash count
  • Branches  — scrollable list of local and remote branches
  • Stashes   — scrollable list of stash entries (most recent first)
  • Alerts    — notifications about the selected repository
  • Actions   — available cleanup operations
  • Recent    — log of recently performed actions

  Facts and Branches auto-populate when you select a repo, and refresh
  after Ctrl+R (fetch).

Commands (type in the input bar)
────────
  /help              Show this general help
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
Press Esc to close this help.
`

// Contextual help texts keyed by context name.
var contextHelp = map[string]string{ //nolint:gochecknoglobals // help text table
	"repos": `Repositories Panel
══════════════════

  Root Tabs:
    Ctrl+A / ← / →    Cycle through root tabs
    Left-click tab     Switch to the clicked root tab

  Filter:
    Type text to filter repos (RE2 regexp, case-insensitive).
    Esc                Clear filter

  Navigation:
    j / k  or  ↑ / ↓  Navigate the repo list
    g / G              Jump to top / bottom
    PgUp / PgDn        Page up / page down

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,

	"facts": `Facts Tab
═════════

  Displays a quick recap of the selected repository:
  path, kind, SCM, last commit, branch, upstream,
  working tree status, remotes, stashes.

  Navigation:
    j / k  or  ↑ / ↓  Scroll content
    g / G              Jump to top / bottom
    PgUp / PgDn        Page up / page down

  Auto-refreshes after Ctrl+R (fetch).

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,

	"branches": `Branches Tab
════════════

  Scrollable list of local and remote branches.

  Columns: Branch name, Hash, Last updated, Upstream

  Ordering:
    Local branches first:
      • Default branch always first
      • Current branch second (if not default)
      • Other locals by most recent commit
    Remote branches after:
      • origin first, upstream second, others alphabetically
      • Within same remote, by most recent commit

  Navigation:
    j / k  or  ↑ / ↓  Navigate branch list
    g / G              Jump to top / bottom
    PgUp / PgDn        Page up / page down

  Auto-refreshes after Ctrl+R (fetch).

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,

	"stashes": `Stashes Tab
═══════════

  Scrollable list of stash entries, most recent first.
  Each entry shows ref, branch, message, and age.

  Navigation:
    j / k  or  ↑ / ↓  Navigate stash list
    g / G              Jump to top / bottom
    PgUp / PgDn        Page up / page down

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,

	"alerts": `Alerts Tab
══════════

  Notifications about the selected repository,
  sorted by severity (highest first).

  Navigation:
    j / k  or  ↑ / ↓  Navigate alert list
    g / G              Jump to top / bottom
    PgUp / PgDn        Page up / page down

  Actions:
    Enter              Show suggested fix actions
    C                  Copy alert URL to clipboard

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,

	"actions": `Actions Tab
═══════════

  Suggested cleanup operations for the selected alert.

  Navigation:
    j / k  or  ↑ / ↓  Navigate suggestions
    PgUp / PgDn        Page up / page down

  Execute:
    Enter              Execute action on all subjects
                       (prompts for input if action needs a parameter)
    Ctrl+P             Open subject picker (multi-subject actions)

  Subject Picker (when active):
    Space              Toggle subject checkbox
    A                  Select all
    N                  Deselect all
    Enter              Execute with selected subjects
    Esc                Cancel picker

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,

	"recent": `Recent Tab
══════════

  Log of actions performed on the selected repository
  (last 30 days). Persisted across restarts.

  Navigation:
    j / k  or  ↑ / ↓  Navigate history
    g / G              Jump to top / bottom
    PgUp / PgDn        Page up / page down

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,

	"input": `Command Input
═════════════

  Type commands starting with "/":

    /help              Show general help
    /config            Open the configuration wizard
    /scan              Scan all configured roots for repositories
    /theme             Cycle to the next color theme
    /theme <name>      Switch to a specific theme
    /themes            List available theme names
    /clear             Clear the status bar message
    /quit              Quit the application

  Esc                  Leave input bar

────────────────────
Press Esc or Ctrl+H to close.  /help for general help.
`,
}

// Popup is a scrollable overlay that displays help text.
type Popup struct {
	Theme    *uxtypes.Theme
	Viewport viewport.Model
	Visible  bool
	Width    int
	Height   int
}

// New creates a new Popup.
func New(theme *uxtypes.Theme) Popup {
	vp := viewport.New(0, 0)
	vp.SetContent(helpText)

	return Popup{Theme: theme, Viewport: vp}
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

// Show makes the help popup visible with the general help text.
func (h *Popup) Show() {
	h.Viewport.SetContent(helpText)
	h.Visible = true
	h.Viewport.GotoTop()
}

// ShowContextual makes the help popup visible with context-specific help.
// Falls back to general help if no contextual help is defined for the given context.
func (h *Popup) ShowContextual(context string) {
	text, ok := contextHelp[context]
	if !ok {
		text = helpText
	}

	h.Viewport.SetContent(text)
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
		BorderForeground(h.Theme.Secondary).
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

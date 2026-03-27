package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
)

var _ tea.Model = &Model{}

// pane identifies which UI pane currently has focus.
type pane int

const (
	paneRepos pane = iota // left panel
	paneRight             // right panel (tabbed: Alerts / Actions)
	paneInput             // command input
)

const paneCount = 3

// Model represents the top-level state of the TUI.
type Model struct {
	cfg *config.Config

	repos  reposPanel
	right  rightPanel
	input  commandInput
	status statusBar
	help   helpPopup
	wizard configWizard

	focused  pane
	width    int
	height   int
	quitting bool
}

// New creates the initial state for the TUI app.
//
// If cfg is nil a zero-value configuration is used.
func New(cfg *config.Config) *Model {
	if cfg == nil {
		cfg = &config.Config{}
	}

	m := &Model{
		cfg:     cfg,
		repos:   newReposPanel(cfg),
		right:   newRightPanel(),
		input:   newCommandInput(),
		status:  newStatusBar(),
		help:    newHelpPopup(),
		wizard:  newConfigWizard(cfg),
		focused: paneRepos,
	}

	return m
}

// Init is called when the program starts.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.WindowSize()}

	if m.cfg.IsEmpty() {
		// No configuration found — launch the wizard automatically.
		cmds = append(cmds, m.wizard.Show())
	} else {
		// Config exists — scan the configured roots in the background.
		cmds = append(cmds, scanRoots(m.cfg))
		m.status.SetMessage("Scanning configured roots...")
	}

	return tea.Batch(cmds...)
}

// Update handles incoming messages and updates the [Model].
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()

		return m, nil

	case commandResult:
		m.status.SetMessage(msg.output)

		return m, nil

	case scanResultMsg:
		return m.handleScanResult(msg)

	case tea.KeyMsg:
		// When the wizard is visible, it captures all keys except quit.
		if m.wizard.visible {
			return m.handleWizardKey(msg)
		}

		return m.handleKey(msg)

	case tea.MouseMsg:
		// Dismiss wizard on click when visible.
		if m.wizard.visible {
			if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
				// Let clicks pass to the wizard (don't dismiss on click).
				cmd, result := m.wizard.Update(msg)
				if result != nil {
					return m.handleWizardDone(result)
				}

				return m, cmd
			}

			return m, nil
		}

		return m.handleMouse(msg)
	}

	// When wizard is visible, forward other messages to it.
	if m.wizard.visible {
		cmd, result := m.wizard.Update(msg)
		if result != nil {
			return m.handleWizardDone(result)
		}

		return m, cmd
	}

	// Forward to focused pane.
	cmd := m.updateFocused(msg)

	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys that work regardless of focus.
	switch key {
	case "ctrl+c", "ctrl+q":
		m.quitting = true

		return m, tea.Quit

	case "ctrl+h":
		m.help.Toggle()

		return m, nil

	case "ctrl+a":
		// Cycle tabs in the focused panel.
		switch m.focused {
		case paneRepos:
			m.repos.CycleTab()
			m.status.SetMessage("Root: " + m.repos.ActiveTabName())
		case paneRight:
			m.right.CycleTab()
			m.status.SetMessage("Tab: " + m.right.ActiveTabName())
		}

		return m, nil
	}

	// When the help popup is visible, it captures all keys.
	if m.help.visible {
		return m.handleHelpKey(msg)
	}

	switch key {
	case "tab":
		m.cycleFocus(1)

		return m, m.applyFocus()

	case "shift+tab":
		m.cycleFocus(-1)

		return m, m.applyFocus()
	}

	// When the input pane is focused, most keys go to the text input.
	if m.focused == paneInput {
		return m.handleInputKey(msg)
	}

	// Panel-level shortcuts (only when a panel is focused).
	switch key {
	case "/":
		// Jump to input pane on "/" for quick command entry.
		m.focused = paneInput
		m.input.input.SetValue("/")

		return m, m.applyFocus()

	case "q":
		m.quitting = true

		return m, tea.Quit

	case "right", "l":
		switch m.focused {
		case paneRepos:
			m.repos.CycleTab()
			m.status.SetMessage("Root: " + m.repos.ActiveTabName())

			return m, nil
		case paneRight:
			m.right.CycleTab()
			m.status.SetMessage("Tab: " + m.right.ActiveTabName())

			return m, nil
		}

	case "left", "h":
		switch m.focused {
		case paneRepos:
			m.repos.CycleTabBack()
			m.status.SetMessage("Root: " + m.repos.ActiveTabName())

			return m, nil
		case paneRight:
			m.right.CycleTabBack()
			m.status.SetMessage("Tab: " + m.right.ActiveTabName())

			return m, nil
		}
	}

	// Forward to the focused panel.
	cmd := m.updateFocused(msg)

	return m, cmd
}

func (m *Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.help.Hide()

		return m, nil
	}

	// Forward scroll keys (j/k/up/down/pgup/pgdn) to the viewport.
	cmd := m.help.Update(msg)

	return m, cmd
}

func (m *Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := m.input.Submit()
		result := executeCommand(cmd)

		switch result {
		case "quit":
			m.quitting = true

			return m, tea.Quit
		case commandShowHelp:
			m.help.Show()

			return m, nil
		case commandShowConfig:
			wizCmd := m.wizard.Show()
			m.wizard.SetSize(m.width, m.height)

			return m, wizCmd
		case commandScanRoots:
			if m.cfg.IsEmpty() {
				m.status.SetMessage("No roots configured. Use /config to add one.")

				return m, nil
			}

			m.status.SetMessage("Scanning configured roots...")

			return m, scanRoots(m.cfg)
		case "":
			// empty input, nothing to do
		default:
			m.status.SetMessage(result)
		}

		return m, nil

	case "esc":
		m.input.input.SetValue("")
		m.focused = paneRepos

		return m, m.applyFocus()
	}

	cmd := m.input.Update(msg)

	return m, cmd
}

// cycleFocus moves focus forward or backward through panes.
func (m *Model) cycleFocus(dir int) {
	m.focused = pane((int(m.focused) + dir + paneCount) % paneCount)
}

// applyFocus ensures only the focused pane receives input events.
func (m *Model) applyFocus() tea.Cmd {
	if m.focused == paneInput {
		return m.input.Focus()
	}

	m.input.Blur()

	return nil
}

// updateFocused forwards a message to the currently focused pane.
func (m *Model) updateFocused(msg tea.Msg) tea.Cmd {
	switch m.focused {
	case paneRepos:
		return m.repos.Update(msg)
	case paneRight:
		return m.right.Update(msg)
	case paneInput:
		return m.input.Update(msg)
	}

	return nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Only handle left-button press events.
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}

	// Dismiss help popup on click anywhere when visible.
	if m.help.visible {
		m.help.Hide()

		return m, nil
	}

	x, y := msg.X, msg.Y

	// Compute zone boundaries (must match recalcLayout).
	const inputHeight = 3
	halfWidth := m.width / 2
	panelHeight := m.height - inputHeight - 1 // 1 for status bar
	if panelHeight < 4 {
		panelHeight = 4
	}

	switch {
	case y < panelHeight && x < halfWidth:
		// Repos panel.
		m.focused = paneRepos

		// Tab bar is at row 1 inside the border (row 0 is the border top).
		if y == 1 {
			localX := x - 1 // -1 for left border column
			if tab := m.repos.TabAtX(localX); tab >= 0 {
				m.repos.SetTab(tab)
				m.status.SetMessage("Root: " + m.repos.ActiveTabName())
			}
		}

		return m, m.applyFocus()

	case y < panelHeight && x >= halfWidth:
		// Right panel — check if click lands on a tab label.
		m.focused = paneRight

		// Tab bar is at row 1 inside the border (row 0 is the border top).
		if y == 1 {
			localX := x - halfWidth - 1 // -1 for left border column
			if tab := m.right.TabAtX(localX); tab >= 0 {
				m.right.SetTab(tab)
				m.status.SetMessage("Tab: " + m.right.ActiveTabName())
			}
		}

		return m, m.applyFocus()

	case y >= panelHeight && y < panelHeight+inputHeight:
		// Input zone.
		m.focused = paneInput

		return m, m.applyFocus()
	}

	return m, nil
}

// handleWizardKey routes key events to the wizard overlay.
func (m *Model) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit still works.
	if key == "ctrl+c" || key == "ctrl+q" {
		m.quitting = true

		return m, tea.Quit
	}

	cmd, result := m.wizard.Update(msg)
	if result != nil {
		return m.handleWizardDone(result)
	}

	return m, cmd
}

// handleWizardDone is called when the wizard finishes successfully.
// It updates the config, rebuilds tabs, triggers a scan, and closes the wizard.
func (m *Model) handleWizardDone(result *configWizardMsg) (tea.Model, tea.Cmd) {
	m.cfg = result.cfg
	m.wizard.Hide()
	m.repos.rebuildTabs(m.cfg)
	m.repos.SetSize(m.repos.width, m.repos.height)

	if len(m.cfg.Roots) > 0 {
		m.status.SetMessage("Configuration saved. Scanning roots...")

		return m, scanRoots(m.cfg)
	}

	m.status.SetMessage("Configuration saved.")

	return m, nil
}

// handleScanResult processes the result of a background repository scan.
func (m *Model) handleScanResult(msg scanResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status.SetMessage("Scan error: " + msg.err.Error())

		return m, nil
	}

	var cmds []tea.Cmd

	total := 0

	for rootIdx, repos := range msg.reposByRoot {
		items := repoItemsToListItems(repos)
		if cmd := m.repos.SetRootItems(rootIdx, items); cmd != nil {
			cmds = append(cmds, cmd)
		}

		total += len(repos)
	}

	m.status.SetMessage(fmt.Sprintf("Found %d repositories across %d roots.", total, len(msg.reposByRoot)))

	return m, tea.Batch(cmds...)
}

// recalcLayout distributes available space among the panes.
func (m *Model) recalcLayout() {
	// Reserve space: 3 lines for input border, 1 line for status bar.
	const inputHeight = 3
	const statusHeight = 1

	panelHeight := m.height - inputHeight - statusHeight
	if panelHeight < 4 {
		panelHeight = 4
	}

	halfWidth := m.width / 2

	m.repos.SetSize(halfWidth, panelHeight)
	m.right.SetSize(m.width-halfWidth, panelHeight)
	m.input.SetSize(m.width, inputHeight)
	m.status.SetSize(m.width)
	m.help.SetSize(m.width, m.height)
	m.wizard.SetSize(m.width, m.height)
}

// View renders the entire UI.
func (m *Model) View() string {
	if m.quitting {
		return "Thanks for using git-janitor!\n"
	}

	// Top row: two panels side by side.
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.repos.View(m.focused == paneRepos),
		m.right.View(m.focused == paneRight),
	)

	// Stack: panels, input, status bar.
	base := lipgloss.JoinVertical(
		lipgloss.Left,
		panels,
		m.input.View(m.focused == paneInput),
		m.status.View(),
	)

	// Overlay the help popup when visible.
	if m.help.visible {
		return m.help.View(m.width, m.height)
	}

	// Overlay the config wizard when visible.
	if m.wizard.visible {
		return m.wizard.View(m.width, m.height)
	}

	return base
}

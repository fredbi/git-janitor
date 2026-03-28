package ux

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/git"
	"github.com/fredbi/git-janitor/internal/ux/commands"
	wizard "github.com/fredbi/git-janitor/internal/ux/commands/config-wizard"
	"github.com/fredbi/git-janitor/internal/ux/commands/help"
	"github.com/fredbi/git-janitor/internal/ux/commands/scan"
	themecmd "github.com/fredbi/git-janitor/internal/ux/commands/theme"
	"github.com/fredbi/git-janitor/internal/ux/panels/infos"
	"github.com/fredbi/git-janitor/internal/ux/panels/repos"
	"github.com/fredbi/git-janitor/internal/ux/statusbar"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

var _ tea.Model = &Model{}

// Pane identifies which UI Pane currently has focus.
type Pane int

const (
	paneRepos Pane = iota // left panel
	paneRight             // right panel (tabbed: Alerts / Actions)
	paneInput             // Command input
)

const paneCount = 3

// Model represents the top-level state of the TUI.
type Model struct {
	Cfg *config.Config

	Repos  repos.Panel
	Right  infos.Panel
	Input  commands.Input
	Status statusbar.StatusBar
	Help   help.Popup
	Wizard wizard.ConfigWizard

	Theme        *uxtypes.Theme
	Focused      Pane
	Width        int
	Height       int
	Quitting     bool
	SelectedRepo string // path of the currently selected repo (to detect changes)
}

// New creates the initial state for the TUI app.
//
// If cfg is nil a zero-value configuration is used.
func New(cfg *config.Config) *Model {
	if cfg == nil {
		cfg = &config.Config{}
	}

	t := themecmd.Default
	uxtypes.CurrentTheme = &t

	m := &Model{
		Cfg:     cfg,
		Theme:   &t,
		Repos:   repos.New(cfg),
		Right:   infos.New(),
		Input:   commands.New(),
		Status:  statusbar.New(),
		Help:    help.New(),
		Wizard:  wizard.New(cfg),
		Focused: paneRepos,
	}

	return m
}

// Init is called when the program starts.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.WindowSize(), m.applyFocus()}

	if m.Cfg.IsEmpty() {
		// No configuration found — launch the wizard automatically.
		cmds = append(cmds, m.Wizard.Show())
	} else {
		// Config exists — scan the configured roots in the background.
		cmds = append(cmds, scan.Roots(m.Cfg))
		m.Status.SetMessage("Scanning configured roots...")
	}

	return tea.Batch(cmds...)
}

// Update handles incoming messages and updates the [Model].
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.recalcLayout()

		return m, nil

	case uxtypes.CommandResult:
		m.Status.SetMessage(msg.Output)

		return m, nil

	case uxtypes.RepoInfoMsg:
		return m.handleRepoInfo(msg)

	case uxtypes.RepoRefreshMsg:
		return m.handleRepoRefresh(msg)

	case uxtypes.ScanResultMsg:
		return m.handleScanResult(msg)

	case statusbar.TickMsg:
		cmd, consumed := m.Status.Update(msg)
		if consumed {
			return m, cmd
		}

		return m, nil

	case tea.KeyMsg:
		// When the wizard is visible, it captures all keys except quit.
		if m.Wizard.Visible {
			return m.handleWizardKey(msg)
		}

		return m.handleKey(msg)

	case tea.MouseMsg:
		// Dismiss wizard on click when visible.
		if m.Wizard.Visible {
			if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
				// Let clicks pass to the wizard (don't dismiss on click).
				cmd, result := m.Wizard.Update(msg)
				if result != nil {
					return m.handleWizardDone(result)
				}

				return m, cmd
			}

			return m, nil
		}

		return m.handleMouse(msg)
	}

	// Forward progress animation frames to the status bar.
	if cmd, consumed := m.Status.Update(msg); consumed {
		return m, cmd
	}

	// When wizard is visible, forward other messages to it.
	if m.Wizard.Visible {
		cmd, result := m.Wizard.Update(msg)
		if result != nil {
			return m.handleWizardDone(result)
		}

		return m, cmd
	}

	// Forward to focused Pane.
	cmd := m.updateFocused(msg)

	// Check if repo selection changed after forwarding to the repos panel.
	if m.Focused == paneRepos {
		if fetchCmd := m.checkSelectedRepo(); fetchCmd != nil {
			return m, tea.Batch(cmd, fetchCmd)
		}
	}

	return m, cmd
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys that work regardless of focus.
	switch key {
	case "ctrl+c", "ctrl+q":
		m.Quitting = true

		return m, tea.Quit

	case "ctrl+h":
		m.Help.Toggle()

		return m, nil

	case "ctrl+r":
		// Refresh: fetch --all --tags on the selected repo, then re-inspect.
		repo, ok := m.Repos.SelectedRepo()
		if !ok {
			m.Status.SetMessage("No repository selected.")

			return m, nil
		}

		progressCmd := m.Status.StartProgress("Fetching " + repo.Name + "...")

		return m, tea.Batch(progressCmd, refreshRepo(repo.Path))

	case "ctrl+a":
		// Cycle tabs in the focused panel.
		switch m.Focused {
		case paneRepos:
			m.Repos.CycleTab()
			m.Status.SetMessage("Root: " + m.Repos.ActiveTabName())

			return m, m.checkSelectedRepo()
		case paneRight:
			m.Right.CycleTab()
			m.Status.SetMessage("Tab: " + m.Right.ActiveTabName())
		}

		return m, nil
	}

	// When the help popup is visible, it captures all keys.
	if m.Help.Visible {
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

	// When the input Pane is focused, most keys go to the text input.
	if m.Focused == paneInput {
		return m.handleInputKey(msg)
	}

	// Panel-level shortcuts (only when a panel is focused).
	switch key {
	case "/":
		// Jump to input Pane on "/" for quick Command entry.
		m.Focused = paneInput
		m.Input.Input.SetValue("/")

		return m, m.applyFocus()

	case "q":
		m.Quitting = true

		return m, tea.Quit

	case "right", "l":
		switch m.Focused {
		case paneRepos:
			m.Repos.CycleTab()
			m.Status.SetMessage("Root: " + m.Repos.ActiveTabName())

			return m, m.checkSelectedRepo()
		case paneRight:
			m.Right.CycleTab()
			m.Status.SetMessage("Tab: " + m.Right.ActiveTabName())

			return m, nil
		}

	case "left", "h":
		switch m.Focused {
		case paneRepos:
			m.Repos.CycleTabBack()
			m.Status.SetMessage("Root: " + m.Repos.ActiveTabName())

			return m, m.checkSelectedRepo()
		case paneRight:
			m.Right.CycleTabBack()
			m.Status.SetMessage("Tab: " + m.Right.ActiveTabName())

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
		m.Help.Hide()

		return m, nil
	}

	// Forward scroll keys (j/k/up/down/pgup/pgdn) to the viewport.
	cmd := m.Help.Update(msg)

	return m, cmd
}

func (m *Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		cmd := m.Input.Submit()
		result := commands.ExecuteCommand(cmd)

		switch result {
		case "quit":
			m.Quitting = true

			return m, tea.Quit
		case commands.CommandShowHelp:
			m.Help.Show()

			return m, nil
		case commands.CommandShowConfig:
			wizCmd := m.Wizard.Show()
			m.Wizard.SetSize(m.Width, m.Height)

			return m, wizCmd
		case commands.CommandScanRoots:
			if m.Cfg.IsEmpty() {
				m.Status.SetMessage("No roots configured. Use /config to add one.")

				return m, nil
			}

			m.Status.SetMessage("Scanning configured roots...")

			return m, scan.Roots(m.Cfg)
		case "":
			// empty input, nothing to do
		default:
			if strings.HasPrefix(result, commands.CommandThemePrefix) {
				themeName := strings.TrimPrefix(result, commands.CommandThemePrefix)
				if themeName == "list" {
					m.Status.SetMessage("Available themes: " + strings.Join(themecmd.Names(), ", "))

					return m, nil
				}

				m.applyThemeCommand(themeName)

				return m, nil
			}

			m.Status.SetMessage(result)
		}

		return m, nil

	case "esc":
		m.Input.Input.SetValue("")
		m.Focused = paneRepos

		return m, m.applyFocus()
	}

	cmd := m.Input.Update(msg)

	return m, cmd
}

// cycleFocus moves focus forward or backward through panes.
func (m *Model) cycleFocus(dir int) {
	m.Focused = Pane((int(m.Focused) + dir + paneCount) % paneCount)
}

// applyFocus ensures only the focused Pane receives input events.
func (m *Model) applyFocus() tea.Cmd {
	switch m.Focused {
	case paneInput:
		m.Repos.Blur()

		return m.Input.Focus()
	case paneRepos:
		m.Input.Blur()

		return m.Repos.Focus()
	default:
		m.Input.Blur()
		m.Repos.Blur()

		return nil
	}
}

// updateFocused forwards a message to the currently focused Pane.
func (m *Model) updateFocused(msg tea.Msg) tea.Cmd {
	switch m.Focused {
	case paneRepos:
		return m.Repos.Update(msg)
	case paneRight:
		return m.Right.Update(msg)
	case paneInput:
		return m.Input.Update(msg)
	}

	return nil
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Only handle left-button press events.
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}

	// Dismiss help popup on click anywhere when visible.
	if m.Help.Visible {
		m.Help.Hide()

		return m, nil
	}

	x, y := msg.X, msg.Y

	// Compute zone boundaries (must match recalcLayout).
	const inputHeight = 3
	halfWidth := m.Width / 2
	panelHeight := m.Height - inputHeight - 1 // 1 for status bar
	if panelHeight < 4 {
		panelHeight = 4
	}

	switch {
	case y < panelHeight && x < halfWidth:
		// Repos panel.
		m.Focused = paneRepos

		// Tab bar is at row 1 inside the border (row 0 is the border top).
		if y == 1 {
			localX := x - 1 // -1 for left border column
			if tab := m.Repos.TabAtX(localX); tab >= 0 {
				m.Repos.SetTab(tab)
				m.Status.SetMessage("Root: " + m.Repos.ActiveTabName())
			}
		}

		return m, m.applyFocus()

	case y < panelHeight && x >= halfWidth:
		// Right panel — check if click lands on a tab label.
		m.Focused = paneRight

		// Tab bar is at row 1 inside the border (row 0 is the border top).
		if y == 1 {
			localX := x - halfWidth - 1 // -1 for left border column
			if tab := m.Right.TabAtX(localX); tab >= 0 {
				m.Right.SetTab(tab)
				m.Status.SetMessage("Tab: " + m.Right.ActiveTabName())
			}
		}

		return m, m.applyFocus()

	case y >= panelHeight && y < panelHeight+inputHeight:
		// Input zone.
		m.Focused = paneInput

		return m, m.applyFocus()
	}

	return m, nil
}

// handleWizardKey routes key events to the wizard overlay.
func (m *Model) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit still works.
	if key == "ctrl+c" || key == "ctrl+q" {
		m.Quitting = true

		return m, tea.Quit
	}

	cmd, result := m.Wizard.Update(msg)
	if result != nil {
		return m.handleWizardDone(result)
	}

	return m, cmd
}

// handleWizardDone is called when the wizard finishes successfully.
// It updates the config, rebuilds tabs, triggers a scan, and closes the wizard.
func (m *Model) handleWizardDone(result *uxtypes.ConfigWizardMsg) (tea.Model, tea.Cmd) {
	m.Cfg = result.Cfg
	m.Wizard.Hide()
	m.Repos.RebuildTabs(m.Cfg)
	m.Repos.SetSize(m.Repos.Width, m.Repos.Height)

	if len(m.Cfg.Roots) > 0 {
		m.Status.SetMessage("Configuration saved. Scanning roots...")

		return m, scan.Roots(m.Cfg)
	}

	m.Status.SetMessage("Configuration saved.")

	return m, nil
}

// handleScanResult processes the result of a background repository scan.
func (m *Model) handleScanResult(msg uxtypes.ScanResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.Status.SetMessage("Scan error: " + msg.Err.Error())

		return m, nil
	}

	var cmds []tea.Cmd

	total := 0

	for rootIdx, repoItems := range msg.ReposByRoot {
		items := repos.RepoItemsToListItems(repoItems)
		if cmd := m.Repos.SetRootItems(rootIdx, items); cmd != nil {
			cmds = append(cmds, cmd)
		}

		total += len(repoItems)
	}

	m.Status.SetMessage(fmt.Sprintf("Found %d repositories across %d roots.", total, len(msg.ReposByRoot)))

	// Trigger repo info fetch for the initially selected repo.
	if fetchCmd := m.checkSelectedRepo(); fetchCmd != nil {
		cmds = append(cmds, fetchCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleRepoInfo processes the result of a background repo info fetch.
func (m *Model) handleRepoInfo(msg uxtypes.RepoInfoMsg) (tea.Model, tea.Cmd) {
	// Only apply if this is still the selected repo.
	if msg.Info.Path != m.SelectedRepo {
		return m, nil
	}

	ti := &msg.Info
	m.Right.SetRepoInfo(ti)

	return m, nil
}

// handleRepoRefresh processes the result of a fetch + re-inspect.
func (m *Model) handleRepoRefresh(msg uxtypes.RepoRefreshMsg) (tea.Model, tea.Cmd) {
	if msg.Info.Path != m.SelectedRepo {
		return m, nil
	}

	if msg.Info.Err != nil {
		m.Status.SetMessage("Fetch error: " + msg.Info.Err.Error())
	} else {
		m.Status.SetMessage("Fetched " + msg.Info.Path)
	}

	ti := &msg.Info
	m.Right.SetRepoInfo(ti)

	return m, nil
}

// checkSelectedRepo detects when the selected repo changes and triggers a fetch.
func (m *Model) checkSelectedRepo() tea.Cmd {
	repo, ok := m.Repos.SelectedRepo()
	if !ok {
		return nil
	}

	if repo.Path == m.SelectedRepo {
		return nil
	}

	m.SelectedRepo = repo.Path

	return fetchRepoInfo(repo.Path, repo.IsGit)
}

// applyThemeCommand handles the /theme Command.
// If name is empty, it cycles to the next theme; otherwise it sets the named theme.
func (m *Model) applyThemeCommand(name string) {
	var t uxtypes.Theme

	if name == "" {
		t = themecmd.Next(m.Theme.Name)
	} else {
		found := themecmd.Find(name)
		if found == nil {
			m.Status.SetMessage("Unknown Theme: " + name + ". Available: " + strings.Join(themecmd.Names(), ", "))

			return
		}

		t = *found
	}

	m.Theme = &t
	uxtypes.CurrentTheme = m.Theme
	m.Repos.RebuildDelegate()
	m.Status.SetMessage("Theme: " + t.Name)
}

// recalcLayout distributes available space among the panes.
func (m *Model) recalcLayout() {
	// Reserve space: 3 lines for input border, 1 line for status bar.
	const inputHeight = 3
	const statusHeight = 1

	panelHeight := m.Height - inputHeight - statusHeight
	if panelHeight < 4 {
		panelHeight = 4
	}

	halfWidth := m.Width / 2

	m.Repos.SetSize(halfWidth, panelHeight)
	m.Right.SetSize(m.Width-halfWidth, panelHeight)
	m.Input.SetSize(m.Width, inputHeight)
	m.Status.SetSize(m.Width)
	m.Help.SetSize(m.Width, m.Height)
	m.Wizard.SetSize(m.Width, m.Height)
}

// fetchRepoInfo runs git commands in the background and returns a RepoInfoMsg.
func fetchRepoInfo(path string, isGit bool) tea.Cmd {
	return func() tea.Msg {
		if !isGit {
			return uxtypes.RepoInfoMsg{Info: git.RepoInfo{
				Path:  path,
				IsGit: false,
				SCM:   git.SCMNone,
				Kind:  git.KindNotGit,
			}}
		}

		ctx := context.Background()
		r := git.NewRunner(path)
		info := git.CollectRepoInfo(ctx, r, path)

		return uxtypes.RepoInfoMsg{Info: info}
	}
}

// refreshRepo runs git fetch --all --tags then re-inspects, returning a RepoRefreshMsg.
func refreshRepo(path string) tea.Cmd {
	return func() tea.Msg {
		info := git.RefreshRepo(context.Background(), path)

		return uxtypes.RepoRefreshMsg{Info: info}
	}
}

// View renders the entire UI.
func (m *Model) View() string {
	if m.Quitting {
		return "Thanks for using git-janitor!\n"
	}

	// Top row: two panels side by side.
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.Repos.View(m.Focused == paneRepos),
		m.Right.View(m.Focused == paneRight),
	)

	// Stack: panels, input, status bar.
	base := lipgloss.JoinVertical(
		lipgloss.Left,
		panels,
		m.Input.View(m.Focused == paneInput),
		m.Status.View(),
	)

	// Overlay the help popup when visible.
	if m.Help.Visible {
		return m.Help.View(m.Width, m.Height)
	}

	// Overlay the config wizard when visible.
	if m.Wizard.Visible {
		return m.Wizard.View(m.Width, m.Height)
	}

	return base
}

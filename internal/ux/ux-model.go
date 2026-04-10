package ux

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/commands"
	wizard "github.com/fredbi/git-janitor/internal/ux/commands/config-wizard"
	"github.com/fredbi/git-janitor/internal/ux/commands/help"
	"github.com/fredbi/git-janitor/internal/ux/commands/scan"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	"github.com/fredbi/git-janitor/internal/ux/key"
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

const (
	paneCount = 3
	app       = "git-janitor"
)

// Model represents the top-level state of the TUI.
type Model struct {
	options

	Repos  repos.Panel
	Right  infos.Panel
	Input  commands.Input
	Status statusbar.StatusBar
	Help   help.Popup
	Detail gadgets.DetailPopup
	Wizard wizard.ConfigWizard

	Focused      Pane
	Width        int
	Height       int
	Quitting     bool
	SelectedRepo string           // path of the currently selected repo (to detect changes)
	SelectedRoot int              // index of the root containing the selected repo
	LastRepoInfo *models.RepoInfo // most recent RepoInfo for the selected repo

	// PendingAction holds an action awaiting user confirmation (Y/N).
	// nil when no confirmation is pending.
	PendingAction *uxtypes.ExecuteActionMsg

	// forceGitHubRefresh is set after action execution to force a GitHub
	// re-fetch on the next handleRepoInfo cycle (avoids using stale data).
	forceGitHubRefresh bool
}

// New creates the initial state for the TUI app.
//
// If cfg is nil a zero-value configuration is used.
func New(opts ...Option) *Model {
	o := applyOptionsWithDefaults(opts)
	m := &Model{
		options: o,
		Focused: paneRepos,
	}

	theme := &m.Theme
	m.Repos = repos.New(o.Cfg, theme)
	m.Right = infos.New(o.Engine, theme)
	m.Input = commands.New(theme)
	m.Status = statusbar.New(theme)
	m.Help = help.New(theme)
	m.Detail = gadgets.NewDetailPopup(theme)
	m.Wizard = wizard.New(o.Cfg)

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

	case uxtypes.ExecuteActionMsg:
		return m.handleExecuteAction(msg)

	case uxtypes.GitHubInfoMsg:
		return m.handleGitHubInfo(msg)

	case uxtypes.CopyToClipboardMsg:
		if err := gadgets.CopyToClipboard(context.Background(), msg.Text); err != nil { // TODO: command context?
			m.Status.SetMessagef("Clipboard: %v", err)
		} else {
			m.Status.SetMessage("Copied to clipboard")
		}

		return m, nil

	case uxtypes.ActionResultMsg:
		return m.handleActionResult(msg)

	case uxtypes.ShowDetailMsg:
		m.Detail.Show(msg.Title, msg.Content)

		return m, nil

	case uxtypes.FetchDetailMsg:
		return m, m.fetchDetail(msg.Scope)

	case uxtypes.ActivityDataMsg:
		m.Right.SetActivityData(msg.Info)

		return m, nil

	case uxtypes.ShowSuggestionsMsg:
		// Forward to the right panel — it switches to Actions tab.
		cmd := m.Right.Update(msg)

		return m, cmd

	case statusbar.TickMsg:
		cmd, consumed := m.Status.Update(msg)
		if consumed {
			return m, cmd
		}

		return m, nil

	case tea.KeyMsg:
		// When a confirmation is pending, intercept Y/N.
		if m.PendingAction != nil {
			return m.handleConfirmKey(msg)
		}

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
	kb := key.MsgBinding(msg)

	// When the right panel has an active text input (e.g. param prompt),
	// forward all keys directly except quit — the input needs to capture
	// letters, arrows, etc. that would otherwise trigger panel navigation.
	if m.Focused == paneRight && m.Right.IsCapturingInput() {
		if kb == key.CtrlC || kb == key.CtrlQ {
			m.Quitting = true

			return m, tea.Quit
		}

		cmd := m.Right.Update(msg)

		return m, cmd
	}

	// Global keys that work regardless of focus.
	switch kb {
	case key.CtrlC, key.CtrlQ:
		m.Quitting = true

		return m, tea.Quit

	case key.CtrlH:
		if m.Help.Visible {
			m.Help.Hide()
		} else {
			m.Help.ShowContextual(m.helpContext())
		}

		return m, nil

	case key.CtrlD:
		// Show full status message in a detail popup.
		m.Detail.Show("Status Details", m.Status.FullMessage())

		return m, nil

	case key.CtrlR:
		// Refresh: fetch --all --tags on the selected repo, then re-inspect.
		repo, ok := m.Repos.SelectedRepo()
		if !ok {
			m.Status.SetMessage("No repository selected.")

			return m, nil
		}

		if !repo.IsGit {
			m.Status.SetMessage("Not a git repository — nothing to refresh.")

			return m, nil
		}

		progressCmd := m.Status.StartProgress("Fetching " + repo.Name + "...")

		return m, tea.Batch(progressCmd, m.refreshRepo(repo.Path))

	case key.CtrlA:
		// Cycle tabs in the focused panel.
		switch m.Focused {
		case paneRepos:
			m.Repos.CycleTab()
			m.Status.SetMessage("Root: " + m.Repos.ActiveTabName())

			return m, m.forceRepoCheck()
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

	// When the detail popup is visible, it captures all keys.
	if m.Detail.Visible {
		return m.handleDetailKey(msg)
	}

	switch kb {
	case key.Tab:
		m.cycleFocus(1)

		return m, tea.Batch(m.applyFocus(), m.checkSelectedRepo())

	case key.ShiftTab:
		m.cycleFocus(-1)

		return m, tea.Batch(m.applyFocus(), m.checkSelectedRepo())
	}

	// When the input Pane is focused, most keys go to the text input.
	if m.Focused == paneInput {
		return m.handleInputKey(msg)
	}

	// Panel-level shortcuts (only when a panel is focused).
	switch kb {
	case key.Slash:
		// Jump to input Pane on "/" for quick Command entry.
		m.Focused = paneInput
		m.Input.Input.SetValue("/")

		return m, m.applyFocus()

	case key.RightArrow, key.L:
		switch m.Focused {
		case paneRepos:
			m.Repos.CycleTab()
			m.Status.SetMessage("Root: " + m.Repos.ActiveTabName())

			return m, m.forceRepoCheck()
		case paneRight:
			m.Right.CycleTab()
			m.Status.SetMessage("Tab: " + m.Right.ActiveTabName())

			return m, nil
		}

	case key.LeftArrow, key.H:
		switch m.Focused {
		case paneRepos:
			m.Repos.CycleTabBack()
			m.Status.SetMessage("Root: " + m.Repos.ActiveTabName())

			return m, m.forceRepoCheck()
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
	if key.MsgBinding(msg).ClosePopup() {
		m.Help.Hide()

		return m, nil
	}

	// Forward scroll keys (j/k/up/down/pgup/pgdn) to the viewport.
	cmd := m.Help.Update(msg)

	return m, cmd
}

func (m *Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.MsgBinding(msg).ClosePopup() {
		m.Detail.Hide()

		return m, nil
	}

	if key.MsgBinding(msg) == key.C {
		if err := gadgets.CopyToClipboard(context.Background(), m.Detail.Content); err != nil {
			m.Status.SetMessagef("Copy failed: %v", err)
		} else {
			m.Status.SetMessage("Copied to clipboard")
		}

		m.Detail.Hide()

		return m, nil
	}

	// Forward scroll keys to the viewport.
	cmd := m.Detail.Update(msg)

	return m, cmd
}

func (m *Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.MsgBinding(msg) {
	case key.Enter:
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
		case commands.CommandShowChecks:
			m.Detail.Show("Registered Checks", m.buildCheckList())

			return m, nil
		case commands.CommandShowActions:
			m.Detail.Show("Registered Actions", m.buildActionList())

			return m, nil
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
					if m.themes == nil {
						m.Status.SetMessage("Available theme: " + m.Theme.Name())

						return m, nil
					}

					m.Status.SetMessage("Available themes: " + strings.Join(m.themes.Names(), ", "))

					return m, nil
				}

				m.applyThemeCommand(themeName)

				return m, nil
			}

			m.Status.SetMessage(result)
		}

		return m, nil

	case key.Esc:
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

// helpContext returns the context key for contextual help based on the
// currently focused pane and active tab.
func (m *Model) helpContext() string {
	switch m.Focused {
	case paneRepos:
		return "repos"
	case paneInput:
		return "input"
	case paneRight:
		switch m.Right.Active {
		case infos.TabFacts:
			return "facts"
		case infos.TabBranches:
			return "branches"
		case infos.TabStashes:
			return "stashes"
		case infos.TabAlerts:
			return "alerts"
		case infos.TabActions:
			return "actions"
		case infos.TabActivity:
			return "activity"
		case infos.TabRecent:
			return "recent"
		}
	}

	return ""
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

	// Dismiss popups on click anywhere when visible.
	if m.Help.Visible {
		m.Help.Hide()

		return m, nil
	}

	if m.Detail.Visible {
		m.Detail.Hide()

		return m, nil
	}

	x, y := msg.X, msg.Y

	// Compute zone boundaries (must match recalcLayout).
	const inputHeight = 3
	halfWidth := m.Width / 2
	panelHeight := m.Height - inputHeight - 2 // 2 for status bar (message + progress) //nolint:mnd // matches recalcLayout
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

	case y >= panelHeight+inputHeight:
		// Status bar zone — show full message if truncated.
		if m.Status.IsTruncated() {
			m.Detail.Show("Status Details", m.Status.FullMessage())
		}

		return m, nil
	}

	return m, nil
}

// handleWizardKey routes key events to the wizard overlay.
func (m *Model) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := key.MsgBinding(msg)

	// Global quit still works.
	if kb.Quit() {
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
	m.Engine.Reload(result.Cfg)
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
		if cmd := m.Repos.SetRootItems(rootIdx, repoItems); cmd != nil {
			cmds = append(cmds, cmd)
		}

		total += len(repoItems)
	}

	m.Status.SetMessagef("Found %d repositories across %d roots.", total, len(msg.ReposByRoot))

	// Trigger repo info fetch for the initially selected repo.
	if fetchCmd := m.checkSelectedRepo(); fetchCmd != nil {
		cmds = append(cmds, fetchCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleRepoInfo processes the result of a background repo info fetch.
func (m *Model) handleRepoInfo(msg uxtypes.RepoInfoMsg) (tea.Model, tea.Cmd) {
	info := msg.Info
	if info.IsEmpty() {
		return m, nil
	}

	// Only apply if this is still the selected repo.
	if info.Path != m.SelectedRepo {
		return m, nil
	}

	m.LastRepoInfo = info
	m.Right.SetRepoInfo(info) // TODO: m.Cfg.EnabledChecks(m.SelectedRoot)) // TODO: filter config w/ built-in

	// Trigger async GitHub fetch if applicable.
	// Force refresh if a recent action execution set the flag.
	forceRefresh := m.forceGitHubRefresh
	m.forceGitHubRefresh = false

	cmd := m.triggerGitHubFetch(info, forceRefresh)

	return m, cmd
}

// handleRepoRefresh processes the result of a fetch + re-inspect.
// If the fetch failed (remote unavailable), local data is still displayed.
func (m *Model) handleRepoRefresh(msg uxtypes.RepoRefreshMsg) (tea.Model, tea.Cmd) {
	info := msg.Info
	if info.IsEmpty() {
		return m, nil
	}

	if info.Path != m.SelectedRepo {
		return m, nil
	}

	if info.FetchErr != nil {
		m.Status.SetMessage("Remote unavailable — showing local data")
	} else {
		m.Status.SetMessage("Fetched " + msg.Info.Path)
	}

	// Always update panels with local data, even if fetch failed.
	m.LastRepoInfo = info
	m.Right.SetRepoInfo(info) // TODO: m.Cfg.EnabledChecks(m.SelectedRoot))

	// Trigger async GitHub fetch (force refresh on Ctrl+R).
	if cmd := m.triggerGitHubFetch(info, true); cmd != nil {
		return m, cmd
	}

	return m, nil
}

// handleGitHubInfo processes the result of an async GitHub API fetch.
func (m *Model) handleGitHubInfo(msg uxtypes.GitHubInfoMsg) (tea.Model, tea.Cmd) {
	// Only apply if this is still the selected repo.
	if msg.RepoPath != m.SelectedRepo {
		return m, nil
	}

	if msg.Data.Platform == nil && msg.Data.UpstreamPlatform == nil {
		return m, nil
	}

	if msg.Data.Platform != nil && msg.Data.Platform.Err != nil {
		m.Status.SetMessage("GitHub: " + msg.Data.Platform.Err.Error())
	}

	// The engine already injected LocalDefaultBranch during Collect.
	m.Right.SetGitHubData(msg.Data)

	return m, nil
}

// handleConfirmKey processes Y/N input when a confirmation is pending.
func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := key.MsgBinding(msg)
	switch {
	case kb.Confirm():
		pending := m.PendingAction
		m.PendingAction = nil

		return m, m.runAction(*pending)

	case kb.Cancel():
		m.PendingAction = nil
		m.Status.SetMessage("Action cancelled")

		return m, nil

	default:
		// Ignore other keys while confirmation is pending.
		return m, nil
	}
}

// handleActionResult processes the result of an executed action.
// On success, it performs a full re-collect (not fast) so that checks
// are re-evaluated against the updated repository state.
func (m *Model) handleActionResult(msg uxtypes.ActionResultMsg) (tea.Model, tea.Cmd) {
	if msg.OK {
		m.Status.SetMessagef("✓ %s: %s", msg.ActionName, msg.Message)
	} else {
		m.Status.SetMessagef("✗ %s: %s", msg.ActionName, msg.Message)
	}

	// Immediately refresh the Recent panel so the new action appears.
	m.Right.RefreshRecent()

	// Re-collect repo info (full, not fast) to reflect changes and re-evaluate alerts.
	// The resulting RepoInfoMsg will trigger a GitHub re-fetch via handleRepoInfo.
	// We don't fire triggerGitHubFetch here because m.LastRepoInfo is stale —
	// it still has the pre-action state. handleRepoInfo will use the fresh data.
	if msg.RepoPath == m.SelectedRepo {
		m.forceGitHubRefresh = m.LastRepoInfo != nil && m.LastRepoInfo.Platform != nil

		return m, m.fullRepoCheck(msg.RepoPath)
	}

	return m, nil
}

// applyThemeCommand handles the /theme Command.
// If name is empty, it cycles to the next theme; otherwise it sets the named theme.
func (m *Model) applyThemeCommand(name string) {
	if name == m.Theme.Name() || m.themes == nil {
		return
	}

	var th uxtypes.Theme
	if name == "" {
		th = m.themes.Next(m.Theme.Name())
	} else {
		var found bool
		th, found = m.themes.Get(name)
		if !found {
			m.Status.SetMessage("Unknown Theme: " + name + ". Available: " + strings.Join(m.themes.Names(), ", "))

			return
		}
	}

	m.Theme = th
	m.setTheme()
	m.Status.SetMessage("Theme: " + th.Name())
}

// recalcLayout distributes available space among the panes.
func (m *Model) recalcLayout() {
	// Reserve space: 3 lines for input border, 2 lines for status bar
	// (message line + progress bar line, always reserved to prevent layout shift).
	const inputHeight = 3
	const statusHeight = 2

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
	m.Detail.SetSize(m.Width, m.Height)
	m.Wizard.SetSize(m.Width, m.Height)
}

// View renders the entire UI.
func (m *Model) View() string {
	if m.Quitting {
		return "Thanks for using " + app + "!\n"
	}

	// Top row: two panels side by side.
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.Repos.View(m.Focused == paneRepos),
		m.Right.View(m.Focused == paneRight),
	)

	// Stack: panels, input, status bar.
	// Use direct concatenation to avoid JoinVertical width-padding
	// which could cause line wrapping.
	base := panels + "\n" + m.Input.View(m.Focused == paneInput) + "\n" + m.Status.View()

	// Final safety: clamp total output to exactly m.Height lines.
	// This prevents any overflow from pushing content off-screen.
	base = normalizeViewLines(base, m.Height)

	// Overlay the help popup when visible.
	if m.Help.Visible {
		return m.Help.View(m.Width, m.Height)
	}

	// Overlay the detail popup when visible.
	if m.Detail.Visible {
		return m.Detail.View(m.Width, m.Height)
	}

	// Overlay the config wizard when visible.
	if m.Wizard.Visible {
		return m.Wizard.View(m.Width, m.Height)
	}

	return base
}

// normalizeViewLines ensures a rendered string has exactly targetLines lines.
func normalizeViewLines(s string, targetLines int) string {
	lines := strings.Split(s, "\n")

	if len(lines) > targetLines {
		lines = lines[:targetLines]
	}

	for len(lines) < targetLines {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// buildCheckList returns a formatted string listing all registered checks.
func (m *Model) buildCheckList() string {
	if m.checks == nil {
		return "(no checks registered)"
	}

	type checkInfo struct {
		name string
		desc string
		kind string
	}

	var gitChecks, githubChecks []checkInfo

	for name, c := range m.checks.All() {
		info := checkInfo{name: name, desc: c.Description()}

		switch c.Kind() {
		case models.CheckKindGitHub:
			info.kind = "github"
			githubChecks = append(githubChecks, info)
		default:
			info.kind = "git"
			gitChecks = append(gitChecks, info)
		}
	}

	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.Theme.Dim)

	var b strings.Builder

	b.WriteString(bold.Render(fmt.Sprintf("Git Checks (%d)", len(gitChecks))) + "\n")
	b.WriteString(dim.Render(strings.Repeat("─", 40)) + "\n") //nolint:mnd // separator width

	for _, c := range gitChecks {
		b.WriteString("  " + bold.Render(c.name) + "\n")
		b.WriteString("    " + dim.Render(c.desc) + "\n")
	}

	b.WriteString("\n" + bold.Render(fmt.Sprintf("GitHub Checks (%d)", len(githubChecks))) + "\n")
	b.WriteString(dim.Render(strings.Repeat("─", 40)) + "\n") //nolint:mnd // separator width

	for _, c := range githubChecks {
		b.WriteString("  " + bold.Render(c.name) + "\n")
		b.WriteString("    " + dim.Render(c.desc) + "\n")
	}

	return b.String()
}

// buildActionList returns a formatted string listing all registered actions.
func (m *Model) buildActionList() string {
	if m.actions == nil {
		return "(no actions registered)"
	}

	type actionInfo struct {
		name string
		desc string
		kind string
	}

	var gitActions, githubActions []actionInfo

	for name, a := range m.actions.All() {
		info := actionInfo{name: name, desc: a.Description()}

		switch a.Kind() {
		case models.ActionKindGitHub:
			info.kind = "github"
			githubActions = append(githubActions, info)
		default:
			info.kind = "git"
			gitActions = append(gitActions, info)
		}
	}

	bold := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Foreground(m.Theme.Dim)
	warn := lipgloss.NewStyle().Foreground(m.Theme.Warning)

	var b strings.Builder

	b.WriteString(bold.Render(fmt.Sprintf("Git Actions (%d)", len(gitActions))) + "\n")
	b.WriteString(dim.Render(strings.Repeat("─", 40)) + "\n") //nolint:mnd // separator width

	for _, a := range gitActions {
		marker := " "
		if action, ok := m.actions.Get(a.name); ok && action.Destructive() {
			marker = warn.Render("!")
		}

		b.WriteString("  " + marker + " " + bold.Render(a.name) + "\n")
		b.WriteString("      " + dim.Render(a.desc) + "\n")
	}

	b.WriteString("\n" + bold.Render(fmt.Sprintf("GitHub Actions (%d)", len(githubActions))) + "\n")
	b.WriteString(dim.Render(strings.Repeat("─", 40)) + "\n") //nolint:mnd // separator width

	for _, a := range githubActions {
		marker := " "
		if action, ok := m.actions.Get(a.name); ok && action.Destructive() {
			marker = warn.Render("!")
		}

		b.WriteString("  " + marker + " " + bold.Render(a.name) + "\n")
		b.WriteString("      " + dim.Render(a.desc) + "\n")
	}

	b.WriteString("\n  " + warn.Render("!") + " = destructive (requires confirmation)\n")

	return b.String()
}

// setTheme propagates the current theme to all sub-components.
func (m *Model) setTheme() {
	theme := &m.Theme
	m.Repos.SetTheme(theme)
	m.Right.SetTheme(theme)
	m.Input.Theme = theme
	m.Status.Theme = theme
	m.Help.Theme = theme
	m.Detail.Theme = theme
}

package ux

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/engine/setup"
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
	Cfg    *config.Config
	Engine *engine.Engine

	Repos  repos.Panel
	Right  infos.Panel
	Input  commands.Input
	Status statusbar.StatusBar
	Help   help.Popup
	Wizard wizard.ConfigWizard

	Theme         *uxtypes.Theme
	Focused       Pane
	Width         int
	Height        int
	Quitting      bool
	SelectedRepo  string        // path of the currently selected repo (to detect changes)
	SelectedRoot  int           // index of the root containing the selected repo
	LastRepoInfo  *git.RepoInfo // most recent RepoInfo for the selected repo

	// PendingAction holds an action awaiting user confirmation (Y/N).
	// nil when no confirmation is pending.
	PendingAction *uxtypes.ExecuteActionMsg
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

	eng := setup.NewEngine()
	rightPanel := infos.New()
	rightPanel.Engine = eng

	m := &Model{
		Cfg:     cfg,
		Engine:  eng,
		Theme:   &t,
		Repos:   repos.New(cfg),
		Right:   rightPanel,
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

	case uxtypes.ExecuteActionMsg:
		return m.handleExecuteAction(msg)

	case uxtypes.ActionResultMsg:
		return m.handleActionResult(msg)

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

		if !repo.IsGit {
			m.Status.SetMessage("Not a git repository — nothing to refresh.")

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

	switch key {
	case "tab":
		m.cycleFocus(1)

		return m, tea.Batch(m.applyFocus(), m.checkSelectedRepo())

	case "shift+tab":
		m.cycleFocus(-1)

		return m, tea.Batch(m.applyFocus(), m.checkSelectedRepo())
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

			return m, m.forceRepoCheck()
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
	m.LastRepoInfo = ti
	m.Right.SetRepoInfo(ti, m.Cfg.EnabledChecks(m.SelectedRoot))

	return m, nil
}

// handleRepoRefresh processes the result of a fetch + re-inspect.
// If the fetch failed (remote unavailable), local data is still displayed.
func (m *Model) handleRepoRefresh(msg uxtypes.RepoRefreshMsg) (tea.Model, tea.Cmd) {
	if msg.Info.Path != m.SelectedRepo {
		return m, nil
	}

	if msg.Info.FetchErr != nil {
		m.Status.SetMessage("Remote unavailable — showing local data")
	} else {
		m.Status.SetMessage("Fetched " + msg.Info.Path)
	}

	// Always update panels with local data, even if fetch failed.
	ti := &msg.Info
	m.LastRepoInfo = ti
	m.Right.SetRepoInfo(ti, m.Cfg.EnabledChecks(m.SelectedRoot))

	return m, nil
}

// handleExecuteAction runs an action in the background.
// If the action requires confirmation (auto=false in config, or destructive),
// the status bar shows a Y/N prompt and the action is held in PendingAction.
func (m *Model) handleExecuteAction(msg uxtypes.ExecuteActionMsg) (tea.Model, tea.Cmd) {
	action, ok := m.Engine.Actions.Get(msg.ActionName)
	if !ok {
		m.Status.SetMessage(fmt.Sprintf("Unknown action: %s", msg.ActionName))

		return m, nil
	}

	// Check if confirmation is needed.
	needsConfirm := action.Destructive() || !m.Cfg.IsActionAuto(msg.ActionName)
	if needsConfirm {
		m.PendingAction = &msg

		label := "Run"
		if action.Destructive() {
			label = "⚠️  Run DESTRUCTIVE"
		}

		subjects := strings.Join(msg.Subjects, ", ")
		if len(subjects) > 40 {
			subjects = subjects[:37] + "..."
		}

		m.Status.SetMessage(fmt.Sprintf("%s action %q on %s?  [Y]es / [N]o", label, msg.ActionName, subjects))

		return m, nil
	}

	return m, m.runAction(msg)
}

// runAction executes an action in a background tea.Cmd.
func (m *Model) runAction(msg uxtypes.ExecuteActionMsg) tea.Cmd {
	m.Status.SetMessage(fmt.Sprintf("Running %s...", msg.ActionName))

	repoPath := msg.RepoPath
	actionName := msg.ActionName
	subjects := msg.Subjects
	eng := m.Engine
	info := m.LastRepoInfo

	return func() tea.Msg {
		r := git.NewRunner(repoPath)
		result, err := eng.Execute(
			context.Background(), r, info,
			engine.ActionSuggestion{
				ActionName: actionName,
				Subjects:   subjects,
			},
		)

		if err != nil {
			return uxtypes.ActionResultMsg{
				RepoPath:   repoPath,
				ActionName: actionName,
				OK:         false,
				Message:    err.Error(),
			}
		}

		return uxtypes.ActionResultMsg{
			RepoPath:   repoPath,
			ActionName: actionName,
			OK:         result.OK,
			Message:    result.Message,
		}
	}
}

// handleConfirmKey processes Y/N input when a confirmation is pending.
func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		pending := m.PendingAction
		m.PendingAction = nil

		return m, m.runAction(*pending)

	case "n", "N", "esc":
		m.PendingAction = nil
		m.Status.SetMessage("Action cancelled")

		return m, nil

	default:
		// Ignore other keys while confirmation is pending.
		return m, nil
	}
}

// handleActionResult processes the result of an executed action.
func (m *Model) handleActionResult(msg uxtypes.ActionResultMsg) (tea.Model, tea.Cmd) {
	if msg.OK {
		m.Status.SetMessage(fmt.Sprintf("✓ %s: %s", msg.ActionName, msg.Message))
	} else {
		m.Status.SetMessage(fmt.Sprintf("✗ %s: %s", msg.ActionName, msg.Message))
	}

	// Re-fetch the repo info to reflect changes.
	if msg.RepoPath == m.SelectedRepo {
		return m, fetchRepoInfo(msg.RepoPath, true)
	}

	return m, nil
}

// forceRepoCheck unconditionally fetches info for the selected repo.
// Used when cycling root tabs — always refreshes even if the repo path
// matches (handles single-repo roots revisited after navigating away).
func (m *Model) forceRepoCheck() tea.Cmd {
	m.SelectedRoot = m.Repos.Active

	repo, ok := m.Repos.SelectedRepo()
	if !ok {
		m.SelectedRepo = ""
		m.LastRepoInfo = nil

		return nil
	}

	m.SelectedRepo = repo.Path

	m.Status.SetMessage("Loading " + repo.Name + "...")

	// For non-git repos, update panels immediately (no async fetch needed).
	if !repo.IsGit {
		info := git.RepoInfo{
			Path:  repo.Path,
			IsGit: false,
			SCM:   git.SCMNone,
			Kind:  git.KindNotGit,
		}

		m.LastRepoInfo = &info
		m.Right.SetRepoInfo(&info, nil)

		return nil
	}

	return fetchRepoInfo(repo.Path, true)
}

// checkSelectedRepo detects when the selected repo or root changes and triggers a fetch.
func (m *Model) checkSelectedRepo() tea.Cmd {
	currentRoot := m.Repos.Active

	repo, ok := m.Repos.SelectedRepo()
	if !ok {
		// Root changed but no repo selected (empty tab) — clear panels.
		if currentRoot != m.SelectedRoot {
			m.SelectedRoot = currentRoot
			m.SelectedRepo = ""
			m.LastRepoInfo = nil
		}

		return nil
	}

	repoChanged := repo.Path != m.SelectedRepo
	rootChanged := currentRoot != m.SelectedRoot

	if !repoChanged && !rootChanged {
		return nil
	}

	m.SelectedRepo = repo.Path
	m.SelectedRoot = currentRoot

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

		// Use the fast path for navigation — skips expensive operations
		// (fsck, file stats, health, merge/rebase checks).
		// Full inspection runs on Ctrl+R (refreshRepo).
		ctx := context.Background()
		r := git.NewRunner(path)
		info := git.CollectRepoInfoFast(ctx, r, path)

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

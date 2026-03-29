package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// WizardStep tracks the current page of the configuration wizard.
type WizardStep int

const (
	wizardStepRoots        WizardStep = iota // browse/select existing roots
	wizardStepEditRoot                       // choose which field of a root to edit
	wizardStepEditPath                       // edit a root's path
	wizardStepEditName                       // edit a root's display name
	wizardStepEditInterval                   // edit a root's schedule interval
	wizardStepPath                           // enter a new root directory path
	wizardStepName                           // enter a name for the new root
	wizardStepInterval                       // enter a schedule interval for the new root
	wizardStepConfirm                        // review the new entry and confirm
	wizardStepDone                           // all done, about to close
)

// ConfigWizard is a multi-step overlay dialog for editing the configuration.
//
// When the config already has roots, it opens on the root list where the user
// can browse with arrow keys and press Enter to edit a root's settings.
// From the root list, [A] adds a new root (entering the add-new flow).
//
// When the config is empty, it goes straight to the add-new flow.
type ConfigWizard struct {
	Cfg *config.Config // working copy of the config

	PathInput       textinput.Model
	NameInput       textinput.Model // name for a new root
	IntervalInput   textinput.Model
	EditInput       textinput.Model // text input for editing an existing root's interval
	EditNameInput   textinput.Model // text input for editing an existing root's name
	EditPathInput   textinput.Model // text input for editing an existing root's path
	EditFieldCursor int             // cursor within the edit-root field list

	Step    WizardStep
	Visible bool
	Dirty   bool   // whether any modification has been made
	Err     string // validation error shown inline

	// root list cursor (for wizardStepRoots)
	RootCursor int

	// index of the root being edited (for wizardStepEditRoot)
	EditIndex int

	// pending values for the root being added
	PendingPath     string
	PendingName     string
	PendingInterval time.Duration

	// defaults loaded from the embedded config
	DefaultPath     string
	DefaultInterval string

	Width  int
	Height int
}

// editRootFields defines the fields available for editing on a root.
var editRootFields = []string{"Path", "Name", "Interval", "GitHub", "Security Alerts"} //nolint:gochecknoglobals // wizard field list

// New creates a new ConfigWizard for the given configuration.
func New(cfg *config.Config) ConfigWizard {
	defPath := resolveDefaultPath()
	defInterval := resolveDefaultInterval()

	pi := textinput.New()
	pi.Placeholder = defPath
	pi.Prompt = "  Path: "
	pi.CharLimit = 512
	pi.Width = 50
	pi.SetValue(defPath)

	ni := textinput.New()
	ni.Placeholder = "(defaults to directory name)"
	ni.Prompt = "  Name: "
	ni.CharLimit = 64
	ni.Width = 50

	ii := textinput.New()
	ii.Placeholder = defInterval
	ii.Prompt = "  Interval: "
	ii.CharLimit = 20
	ii.Width = 30

	ei := textinput.New()
	ei.Placeholder = defInterval
	ei.Prompt = "  Interval: "
	ei.CharLimit = 20
	ei.Width = 30

	eni := textinput.New()
	eni.Placeholder = "(directory name)"
	eni.Prompt = "  Name: "
	eni.CharLimit = 64
	eni.Width = 50

	epi := textinput.New()
	epi.Placeholder = "/path/to/root"
	epi.Prompt = "  Path: "
	epi.CharLimit = 512
	epi.Width = 50

	return ConfigWizard{
		Cfg:             cfg,
		PathInput:       pi,
		NameInput:       ni,
		IntervalInput:   ii,
		EditInput:       ei,
		EditNameInput:   eni,
		EditPathInput:   epi,
		Step:            wizardStepRoots,
		DefaultPath:     defPath,
		DefaultInterval: defInterval,
	}
}

// resolveDefaultPath returns the absolute path of the parent of the current
// working directory, or an empty string on error.
func resolveDefaultPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	parent := filepath.Join(cwd, "..")

	abs, err := filepath.Abs(parent)
	if err != nil {
		return ""
	}

	return abs
}

// resolveDefaultInterval returns the schedule interval from the embedded
// default config, formatted as a Go duration string (e.g. "5m0s").
func resolveDefaultInterval() string {
	defaults, err := config.LoadDefaults()
	if err != nil || defaults.Defaults.RootConfig == nil {
		return "5m"
	}

	d := defaults.Defaults.RootConfig.ScheduleInterval
	if d <= 0 {
		return "5m"
	}

	return d.String()
}

// Show opens the wizard overlay.
//
// If the config already has roots, it starts on the root list.
// Otherwise, it starts on the add-new-root flow.
func (w *ConfigWizard) Show() tea.Cmd {
	w.Visible = true
	w.Err = ""
	w.Dirty = false
	w.RootCursor = 0

	if len(w.Cfg.Roots) > 0 {
		w.Step = wizardStepRoots
		w.blurAll()

		return nil
	}

	// No roots — jump to add-new flow.
	w.Step = wizardStepPath
	w.PathInput.SetValue(w.DefaultPath)
	w.NameInput.SetValue("")
	w.IntervalInput.SetValue("")
	w.PathInput.Focus()

	return textinput.Blink
}

// Hide closes the wizard overlay.
func (w *ConfigWizard) Hide() {
	w.Visible = false
	w.blurAll()
}

func (w *ConfigWizard) blurAll() {
	w.PathInput.Blur()
	w.NameInput.Blur()
	w.IntervalInput.Blur()
	w.EditInput.Blur()
	w.EditNameInput.Blur()
	w.EditPathInput.Blur()
}

// SetSize adjusts the wizard popup dimensions.
func (w *ConfigWizard) SetSize(termWidth, termHeight int) {
	w.Width = termWidth * 3 / 4
	if w.Width < 50 {
		w.Width = min(50, termWidth)
	}

	w.Height = termHeight / 2
	if w.Height < 14 {
		w.Height = min(14, termHeight)
	}

	innerW := w.Width - 8 // borders + padding
	w.PathInput.Width = innerW
	w.NameInput.Width = innerW
	w.IntervalInput.Width = innerW
	w.EditInput.Width = innerW
	w.EditNameInput.Width = innerW
	w.EditPathInput.Width = innerW
}

// Update handles messages while the wizard is visible.
func (w *ConfigWizard) Update(msg tea.Msg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return w.handleKey(msg)
	}

	// Forward to the active text input.
	var cmd tea.Cmd

	switch w.Step {
	case wizardStepPath:
		w.PathInput, cmd = w.PathInput.Update(msg)
	case wizardStepName:
		w.NameInput, cmd = w.NameInput.Update(msg)
	case wizardStepInterval:
		w.IntervalInput, cmd = w.IntervalInput.Update(msg)
	case wizardStepEditPath:
		w.EditPathInput, cmd = w.EditPathInput.Update(msg)
	case wizardStepEditInterval:
		w.EditInput, cmd = w.EditInput.Update(msg)
	case wizardStepEditName:
		w.EditNameInput, cmd = w.EditNameInput.Update(msg)
	}

	return cmd, nil
}

func (w *ConfigWizard) handleKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	// Esc behavior depends on context.
	if key == "esc" {
		return w.handleEsc()
	}

	switch w.Step {
	case wizardStepRoots:
		return w.handleRootsKey(msg)
	case wizardStepEditRoot:
		return w.handleEditRootKey(msg)
	case wizardStepEditPath:
		return w.handleEditPathKey(msg)
	case wizardStepEditName:
		return w.handleEditNameKey(msg)
	case wizardStepEditInterval:
		return w.handleEditIntervalKey(msg)
	case wizardStepPath:
		return w.handlePathKey(msg)
	case wizardStepName:
		return w.handleNameKey(msg)
	case wizardStepInterval:
		return w.handleIntervalKey(msg)
	case wizardStepConfirm:
		return w.handleConfirmKey(msg)
	case wizardStepDone:
		w.Hide()

		return nil, &uxtypes.ConfigWizardMsg{Cfg: w.Cfg}
	}

	return nil, nil
}

func (w *ConfigWizard) handleEsc() (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	switch w.Step {
	case wizardStepEditRoot, wizardStepEditPath, wizardStepEditName, wizardStepEditInterval:
		// Go back to root list without saving the edit.
		w.Step = wizardStepRoots
		w.Err = ""
		w.blurAll()

		return nil, nil

	case wizardStepPath, wizardStepName, wizardStepInterval, wizardStepConfirm:
		if len(w.Cfg.Roots) > 0 {
			// Go back to root list.
			w.Step = wizardStepRoots
			w.Err = ""
			w.blurAll()

			return nil, nil
		}

		// No roots — Esc closes the wizard entirely.
		w.Hide()

		return nil, nil

	default:
		w.Hide()

		if w.Dirty {
			return nil, &uxtypes.ConfigWizardMsg{Cfg: w.Cfg}
		}

		return nil, nil
	}
}

// handleRootsKey handles keyboard input on the root list step.
func (w *ConfigWizard) handleRootsKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()
	n := len(w.Cfg.Roots)

	switch key {
	case "up", "k":
		if w.RootCursor > 0 {
			w.RootCursor--
		}

	case "down", "j":
		if w.RootCursor < n-1 {
			w.RootCursor++
		}

	case "enter":
		if n == 0 {
			return nil, nil
		}

		// Enter edit mode for the selected root (field selection).
		w.EditIndex = w.RootCursor
		w.EditFieldCursor = 0
		w.Step = wizardStepEditRoot
		w.Err = ""

		return nil, nil

	case "d", "D":
		// Delete the selected root.
		if n == 0 {
			return nil, nil
		}

		w.Cfg.DeleteRoot(w.RootCursor)
		w.Dirty = true

		// Clamp cursor.
		if w.RootCursor >= len(w.Cfg.Roots) && w.RootCursor > 0 {
			w.RootCursor--
		}

		return nil, nil

	case "a", "A":
		// Start add-new-root flow.
		w.Step = wizardStepPath
		w.Err = ""
		w.PathInput.SetValue(w.DefaultPath)
		w.NameInput.SetValue("")
		w.IntervalInput.SetValue("")
		w.PathInput.Focus()

		return textinput.Blink, nil

	case "s", "S":
		if !w.Dirty {
			return nil, nil
		}

		if err := w.Cfg.Save(); err != nil {
			w.Err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		w.Err = ""
		w.Step = wizardStepDone

		return nil, nil
	}

	return nil, nil
}

// handleEditRootKey handles the field-selection menu for editing a root.
func (w *ConfigWizard) handleEditRootKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "up", "k":
		if w.EditFieldCursor > 0 {
			w.EditFieldCursor--
		}
	case "down", "j":
		if w.EditFieldCursor < len(editRootFields)-1 {
			w.EditFieldCursor++
		}
	case "enter":
		root := w.Cfg.Roots[w.EditIndex]

		switch editRootFields[w.EditFieldCursor] {
		case "Path":
			w.Step = wizardStepEditPath
			w.Err = ""
			w.EditPathInput.SetValue(root.Path)
			w.EditPathInput.Focus()

			return textinput.Blink, nil
		case "Name":
			w.Step = wizardStepEditName
			w.Err = ""
			w.EditNameInput.SetValue(w.Cfg.RootDisplayName(w.EditIndex))
			w.EditNameInput.Focus()

			return textinput.Blink, nil
		case "Interval":
			w.Step = wizardStepEditInterval
			w.Err = ""
			w.EditInput.SetValue(root.RootConfig.ScheduleInterval.String())
			w.EditInput.Focus()

			return textinput.Blink, nil
		case "GitHub":
			w.toggleRootGitHub()

			return nil, nil
		case "Security Alerts":
			w.toggleRootSecurityAlerts()

			return nil, nil
		}

	case "s", "S":
		if !w.Dirty {
			return nil, nil
		}

		if err := w.Cfg.Save(); err != nil {
			w.Err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		w.Err = ""
		w.Step = wizardStepDone

		return nil, nil
	}

	return nil, nil
}

// handleEditPathKey handles keyboard input when editing a root's path.
func (w *ConfigWizard) handleEditPathKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		path := strings.TrimSpace(w.EditPathInput.Value())
		if path == "" {
			w.Err = "Path cannot be empty."

			return nil, nil
		}

		// Expand ~ to home directory.
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				w.Err = fmt.Sprintf("Cannot resolve home directory: %v", err)

				return nil, nil
			}

			path = filepath.Join(home, path[2:])
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			w.Err = fmt.Sprintf("Cannot resolve absolute Path: %v", err)

			return nil, nil
		}

		path = absPath

		info, err := os.Stat(path)
		if err != nil {
			w.Err = fmt.Sprintf("Cannot access Path: %v", err)

			return nil, nil
		}

		if !info.IsDir() {
			w.Err = "Path is not a directory."

			return nil, nil
		}

		w.Cfg.UpdateRootPath(w.EditIndex, path)
		w.Dirty = true
		w.Err = ""
		w.Step = wizardStepEditRoot
		w.EditPathInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.EditPathInput, cmd = w.EditPathInput.Update(msg)

		return cmd, nil
	}
}

// handleEditNameKey handles keyboard input when editing a root's display name.
func (w *ConfigWizard) handleEditNameKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		name := strings.TrimSpace(w.EditNameInput.Value())
		// Empty name is valid — it will fall back to basename(path).
		w.Cfg.UpdateRootName(w.EditIndex, name)
		w.Dirty = true
		w.Err = ""
		w.Step = wizardStepEditRoot
		w.EditNameInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.EditNameInput, cmd = w.EditNameInput.Update(msg)

		return cmd, nil
	}
}

// handleEditIntervalKey handles keyboard input when editing a root's schedule interval.
func (w *ConfigWizard) handleEditIntervalKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		raw := strings.TrimSpace(w.EditInput.Value())
		if raw == "" {
			w.Err = "Interval cannot be empty."

			return nil, nil
		}

		d, err := time.ParseDuration(raw)
		if err != nil {
			w.Err = fmt.Sprintf("Invalid duration %q — use Go syntax (e.g. 24h, 30m).", raw)

			return nil, nil
		}

		if d <= 0 {
			w.Err = "Interval must be positive."

			return nil, nil
		}

		// Apply the change.
		w.Cfg.UpdateRootInterval(w.EditIndex, d)
		w.Dirty = true
		w.Err = ""
		w.Step = wizardStepEditRoot
		w.EditInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.EditInput, cmd = w.EditInput.Update(msg)

		return cmd, nil
	}
}

// toggleRootGitHub toggles the per-root GitHub.Enabled override.
func (w *ConfigWizard) toggleRootGitHub() {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	root := &w.Cfg.Roots[w.EditIndex]
	if root.RootConfig.GitHub == nil {
		// First toggle: inherit global, then flip.
		enabled := !w.Cfg.GitHub.Enabled
		root.RootConfig.GitHub = &config.GitHubConfig{Enabled: enabled}
	} else {
		root.RootConfig.GitHub.Enabled = !root.RootConfig.GitHub.Enabled
	}

	w.Dirty = true
}

// toggleRootSecurityAlerts toggles the per-root GitHub.SecurityAlerts override.
func (w *ConfigWizard) toggleRootSecurityAlerts() {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	root := &w.Cfg.Roots[w.EditIndex]
	if root.RootConfig.GitHub == nil {
		root.RootConfig.GitHub = &config.GitHubConfig{Enabled: w.Cfg.GitHub.Enabled}
	}

	if root.RootConfig.GitHub.SecurityAlerts == nil {
		// First toggle: inherit global, then flip.
		v := !w.Cfg.GitHubSecurityAlerts(w.EditIndex)
		root.RootConfig.GitHub.SecurityAlerts = &v
	} else {
		v := !*root.RootConfig.GitHub.SecurityAlerts
		root.RootConfig.GitHub.SecurityAlerts = &v
	}

	w.Dirty = true
}

func (w *ConfigWizard) handlePathKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		path := strings.TrimSpace(w.PathInput.Value())
		if path == "" {
			w.Err = "Path cannot be empty."

			return nil, nil
		}

		// Expand ~ to home directory.
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				w.Err = fmt.Sprintf("Cannot resolve home directory: %v", err)

				return nil, nil
			}

			path = filepath.Join(home, path[2:])
		}

		// Convert to absolute path.
		absPath, err := filepath.Abs(path)
		if err != nil {
			w.Err = fmt.Sprintf("Cannot resolve absolute Path: %v", err)

			return nil, nil
		}

		path = absPath

		// Validate the path exists and is a directory.
		info, err := os.Stat(path)
		if err != nil {
			w.Err = fmt.Sprintf("Cannot access Path: %v", err)

			return nil, nil
		}

		if !info.IsDir() {
			w.Err = "Path is not a directory."

			return nil, nil
		}

		w.PendingPath = path
		w.Err = ""
		w.Step = wizardStepName
		w.PathInput.Blur()

		// Pre-fill name with the basename of the path.
		w.NameInput.SetValue(filepath.Base(path))
		w.NameInput.Focus()

		return textinput.Blink, nil

	default:
		var cmd tea.Cmd
		w.PathInput, cmd = w.PathInput.Update(msg)

		return cmd, nil
	}
}

// handleNameKey handles keyboard input on the name step (add-new flow).
func (w *ConfigWizard) handleNameKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		name := strings.TrimSpace(w.NameInput.Value())
		// Empty name is valid — AddRoot defaults to basename(path).
		w.PendingName = name
		w.Err = ""
		w.Step = wizardStepInterval
		w.NameInput.Blur()
		w.IntervalInput.Focus()

		return textinput.Blink, nil

	default:
		var cmd tea.Cmd
		w.NameInput, cmd = w.NameInput.Update(msg)

		return cmd, nil
	}
}

func (w *ConfigWizard) handleIntervalKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		raw := strings.TrimSpace(w.IntervalInput.Value())
		if raw == "" {
			raw = w.DefaultInterval
		}

		d, err := time.ParseDuration(raw)
		if err != nil {
			w.Err = fmt.Sprintf("Invalid duration %q — use Go duration syntax (e.g. 24h, 30m, 168h).", raw)

			return nil, nil
		}

		if d <= 0 {
			w.Err = "Interval must be positive."

			return nil, nil
		}

		w.PendingInterval = d
		w.Err = ""
		w.Step = wizardStepConfirm
		w.IntervalInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.IntervalInput, cmd = w.IntervalInput.Update(msg)

		return cmd, nil
	}
}

func (w *ConfigWizard) handleConfirmKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case "s", "S", "enter":
		// Commit the root to the working config.
		w.Cfg.AddRoot(w.PendingName, w.PendingPath, w.PendingInterval)
		w.Dirty = true

		// Save to disk.
		if err := w.Cfg.Save(); err != nil {
			w.Err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		w.Err = ""
		w.Step = wizardStepDone

		return nil, nil

	case "a", "A":
		// Add another root — save first, then restart add flow.
		w.Cfg.AddRoot(w.PendingName, w.PendingPath, w.PendingInterval)
		w.Dirty = true

		if err := w.Cfg.Save(); err != nil {
			w.Err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		// Reset for another entry.
		w.Step = wizardStepPath
		w.Err = ""
		w.PathInput.SetValue(w.DefaultPath)
		w.NameInput.SetValue("")
		w.IntervalInput.SetValue("")
		w.PathInput.Focus()

		return textinput.Blink, nil

	case "n", "N":
		// Cancel this entry — go back to root list if roots exist.
		if len(w.Cfg.Roots) > 0 {
			w.Step = wizardStepRoots
			w.Err = ""
			w.blurAll()

			return nil, nil
		}

		w.Hide()

		return nil, nil
	}

	return nil, nil
}

// View renders the wizard overlay.
func (w *ConfigWizard) View(termWidth, termHeight int) string {
	if !w.Visible {
		return ""
	}

	var content strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).
		Render("Configuration Wizard")
	content.WriteString(title + "\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("Configure root directories for git-janitor to scan.") + "\n\n")

	switch w.Step {
	case wizardStepRoots:
		w.viewRoots(&content)
	case wizardStepEditRoot:
		w.viewEditRoot(&content)
	case wizardStepEditPath:
		w.viewEditPath(&content)
	case wizardStepEditName:
		w.viewEditName(&content)
	case wizardStepEditInterval:
		w.viewEditInterval(&content)
	case wizardStepPath:
		w.viewPath(&content)
	case wizardStepName:
		w.viewName(&content)
	case wizardStepInterval:
		w.viewInterval(&content)
	case wizardStepConfirm:
		w.viewConfirm(&content)
	case wizardStepDone:
		w.viewDone(&content)
	}

	// Error line.
	if w.Err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		content.WriteString("\n" + errStyle.Render("  ⚠ "+w.Err) + "\n")
	}

	// Footer.
	w.viewFooter(&content)

	// Wrap in a bordered box.
	border := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("170")).
		Width(w.Width-2).
		Padding(1, 2)

	popup := border.Render(content.String())

	return lipgloss.Place(
		termWidth, termHeight,
		lipgloss.Center, lipgloss.Center,
		popup,
	)
}

func (w *ConfigWizard) viewRoots(content *strings.Builder) {
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).
		Render(fmt.Sprintf("Configured roots (%d):", len(w.Cfg.Roots)))
	content.WriteString(header + "\n")

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)

	for i := range w.Cfg.Roots {
		cursor := "  "
		style := dim
		if i == w.RootCursor {
			cursor = "▸ "
			style = selected
		}

		name := w.Cfg.RootDisplayName(i)
		r := w.Cfg.Roots[i]
		line := fmt.Sprintf("%s%s  %s  (every %s)", cursor, name, r.Path, r.RootConfig.ScheduleInterval)
		content.WriteString(style.Render(line) + "\n")
	}

	content.WriteString("\n")

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(hint.Render("  ↑/↓ select   Enter edit   [A] add new   [D] delete") + "\n")
}

func (w *ConfigWizard) viewEditRoot(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	root := w.Cfg.Roots[w.EditIndex]
	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	content.WriteString(heading.Render(fmt.Sprintf("Editing root: %s", w.Cfg.RootDisplayName(w.EditIndex))) + "\n\n")

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)

	ghEnabled := w.Cfg.GitHubEnabled(w.EditIndex)
	ghSecurity := w.Cfg.GitHubSecurityAlerts(w.EditIndex)

	ghLabel := "yes"
	if !ghEnabled {
		ghLabel = "no"
	}

	secLabel := "yes"
	if !ghSecurity {
		secLabel = "no"
	}

	// Show "(inherited)" when no per-root override is set.
	if root.RootConfig.GitHub == nil {
		ghLabel += " (inherited)"
		secLabel += " (inherited)"
	} else if root.RootConfig.GitHub.SecurityAlerts == nil {
		secLabel += " (inherited)"
	}

	fields := []struct {
		label string
		value string
	}{
		{"Path", root.Path},
		{"Name", w.Cfg.RootDisplayName(w.EditIndex)},
		{"Interval", root.RootConfig.ScheduleInterval.String()},
		{"GitHub", ghLabel},
		{"Security Alerts", secLabel},
	}

	for i, f := range fields {
		cursor := "  "
		style := dim
		if i == w.EditFieldCursor {
			cursor = "▸ "
			style = selected
		}

		line := fmt.Sprintf("%s%-10s %s", cursor, f.label+":", f.value)
		content.WriteString(style.Render(line) + "\n")
	}

	content.WriteString("\n")

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(hint.Render("  ↑/↓ select   Enter edit/toggle   Esc back") + "\n")
}

func (w *ConfigWizard) viewEditPath(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render(fmt.Sprintf("Editing path for: %s", w.Cfg.RootDisplayName(w.EditIndex))) + "\n\n")
	content.WriteString("  Absolute path to a directory containing git Repos:\n\n")
	content.WriteString(w.EditPathInput.View() + "\n")
}

func (w *ConfigWizard) viewEditName(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render(fmt.Sprintf("Editing name for: %s", w.Cfg.Roots[w.EditIndex].Path)) + "\n\n")
	content.WriteString("  Display name (leave empty to use directory name):\n\n")
	content.WriteString(w.EditNameInput.View() + "\n")
}

func (w *ConfigWizard) viewEditInterval(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render(fmt.Sprintf("Editing interval for: %s", w.Cfg.RootDisplayName(w.EditIndex))) + "\n\n")
	content.WriteString("  Schedule interval (Go duration syntax, e.g. 24h, 30m):\n\n")
	content.WriteString(w.EditInput.View() + "\n")
}

func (w *ConfigWizard) viewPath(content *strings.Builder) {
	// Show existing roots if any (context for "add another").
	if len(w.Cfg.Roots) > 0 {
		existing := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).
			Render(fmt.Sprintf("Configured roots (%d):", len(w.Cfg.Roots)))
		content.WriteString(existing + "\n")

		for i, r := range w.Cfg.Roots {
			name := w.Cfg.RootDisplayName(i)
			fmt.Fprintf(content, "  • %s  %s  (every %s)\n", name, r.Path, r.RootConfig.ScheduleInterval)
		}

		content.WriteString("\n")
	}

	content.WriteString("Add root — Path\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  Enter an absolute path to a directory containing git repos.") + "\n\n")
	content.WriteString(w.PathInput.View() + "\n")
}

func (w *ConfigWizard) viewName(content *strings.Builder) {
	fmt.Fprintf(content, "Add root — Name for %s\n", w.PendingPath)
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  Display name for this root (leave empty to use directory name).") + "\n\n")
	content.WriteString(w.NameInput.View() + "\n")
}

func (w *ConfigWizard) viewInterval(content *strings.Builder) {
	fmt.Fprintf(content, "Add root — Interval for %s\n", w.PendingPath)
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  How often should this root be checked? (default: "+w.DefaultInterval+")") + "\n\n")
	content.WriteString(w.IntervalInput.View() + "\n")
}

func (w *ConfigWizard) viewConfirm(content *strings.Builder) {
	content.WriteString("Add root — Review\n\n")

	displayName := w.PendingName
	if displayName == "" {
		displayName = filepath.Base(w.PendingPath) + " (default)"
	}

	fmt.Fprintf(content, "  Name:     %s\n", displayName)
	fmt.Fprintf(content, "  Path:     %s\n", w.PendingPath)
	fmt.Fprintf(content, "  Interval: %s\n\n", w.PendingInterval)

	actions := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	content.WriteString(actions.Render("  [S/Enter] Save & close") + "  ")
	content.WriteString(actions.Render("[A] Save & add another") + "  ")
	content.WriteString(actions.Render("[N] Cancel") + "\n")
}

func (w *ConfigWizard) viewDone(content *strings.Builder) {
	checkmark := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	fmt.Fprintf(content, "\n  %s Configuration saved!\n", checkmark)

	path, _ := config.DefaultConfigPath()
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render(fmt.Sprintf("    Written to %s", path)) + "\n\n")
	content.WriteString("  Press any key to close.\n")
}

func (w *ConfigWizard) viewFooter(content *strings.Builder) {
	if w.Step == wizardStepDone {
		return
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var parts []string
	if w.Dirty {
		save := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).
			Render("[S] Save")
		parts = append(parts, save)
	}

	parts = append(parts, hint.Render("Esc to cancel"))

	content.WriteString("\n  " + strings.Join(parts, "  ") + "\n")
}

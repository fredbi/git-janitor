package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/fs"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Step tracks the current page of the configuration wizard.
type Step int

const (
	stepRoots        Step = iota // browse/select existing roots
	stepEditRoot                 // choose which field of a root to edit
	stepEditPath                 // edit a root's path
	stepEditName                 // edit a root's display name
	stepEditInterval             // edit a root's schedule interval
	stepEditMaxDepth             // edit a root's discovery max depth
	stepPath                     // enter a new root directory path
	stepName                     // enter a name for the new root
	stepInterval                 // enter a schedule interval for the new root
	stepConfirm                  // review the new entry and confirm
	stepDone                     // all done, about to close
)

const keyEnter = "enter"

const (
	paneWidth       = 50
	paneHeight      = 14
	padding         = 8
	depthInputWidth = 10
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

	PathInput         gadgets.PathAutocomplete // path input for a new root with directory autocompletion
	NameInput         textinput.Model          // name for a new root
	IntervalInput     textinput.Model
	EditInput         textinput.Model          // text input for editing an existing root's interval
	EditNameInput     textinput.Model          // text input for editing an existing root's name
	EditMaxDepthInput textinput.Model          // text input for editing an existing root's MaxDepth
	EditPathInput     gadgets.PathAutocomplete // path input for editing an existing root with directory autocompletion
	EditFieldCursor   int                      // cursor within the edit-root field list

	Step    Step
	Visible bool
	Dirty   bool   // whether any modification has been made
	Err     string // validation error shown inline

	// root list cursor (for stepRoots)
	RootCursor int

	// index of the root being edited (for stepEditRoot)
	EditIndex int

	// pending values for the root being added
	PendingPath     string
	PendingName     string
	PendingInterval time.Duration

	// defaults loaded from the embedded config
	DefaultPath     string
	DefaultInterval string

	// ThemeNames is the ordered list of available theme names (for cycling).
	ThemeNames []string

	Width  int
	Height int
}

// editRootFields defines the fields available for editing on a root.
var editRootFields = []string{"Path", "Name", "Interval", "Max Depth", "GitHub", "Security Alerts"} //nolint:gochecknoglobals // wizard field list

// New creates a new ConfigWizard for the given configuration.
// themeNames provides the ordered list of available color themes for the [T] toggle.
func New(cfg *config.Config, themeNames []string) ConfigWizard {
	defPath := resolveDefaultPath()
	defInterval := resolveDefaultInterval()

	pi := textinput.New()
	pi.Placeholder = defPath
	pi.Prompt = "  Path: "
	pi.CharLimit = 512
	pi.Width = paneWidth
	pi.SetValue(defPath)

	ni := textinput.New()
	ni.Placeholder = "(defaults to directory name)"
	ni.Prompt = "  Name: "
	ni.CharLimit = 64
	ni.Width = paneWidth

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
	eni.Width = paneWidth

	epi := textinput.New()
	epi.Placeholder = "/path/to/root"
	epi.Prompt = "  Path: "
	epi.CharLimit = 512
	epi.Width = paneWidth

	emdi := textinput.New()
	emdi.Placeholder = strconv.Itoa(config.DefaultMaxDepth)
	emdi.Prompt = "  Max Depth: "
	emdi.CharLimit = 4
	emdi.Width = depthInputWidth

	return ConfigWizard{
		Cfg:               cfg,
		PathInput:         gadgets.NewPathAutocomplete(pi),
		NameInput:         ni,
		IntervalInput:     ii,
		EditInput:         ei,
		EditNameInput:     eni,
		EditMaxDepthInput: emdi,
		EditPathInput:     gadgets.NewPathAutocomplete(epi),
		Step:              stepRoots,
		DefaultPath:       defPath,
		DefaultInterval:   defInterval,
		ThemeNames:        themeNames,
	}
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
		w.Step = stepRoots
		w.blurAll()

		return nil
	}

	// No roots — jump to add-new flow.
	w.Step = stepPath
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

// SetSize adjusts the wizard popup dimensions.
func (w *ConfigWizard) SetSize(termWidth, termHeight int) {
	w.Width = termWidth * 3 / 4
	if w.Width < paneWidth {
		w.Width = min(paneWidth, termWidth)
	}

	w.Height = termHeight / 2
	if w.Height < paneHeight {
		w.Height = min(paneHeight, termHeight)
	}

	innerW := w.Width - padding // borders + padding
	w.PathInput.SetWidth(innerW)
	w.NameInput.Width = innerW
	w.IntervalInput.Width = innerW
	w.EditInput.Width = innerW
	w.EditNameInput.Width = innerW
	w.EditMaxDepthInput.Width = min(depthInputWidth, innerW)
	w.EditPathInput.SetWidth(innerW)
}

// Update handles messages while the wizard is visible.
func (w *ConfigWizard) Update(msg tea.Msg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		return w.handleKey(msg)
	}

	// Forward to the active text input.
	var cmd tea.Cmd

	switch w.Step {
	case stepPath:
		cmd, _ = w.PathInput.Update(msg)
	case stepName:
		w.NameInput, cmd = w.NameInput.Update(msg)
	case stepInterval:
		w.IntervalInput, cmd = w.IntervalInput.Update(msg)
	case stepEditPath:
		cmd, _ = w.EditPathInput.Update(msg)
	case stepEditInterval:
		w.EditInput, cmd = w.EditInput.Update(msg)
	case stepEditName:
		w.EditNameInput, cmd = w.EditNameInput.Update(msg)
	case stepEditMaxDepth:
		w.EditMaxDepthInput, cmd = w.EditMaxDepthInput.Update(msg)
	default:
	}

	return cmd, nil
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
	case stepRoots:
		w.viewRoots(&content)
	case stepEditRoot:
		w.viewEditRoot(&content)
	case stepEditPath:
		w.viewEditPath(&content)
	case stepEditName:
		w.viewEditName(&content)
	case stepEditInterval:
		w.viewEditInterval(&content)
	case stepEditMaxDepth:
		w.viewEditMaxDepth(&content)
	case stepPath:
		w.viewPath(&content)
	case stepName:
		w.viewName(&content)
	case stepInterval:
		w.viewInterval(&content)
	case stepConfirm:
		w.viewConfirm(&content)
	case stepDone:
		w.viewDone(&content)
	default:
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

func (w *ConfigWizard) blurAll() {
	w.PathInput.Blur()
	w.NameInput.Blur()
	w.IntervalInput.Blur()
	w.EditInput.Blur()
	w.EditNameInput.Blur()
	w.EditMaxDepthInput.Blur()
	w.EditPathInput.Blur()
}

func (w *ConfigWizard) handleKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	// Esc behavior depends on context.
	if key == "esc" {
		return w.handleEsc()
	}

	switch w.Step {
	case stepRoots:
		return w.handleRootsKey(msg)
	case stepEditRoot:
		return w.handleEditRootKey(msg)
	case stepEditPath:
		return w.handleEditPathKey(msg)
	case stepEditName:
		return w.handleEditNameKey(msg)
	case stepEditInterval:
		return w.handleEditIntervalKey(msg)
	case stepEditMaxDepth:
		return w.handleEditMaxDepthKey(msg)
	case stepPath:
		return w.handlePathKey(msg)
	case stepName:
		return w.handleNameKey(msg)
	case stepInterval:
		return w.handleIntervalKey(msg)
	case stepConfirm:
		return w.handleConfirmKey(msg)
	case stepDone:
		w.Hide()

		return nil, &uxtypes.ConfigWizardMsg{Cfg: w.Cfg}
	default:
		return nil, nil
	}
}

func (w *ConfigWizard) handleEsc() (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	switch w.Step {
	case stepEditRoot, stepEditPath, stepEditName, stepEditInterval, stepEditMaxDepth:
		// Go back to root list without saving the edit.
		w.Step = stepRoots
		w.Err = ""
		w.blurAll()

		return nil, nil

	case stepPath, stepName, stepInterval, stepConfirm:
		if len(w.Cfg.Roots) > 0 {
			// Go back to root list.
			w.Step = stepRoots
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

	case keyEnter:
		if n == 0 {
			return nil, nil
		}

		// Enter edit mode for the selected root (field selection).
		w.EditIndex = w.RootCursor
		w.EditFieldCursor = 0
		w.Step = stepEditRoot
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
		w.Step = stepPath
		w.Err = ""
		w.PathInput.SetValue(w.DefaultPath)
		w.NameInput.SetValue("")
		w.IntervalInput.SetValue("")
		w.PathInput.Focus()

		return textinput.Blink, nil

	case "t", "T":
		w.cycleTheme()

		return nil, nil

	case "s", "S":
		if !w.Dirty {
			return nil, nil
		}

		if err := w.Cfg.Save(); err != nil {
			w.Err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		w.Err = ""
		w.Step = stepDone

		return nil, nil
	}

	return nil, nil
}

// cycleTheme advances the config theme to the next available theme name.
func (w *ConfigWizard) cycleTheme() {
	if len(w.ThemeNames) == 0 {
		return
	}

	current := w.Cfg.Theme
	if current == "" {
		current = "default"
	}

	// Find current index and advance.
	next := w.ThemeNames[0]
	for i, name := range w.ThemeNames {
		if name == current && i+1 < len(w.ThemeNames) {
			next = w.ThemeNames[i+1]

			break
		}
	}

	w.Cfg.Theme = next
	w.Dirty = true
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
	case keyEnter:
		root := w.Cfg.Roots[w.EditIndex]

		switch editRootFields[w.EditFieldCursor] {
		case "Path":
			w.Step = stepEditPath
			w.Err = ""
			w.EditPathInput.SetValue(root.Path)
			w.EditPathInput.Focus()

			return textinput.Blink, nil
		case "Name":
			w.Step = stepEditName
			w.Err = ""
			w.EditNameInput.SetValue(w.Cfg.RootDisplayName(w.EditIndex))
			w.EditNameInput.Focus()

			return textinput.Blink, nil
		case "Interval":
			w.Step = stepEditInterval
			w.Err = ""
			w.EditInput.SetValue(root.RootConfig.ScheduleInterval.String())
			w.EditInput.Focus()

			return textinput.Blink, nil
		case "Max Depth":
			w.Step = stepEditMaxDepth
			w.Err = ""
			w.EditMaxDepthInput.SetValue(strconv.Itoa(w.Cfg.RootMaxDepth(w.EditIndex)))
			w.EditMaxDepthInput.Focus()

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
		w.Step = stepDone

		return nil, nil
	}

	return nil, nil
}

// handleEditPathKey handles keyboard input when editing a root's path.
func (w *ConfigWizard) handleEditPathKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	// Let the autocomplete gadget have first crack at the key.
	// It consumes navigation and selection keys; non-consumed keys
	// (Enter without an active selection, Esc) fall through.
	cmd, consumed := w.EditPathInput.Update(msg)
	if consumed {
		return cmd, nil
	}

	if msg.String() != keyEnter {
		return cmd, nil
	}

	path, errMsg := resolvePath(w.EditPathInput.Value())
	if errMsg != "" {
		w.Err = errMsg

		return nil, nil
	}

	// The path change can shift the root's position when no display name
	// is set; follow it via the returned new index.
	w.EditIndex = w.Cfg.UpdateRootPath(w.EditIndex, path)
	w.RootCursor = w.EditIndex
	w.Dirty = true
	w.Err = ""
	w.Step = stepEditRoot
	w.EditPathInput.Blur()

	return nil, nil
}

// handleEditNameKey handles keyboard input when editing a root's display name.
func (w *ConfigWizard) handleEditNameKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case keyEnter:
		name := strings.TrimSpace(w.EditNameInput.Value())
		// Empty name is valid — it will fall back to basename(path).
		// The rename can shift the root's position; follow it.
		w.EditIndex = w.Cfg.UpdateRootName(w.EditIndex, name)
		w.RootCursor = w.EditIndex
		w.Dirty = true
		w.Err = ""
		w.Step = stepEditRoot
		w.EditNameInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.EditNameInput, cmd = w.EditNameInput.Update(msg)

		return cmd, nil
	}
}

// handleEditMaxDepthKey handles keyboard input when editing a root's discovery depth.
func (w *ConfigWizard) handleEditMaxDepthKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case keyEnter:
		raw := strings.TrimSpace(w.EditMaxDepthInput.Value())
		if raw == "" {
			w.Err = "Max Depth cannot be empty."

			return nil, nil
		}

		d, err := strconv.Atoi(raw)
		if err != nil {
			w.Err = fmt.Sprintf("Invalid integer %q.", raw)

			return nil, nil
		}

		if d < -1 {
			w.Err = "Max Depth must be -1 (unlimited), 0 (default), or a positive integer."

			return nil, nil
		}

		w.Cfg.UpdateRootMaxDepth(w.EditIndex, d)
		w.Dirty = true
		w.Err = ""
		w.Step = stepEditRoot
		w.EditMaxDepthInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.EditMaxDepthInput, cmd = w.EditMaxDepthInput.Update(msg)

		return cmd, nil
	}
}

// handleEditIntervalKey handles keyboard input when editing a root's schedule interval.
func (w *ConfigWizard) handleEditIntervalKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case keyEnter:
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
		w.Step = stepEditRoot
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
	// Let the autocomplete gadget have first crack at the key.
	cmd, consumed := w.PathInput.Update(msg)
	if consumed {
		return cmd, nil
	}

	if msg.String() != keyEnter {
		return cmd, nil
	}

	path, errMsg := resolvePath(w.PathInput.Value())
	if errMsg != "" {
		w.Err = errMsg

		return nil, nil
	}

	w.PendingPath = path
	w.Err = ""
	w.Step = stepName
	w.PathInput.Blur()

	// Pre-fill name with the basename of the path.
	w.NameInput.SetValue(filepath.Base(path))
	w.NameInput.Focus()

	return textinput.Blink, nil
}

// resolvePath cleans, expands and validates a directory path entered by
// the user. It returns the absolute resolved path and an empty errMsg on
// success, or an empty path and a user-facing errMsg on failure.
func resolvePath(raw string) (string, string) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", "Path cannot be empty."
	}

	expanded, err := fs.ExpandHome(path)
	if err != nil {
		return "", fmt.Sprintf("Cannot resolve home directory: %v", err)
	}

	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Sprintf("Cannot resolve absolute Path: %v", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Sprintf("Cannot access Path: %v", err)
	}

	if !info.IsDir() {
		return "", "Path is not a directory."
	}

	return absPath, ""
}

// handleNameKey handles keyboard input on the name step (add-new flow).
func (w *ConfigWizard) handleNameKey(msg tea.KeyMsg) (tea.Cmd, *uxtypes.ConfigWizardMsg) {
	key := msg.String()

	switch key {
	case keyEnter:
		name := strings.TrimSpace(w.NameInput.Value())
		// Empty name is valid — AddRoot defaults to basename(path).
		w.PendingName = name
		w.Err = ""
		w.Step = stepInterval
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
	case keyEnter:
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
		w.Step = stepConfirm
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
	case "s", "S", keyEnter:
		// Commit the root to the working config.
		w.Cfg.AddRoot(w.PendingName, w.PendingPath, w.PendingInterval)
		w.Dirty = true

		// Save to disk.
		if err := w.Cfg.Save(); err != nil {
			w.Err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		w.Err = ""
		w.Step = stepDone

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
		w.Step = stepPath
		w.Err = ""
		w.PathInput.SetValue(w.DefaultPath)
		w.NameInput.SetValue("")
		w.IntervalInput.SetValue("")
		w.PathInput.Focus()

		return textinput.Blink, nil

	case "n", "N":
		// Cancel this entry — go back to root list if roots exist.
		if len(w.Cfg.Roots) > 0 {
			w.Step = stepRoots
			w.Err = ""
			w.blurAll()

			return nil, nil
		}

		w.Hide()

		return nil, nil
	}

	return nil, nil
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

	// Show current theme.
	themeName := w.Cfg.Theme
	if themeName == "" {
		themeName = "default"
	}

	themeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).
		Render("Theme: " + themeName)
	content.WriteString("  " + themeLabel + "\n\n")

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(hint.Render("  ↑/↓ select   Enter edit   [A] add new   [D] delete   [T] theme") + "\n")
}

func (w *ConfigWizard) viewEditRoot(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	root := w.Cfg.Roots[w.EditIndex]
	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	content.WriteString(heading.Render("Editing root: "+w.Cfg.RootDisplayName(w.EditIndex)) + "\n\n")

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
	const inherited = " (inherited)"
	if root.RootConfig.GitHub == nil {
		ghLabel += inherited
		secLabel += inherited
	} else if root.RootConfig.GitHub.SecurityAlerts == nil {
		secLabel += inherited
	}

	depthLabel := strconv.Itoa(w.Cfg.RootMaxDepth(w.EditIndex))
	if root.RootConfig.MaxDepth == 0 {
		depthLabel += " (default)"
	} else if root.RootConfig.MaxDepth < 0 {
		depthLabel = "unlimited"
	}

	fields := []struct {
		label string
		value string
	}{
		{"Path", root.Path},
		{"Name", w.Cfg.RootDisplayName(w.EditIndex)},
		{"Interval", root.RootConfig.ScheduleInterval.String()},
		{"Max Depth", depthLabel},
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

		line := fmt.Sprintf("%s%-16s %s", cursor, f.label+":", f.value)
		content.WriteString(style.Render(line) + "\n")
	}

	content.WriteString("\n")

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(hint.Render("  ↑/↓ select   Enter edit/toggle   Esc back") + "\n")
}

func (w *ConfigWizard) viewEditMaxDepth(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render("Editing max depth for: "+w.Cfg.RootDisplayName(w.EditIndex)) + "\n\n")
	content.WriteString("  How many directory levels should the scanner descend?\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("    1 = flat (GitHub-style), 4 = default, -1 = unlimited") + "\n\n")
	content.WriteString(w.EditMaxDepthInput.View() + "\n")
}

func (w *ConfigWizard) viewEditPath(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render("Editing path for: "+w.Cfg.RootDisplayName(w.EditIndex)) + "\n\n")
	content.WriteString("  Absolute path to a directory containing git Repos:\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  ↑/↓ to pick from suggestions, Enter or Tab to accept.") + "\n\n")
	content.WriteString(w.EditPathInput.View() + "\n")
}

func (w *ConfigWizard) viewEditName(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render("Editing name for: "+w.Cfg.Roots[w.EditIndex].Path) + "\n\n")
	content.WriteString("  Display name (leave empty to use directory name):\n\n")
	content.WriteString(w.EditNameInput.View() + "\n")
}

func (w *ConfigWizard) viewEditInterval(content *strings.Builder) {
	if w.EditIndex < 0 || w.EditIndex >= len(w.Cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render("Editing interval for: "+w.Cfg.RootDisplayName(w.EditIndex)) + "\n\n")
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
		Render("  Enter an absolute path to a directory containing git repos.") + "\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  ↑/↓ to pick from suggestions, Enter or Tab to accept.") + "\n\n")
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
		Render("    Written to "+path) + "\n\n")
	content.WriteString("  Press any key to close.\n")
}

func (w *ConfigWizard) viewFooter(content *strings.Builder) {
	if w.Step == stepDone {
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

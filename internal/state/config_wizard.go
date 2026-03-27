package state

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
)

// wizardStep tracks the current page of the configuration wizard.
type wizardStep int

const (
	wizardStepRoots        wizardStep = iota // browse/select existing roots
	wizardStepEditRoot                       // choose which field of a root to edit
	wizardStepEditName                       // edit a root's display name
	wizardStepEditInterval                   // edit a root's schedule interval
	wizardStepPath                           // enter a new root directory path
	wizardStepName                           // enter a name for the new root
	wizardStepInterval                       // enter a schedule interval for the new root
	wizardStepConfirm                        // review the new entry and confirm
	wizardStepDone                           // all done, about to close
)

// configWizardMsg is sent by the wizard when it finishes successfully.
type configWizardMsg struct {
	cfg *config.Config
}

// configWizard is a multi-step overlay dialog for editing the configuration.
//
// When the config already has roots, it opens on the root list where the user
// can browse with arrow keys and press Enter to edit a root's settings.
// From the root list, [A] adds a new root (entering the add-new flow).
//
// When the config is empty, it goes straight to the add-new flow.
type configWizard struct {
	cfg *config.Config // working copy of the config

	pathInput       textinput.Model
	nameInput       textinput.Model // name for a new root
	intervalInput   textinput.Model
	editInput       textinput.Model // text input for editing an existing root's interval
	editNameInput   textinput.Model // text input for editing an existing root's name
	editFieldCursor int             // cursor within the edit-root field list

	step    wizardStep
	visible bool
	dirty   bool   // whether any modification has been made
	err     string // validation error shown inline

	// root list cursor (for wizardStepRoots)
	rootCursor int

	// index of the root being edited (for wizardStepEditRoot)
	editIndex int

	// pending values for the root being added
	pendingPath     string
	pendingName     string
	pendingInterval time.Duration

	// defaults loaded from the embedded config
	defaultPath     string
	defaultInterval string

	width  int
	height int
}

func newConfigWizard(cfg *config.Config) configWizard {
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

	return configWizard{
		cfg:             cfg,
		pathInput:       pi,
		nameInput:       ni,
		intervalInput:   ii,
		editInput:       ei,
		editNameInput:   eni,
		step:            wizardStepRoots,
		defaultPath:     defPath,
		defaultInterval: defInterval,
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
func (w *configWizard) Show() tea.Cmd {
	w.visible = true
	w.err = ""
	w.dirty = false
	w.rootCursor = 0

	if len(w.cfg.Roots) > 0 {
		w.step = wizardStepRoots
		w.blurAll()

		return nil
	}

	// No roots — jump to add-new flow.
	w.step = wizardStepPath
	w.pathInput.SetValue(w.defaultPath)
	w.nameInput.SetValue("")
	w.intervalInput.SetValue("")
	w.pathInput.Focus()

	return textinput.Blink
}

// Hide closes the wizard overlay.
func (w *configWizard) Hide() {
	w.visible = false
	w.blurAll()
}

func (w *configWizard) blurAll() {
	w.pathInput.Blur()
	w.nameInput.Blur()
	w.intervalInput.Blur()
	w.editInput.Blur()
	w.editNameInput.Blur()
}

// SetSize adjusts the wizard popup dimensions.
func (w *configWizard) SetSize(termWidth, termHeight int) {
	w.width = termWidth * 3 / 4
	if w.width < 50 {
		w.width = min(50, termWidth)
	}

	w.height = termHeight / 2
	if w.height < 14 {
		w.height = min(14, termHeight)
	}

	innerW := w.width - 8 // borders + padding
	w.pathInput.Width = innerW
	w.nameInput.Width = innerW
	w.intervalInput.Width = innerW
	w.editInput.Width = innerW
	w.editNameInput.Width = innerW
}

// Update handles messages while the wizard is visible.
func (w *configWizard) Update(msg tea.Msg) (tea.Cmd, *configWizardMsg) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return w.handleKey(msg)
	}

	// Forward to the active text input.
	var cmd tea.Cmd

	switch w.step {
	case wizardStepPath:
		w.pathInput, cmd = w.pathInput.Update(msg)
	case wizardStepName:
		w.nameInput, cmd = w.nameInput.Update(msg)
	case wizardStepInterval:
		w.intervalInput, cmd = w.intervalInput.Update(msg)
	case wizardStepEditInterval:
		w.editInput, cmd = w.editInput.Update(msg)
	case wizardStepEditName:
		w.editNameInput, cmd = w.editNameInput.Update(msg)
	}

	return cmd, nil
}

func (w *configWizard) handleKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	// Esc behavior depends on context.
	if key == "esc" {
		return w.handleEsc()
	}

	switch w.step {
	case wizardStepRoots:
		return w.handleRootsKey(msg)
	case wizardStepEditRoot:
		return w.handleEditRootKey(msg)
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

		return nil, &configWizardMsg{cfg: w.cfg}
	}

	return nil, nil
}

func (w *configWizard) handleEsc() (tea.Cmd, *configWizardMsg) {
	switch w.step {
	case wizardStepEditRoot, wizardStepEditName, wizardStepEditInterval:
		// Go back to root list without saving the edit.
		w.step = wizardStepRoots
		w.err = ""
		w.blurAll()

		return nil, nil

	case wizardStepPath, wizardStepName, wizardStepInterval, wizardStepConfirm:
		if len(w.cfg.Roots) > 0 {
			// Go back to root list.
			w.step = wizardStepRoots
			w.err = ""
			w.blurAll()

			return nil, nil
		}

		// No roots — Esc closes the wizard entirely.
		w.Hide()

		return nil, nil

	default:
		w.Hide()

		if w.dirty {
			return nil, &configWizardMsg{cfg: w.cfg}
		}

		return nil, nil
	}
}

// handleRootsKey handles keyboard input on the root list step.
func (w *configWizard) handleRootsKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()
	n := len(w.cfg.Roots)

	switch key {
	case "up", "k":
		if w.rootCursor > 0 {
			w.rootCursor--
		}

	case "down", "j":
		if w.rootCursor < n-1 {
			w.rootCursor++
		}

	case "enter":
		if n == 0 {
			return nil, nil
		}

		// Enter edit mode for the selected root (field selection).
		w.editIndex = w.rootCursor
		w.editFieldCursor = 0
		w.step = wizardStepEditRoot
		w.err = ""

		return nil, nil

	case "a", "A":
		// Start add-new-root flow.
		w.step = wizardStepPath
		w.err = ""
		w.pathInput.SetValue(w.defaultPath)
		w.nameInput.SetValue("")
		w.intervalInput.SetValue("")
		w.pathInput.Focus()

		return textinput.Blink, nil

	case "s", "S":
		if !w.dirty {
			return nil, nil
		}

		if err := w.cfg.Save(); err != nil {
			w.err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		w.err = ""
		w.step = wizardStepDone

		return nil, nil
	}

	return nil, nil
}

// editRootFields defines the fields available for editing on a root.
var editRootFields = []string{"Name", "Interval"} //nolint:gochecknoglobals // wizard field list

// handleEditRootKey handles the field-selection menu for editing a root.
func (w *configWizard) handleEditRootKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	switch key {
	case "up", "k":
		if w.editFieldCursor > 0 {
			w.editFieldCursor--
		}
	case "down", "j":
		if w.editFieldCursor < len(editRootFields)-1 {
			w.editFieldCursor++
		}
	case "enter":
		root := w.cfg.Roots[w.editIndex]

		switch editRootFields[w.editFieldCursor] {
		case "Name":
			w.step = wizardStepEditName
			w.err = ""
			w.editNameInput.SetValue(w.cfg.RootDisplayName(w.editIndex))
			w.editNameInput.Focus()

			return textinput.Blink, nil
		case "Interval":
			w.step = wizardStepEditInterval
			w.err = ""
			w.editInput.SetValue(root.RootConfig.ScheduleInterval.String())
			w.editInput.Focus()

			return textinput.Blink, nil
		}
	}

	return nil, nil
}

// handleEditNameKey handles keyboard input when editing a root's display name.
func (w *configWizard) handleEditNameKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		name := strings.TrimSpace(w.editNameInput.Value())
		// Empty name is valid — it will fall back to basename(path).
		w.cfg.UpdateRootName(w.editIndex, name)
		w.dirty = true
		w.err = ""
		w.step = wizardStepEditRoot
		w.editNameInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.editNameInput, cmd = w.editNameInput.Update(msg)

		return cmd, nil
	}
}

// handleEditIntervalKey handles keyboard input when editing a root's schedule interval.
func (w *configWizard) handleEditIntervalKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		raw := strings.TrimSpace(w.editInput.Value())
		if raw == "" {
			w.err = "Interval cannot be empty."

			return nil, nil
		}

		d, err := time.ParseDuration(raw)
		if err != nil {
			w.err = fmt.Sprintf("Invalid duration %q — use Go syntax (e.g. 24h, 30m).", raw)

			return nil, nil
		}

		if d <= 0 {
			w.err = "Interval must be positive."

			return nil, nil
		}

		// Apply the change.
		w.cfg.UpdateRootInterval(w.editIndex, d)
		w.dirty = true
		w.err = ""
		w.step = wizardStepEditRoot
		w.editInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.editInput, cmd = w.editInput.Update(msg)

		return cmd, nil
	}
}

func (w *configWizard) handlePathKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		path := strings.TrimSpace(w.pathInput.Value())
		if path == "" {
			w.err = "Path cannot be empty."

			return nil, nil
		}

		// Expand ~ to home directory.
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				w.err = fmt.Sprintf("Cannot resolve home directory: %v", err)

				return nil, nil
			}

			path = filepath.Join(home, path[2:])
		}

		// Convert to absolute path.
		absPath, err := filepath.Abs(path)
		if err != nil {
			w.err = fmt.Sprintf("Cannot resolve absolute path: %v", err)

			return nil, nil
		}

		path = absPath

		// Validate the path exists and is a directory.
		info, err := os.Stat(path)
		if err != nil {
			w.err = fmt.Sprintf("Cannot access path: %v", err)

			return nil, nil
		}

		if !info.IsDir() {
			w.err = "Path is not a directory."

			return nil, nil
		}

		w.pendingPath = path
		w.err = ""
		w.step = wizardStepName
		w.pathInput.Blur()

		// Pre-fill name with the basename of the path.
		w.nameInput.SetValue(filepath.Base(path))
		w.nameInput.Focus()

		return textinput.Blink, nil

	default:
		var cmd tea.Cmd
		w.pathInput, cmd = w.pathInput.Update(msg)

		return cmd, nil
	}
}

// handleNameKey handles keyboard input on the name step (add-new flow).
func (w *configWizard) handleNameKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		name := strings.TrimSpace(w.nameInput.Value())
		// Empty name is valid — AddRoot defaults to basename(path).
		w.pendingName = name
		w.err = ""
		w.step = wizardStepInterval
		w.nameInput.Blur()
		w.intervalInput.Focus()

		return textinput.Blink, nil

	default:
		var cmd tea.Cmd
		w.nameInput, cmd = w.nameInput.Update(msg)

		return cmd, nil
	}
}

func (w *configWizard) handleIntervalKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	switch key {
	case "enter":
		raw := strings.TrimSpace(w.intervalInput.Value())
		if raw == "" {
			raw = w.defaultInterval
		}

		d, err := time.ParseDuration(raw)
		if err != nil {
			w.err = fmt.Sprintf("Invalid duration %q — use Go duration syntax (e.g. 24h, 30m, 168h).", raw)

			return nil, nil
		}

		if d <= 0 {
			w.err = "Interval must be positive."

			return nil, nil
		}

		w.pendingInterval = d
		w.err = ""
		w.step = wizardStepConfirm
		w.intervalInput.Blur()

		return nil, nil

	default:
		var cmd tea.Cmd
		w.intervalInput, cmd = w.intervalInput.Update(msg)

		return cmd, nil
	}
}

func (w *configWizard) handleConfirmKey(msg tea.KeyMsg) (tea.Cmd, *configWizardMsg) {
	key := msg.String()

	switch key {
	case "y", "Y", "enter":
		// Commit the root to the working config.
		w.cfg.AddRoot(w.pendingName, w.pendingPath, w.pendingInterval)
		w.dirty = true

		// Save to disk.
		if err := w.cfg.Save(); err != nil {
			w.err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		w.err = ""
		w.step = wizardStepDone

		return nil, nil

	case "a", "A":
		// Add another root — save first, then restart add flow.
		w.cfg.AddRoot(w.pendingName, w.pendingPath, w.pendingInterval)
		w.dirty = true

		if err := w.cfg.Save(); err != nil {
			w.err = fmt.Sprintf("Failed to save: %v", err)

			return nil, nil
		}

		// Reset for another entry.
		w.step = wizardStepPath
		w.err = ""
		w.pathInput.SetValue(w.defaultPath)
		w.nameInput.SetValue("")
		w.intervalInput.SetValue("")
		w.pathInput.Focus()

		return textinput.Blink, nil

	case "n", "N":
		// Cancel this entry — go back to root list if roots exist.
		if len(w.cfg.Roots) > 0 {
			w.step = wizardStepRoots
			w.err = ""
			w.blurAll()

			return nil, nil
		}

		w.Hide()

		return nil, nil
	}

	return nil, nil
}

// View renders the wizard overlay.
func (w *configWizard) View(termWidth, termHeight int) string {
	if !w.visible {
		return ""
	}

	var content strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).
		Render("Configuration Wizard")
	content.WriteString(title + "\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("Configure root directories for git-janitor to scan.") + "\n\n")

	switch w.step {
	case wizardStepRoots:
		w.viewRoots(&content)
	case wizardStepEditRoot:
		w.viewEditRoot(&content)
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
	if w.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		content.WriteString("\n" + errStyle.Render("  ⚠ "+w.err) + "\n")
	}

	// Footer.
	w.viewFooter(&content)

	// Wrap in a bordered box.
	border := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("170")).
		Width(w.width-2).
		Padding(1, 2)

	popup := border.Render(content.String())

	return lipgloss.Place(
		termWidth, termHeight,
		lipgloss.Center, lipgloss.Center,
		popup,
	)
}

func (w *configWizard) viewRoots(content *strings.Builder) {
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).
		Render(fmt.Sprintf("Configured roots (%d):", len(w.cfg.Roots)))
	content.WriteString(header + "\n")

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)

	for i := range w.cfg.Roots {
		cursor := "  "
		style := dim
		if i == w.rootCursor {
			cursor = "▸ "
			style = selected
		}

		name := w.cfg.RootDisplayName(i)
		r := w.cfg.Roots[i]
		line := fmt.Sprintf("%s%s  %s  (every %s)", cursor, name, r.Path, r.RootConfig.ScheduleInterval)
		content.WriteString(style.Render(line) + "\n")
	}

	content.WriteString("\n")

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(hint.Render("  ↑/↓ select   Enter edit   [A] add new") + "\n")
}

func (w *configWizard) viewEditRoot(content *strings.Builder) {
	if w.editIndex < 0 || w.editIndex >= len(w.cfg.Roots) {
		return
	}

	root := w.cfg.Roots[w.editIndex]
	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))

	content.WriteString(heading.Render(fmt.Sprintf("Editing root: %s", w.cfg.RootDisplayName(w.editIndex))) + "\n\n")

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)

	fields := []struct {
		label string
		value string
	}{
		{"Name", w.cfg.RootDisplayName(w.editIndex)},
		{"Interval", root.RootConfig.ScheduleInterval.String()},
	}

	for i, f := range fields {
		cursor := "  "
		style := dim
		if i == w.editFieldCursor {
			cursor = "▸ "
			style = selected
		}

		line := fmt.Sprintf("%s%-10s %s", cursor, f.label+":", f.value)
		content.WriteString(style.Render(line) + "\n")
	}

	content.WriteString("\n")

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(hint.Render("  ↑/↓ select   Enter edit field   Esc back") + "\n")
}

func (w *configWizard) viewEditName(content *strings.Builder) {
	if w.editIndex < 0 || w.editIndex >= len(w.cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render(fmt.Sprintf("Editing name for: %s", w.cfg.Roots[w.editIndex].Path)) + "\n\n")
	content.WriteString("  Display name (leave empty to use directory name):\n\n")
	content.WriteString(w.editNameInput.View() + "\n")
}

func (w *configWizard) viewEditInterval(content *strings.Builder) {
	if w.editIndex < 0 || w.editIndex >= len(w.cfg.Roots) {
		return
	}

	heading := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	content.WriteString(heading.Render(fmt.Sprintf("Editing interval for: %s", w.cfg.RootDisplayName(w.editIndex))) + "\n\n")
	content.WriteString("  Schedule interval (Go duration syntax, e.g. 24h, 30m):\n\n")
	content.WriteString(w.editInput.View() + "\n")
}

func (w *configWizard) viewPath(content *strings.Builder) {
	// Show existing roots if any (context for "add another").
	if len(w.cfg.Roots) > 0 {
		existing := lipgloss.NewStyle().Foreground(lipgloss.Color("63")).
			Render(fmt.Sprintf("Configured roots (%d):", len(w.cfg.Roots)))
		content.WriteString(existing + "\n")

		for i, r := range w.cfg.Roots {
			name := w.cfg.RootDisplayName(i)
			fmt.Fprintf(content, "  • %s  %s  (every %s)\n", name, r.Path, r.RootConfig.ScheduleInterval)
		}

		content.WriteString("\n")
	}

	content.WriteString("Add root — Path\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  Enter an absolute path to a directory containing git repos.") + "\n\n")
	content.WriteString(w.pathInput.View() + "\n")
}

func (w *configWizard) viewName(content *strings.Builder) {
	fmt.Fprintf(content, "Add root — Name for %s\n", w.pendingPath)
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  Display name for this root (leave empty to use directory name).") + "\n\n")
	content.WriteString(w.nameInput.View() + "\n")
}

func (w *configWizard) viewInterval(content *strings.Builder) {
	fmt.Fprintf(content, "Add root — Interval for %s\n", w.pendingPath)
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render("  How often should this root be checked? (default: "+w.defaultInterval+")") + "\n\n")
	content.WriteString(w.intervalInput.View() + "\n")
}

func (w *configWizard) viewConfirm(content *strings.Builder) {
	content.WriteString("Add root — Review\n\n")

	displayName := w.pendingName
	if displayName == "" {
		displayName = filepath.Base(w.pendingPath) + " (default)"
	}

	fmt.Fprintf(content, "  Name:     %s\n", displayName)
	fmt.Fprintf(content, "  Path:     %s\n", w.pendingPath)
	fmt.Fprintf(content, "  Interval: %s\n\n", w.pendingInterval)

	actions := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	content.WriteString(actions.Render("  [Y/Enter] Save & close") + "  ")
	content.WriteString(actions.Render("[A] Save & add another") + "  ")
	content.WriteString(actions.Render("[N] Cancel") + "\n")
}

func (w *configWizard) viewDone(content *strings.Builder) {
	checkmark := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	fmt.Fprintf(content, "\n  %s Configuration saved!\n", checkmark)

	path, _ := config.DefaultConfigPath()
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).
		Render(fmt.Sprintf("    Written to %s", path)) + "\n\n")
	content.WriteString("  Press any key to close.\n")
}

func (w *configWizard) viewFooter(content *strings.Builder) {
	if w.step == wizardStepDone {
		return
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var parts []string
	if w.dirty {
		save := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).
			Render("[S] Save")
		parts = append(parts, save)
	}

	parts = append(parts, hint.Render("Esc to cancel"))

	content.WriteString("\n  " + strings.Join(parts, "  ") + "\n")
}

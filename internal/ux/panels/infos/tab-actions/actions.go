// SPDX-License-Identifier: Apache-2.0

package actions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/ifaces"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/key"
	"github.com/fredbi/git-janitor/internal/ux/panels"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// cardLines is the number of display lines per suggestion card.
const cardLines = 4

// Panel displays suggested actions for the currently selected alert.
type Panel struct {
	panels.Base

	RepoPath    string
	Alert       *models.Alert
	Suggestions []models.ActionSuggestion

	// Picker state (non-nil when Ctrl+P is active).
	Picker *SubjectPicker
	// ParamInput state (non-nil when an action requires user input).
	ParamInput *ParamInputState
	Engine     ifaces.Engineer
}

// ParamInputState holds the text input state for actions that need a parameter.
type ParamInputState struct {
	Suggestion models.ActionSuggestion
	Prompt     string
	Input      textinput.Model
}

// SubjectPicker is an overlay for selecting individual subjects.
type SubjectPicker struct {
	Suggestion models.ActionSuggestion
	Checked    []bool // one per subject
	Cursor     int
	Offset     int
}

// New creates an empty Panel.
func New(eng ifaces.Engineer, theme *types.Theme) Panel {
	return Panel{
		Base:   panels.Base{Theme: theme},
		Engine: eng,
	}
}

// SetAlert sets the alert whose suggestions are displayed.
func (p *Panel) SetAlert(repoPath string, alert *models.Alert) {
	p.RepoPath = repoPath
	p.Picker = nil

	if alert == nil || len(alert.Suggestions) == 0 {
		p.Alert = nil
		p.Suggestions = nil
		p.ResetScroll()

		return
	}

	p.Alert = alert
	p.Suggestions = alert.Suggestions
	p.ResetScroll()
}

// IsCapturingInput reports whether the panel has an active text input
// that should capture all key events (preventing parent key bindings).
func (p *Panel) IsCapturingInput() bool {
	return p.ParamInput != nil
}

// Clear removes all suggestions.
func (p *Panel) Clear() {
	p.Alert = nil
	p.Suggestions = nil
	p.Picker = nil
	p.ParamInput = nil
	p.ResetScroll()
}

func (p *Panel) SetSize(w, h int) {
	p.Base.SetSize(w, h, 2, cardLines)
}

func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	// When the param input is active, it captures all messages.
	if p.ParamInput != nil {
		return p.updateParamInput(msg)
	}

	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	// When the picker is active, it captures all keys.
	if p.Picker != nil {
		return p.updatePicker(km)
	}

	if len(p.Suggestions) == 0 {
		return nil
	}

	if p.NavigateKey(km, len(p.Suggestions)) {
		p.ClampScroll(p.visibleCards())

		return nil
	}

	switch key.MsgBinding(km) {
	case key.Enter:
		// Execute on ALL subjects.
		if p.Cursor >= 0 && p.Cursor < len(p.Suggestions) {
			sug := p.Suggestions[p.Cursor]

			// If the action needs a parameter, show the input overlay.
			if cmd := p.maybeShowParamInput(sug); cmd != nil {
				return cmd
			}

			return func() tea.Msg {
				return types.ExecuteActionMsg{
					RepoPath:   p.RepoPath,
					ActionName: sug.ActionName,
					Subjects:   sug.Subjects,
				}
			}
		}
	case key.CtrlP:
		// Open subject picker for multi-subject suggestions.
		if p.Cursor >= 0 && p.Cursor < len(p.Suggestions) {
			sug := p.Suggestions[p.Cursor]
			if len(sug.Subjects) > 1 {
				checked := make([]bool, len(sug.Subjects))
				for i := range checked {
					checked[i] = true // all pre-checked
				}

				p.Picker = &SubjectPicker{
					Suggestion: sug,
					Checked:    checked,
				}
			}
		}
	}

	return nil
}

func (p *Panel) View() string {
	// If param input is active, render it instead of the card list.
	if p.ParamInput != nil {
		return p.viewParamInput()
	}

	// If picker is active, render it instead of the card list.
	if p.Picker != nil {
		return p.viewPicker()
	}

	return p.viewCards()
}

func (p *Panel) updatePicker(km tea.KeyMsg) tea.Cmd {
	pk := p.Picker
	n := len(pk.Suggestion.Subjects)

	switch key.MsgBinding(km) {
	case key.Up, key.K:
		if pk.Cursor > 0 {
			pk.Cursor--
			p.clampPickerScroll()
		}
	case key.Down, key.J:
		if pk.Cursor < n-1 {
			pk.Cursor++
			p.clampPickerScroll()
		}
	case key.Home, key.G:
		pk.Cursor = 0
		p.clampPickerScroll()
	case key.End, key.GG:
		pk.Cursor = max(0, n-1)
		p.clampPickerScroll()
	case key.Space:
		// Toggle checkbox.
		if pk.Cursor >= 0 && pk.Cursor < n {
			pk.Checked[pk.Cursor] = !pk.Checked[pk.Cursor]
		}
	case key.Enter:
		// Execute with selected subjects only.
		var selected []models.ActionSubject

		for i, subject := range pk.Suggestion.Subjects {
			if pk.Checked[i] {
				selected = append(selected, subject)
			}
		}

		p.Picker = nil

		if len(selected) == 0 {
			return nil // nothing selected, do nothing
		}

		return func() tea.Msg {
			return types.ExecuteActionMsg{
				RepoPath:   p.RepoPath,
				ActionName: pk.Suggestion.ActionName,
				Subjects:   selected,
			}
		}
	case key.Esc:
		p.Picker = nil
	case key.A:
		// Select all.
		for i := range pk.Checked {
			pk.Checked[i] = true
		}
	case key.N:
		// Select none.
		for i := range pk.Checked {
			pk.Checked[i] = false
		}
	}

	return nil
}

func (p *Panel) visibleCards() int {
	return max(p.Height/cardLines, 1)
}

const pickerReservedLines = 3 // header + hint + separator

func (p *Panel) clampPickerScroll() {
	pk := p.Picker
	visibleRows := max(p.Height-pickerReservedLines, 1)

	if pk.Cursor < pk.Offset {
		pk.Offset = pk.Cursor
	}

	if pk.Cursor >= pk.Offset+visibleRows {
		pk.Offset = pk.Cursor - visibleRows + 1
	}
}

func (p *Panel) viewPicker() string {
	pk := p.Picker
	t := p.Theme
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText)
	detailStyle := lipgloss.NewStyle().Foreground(t.Dim)
	selectedBg := lipgloss.NewStyle().Bold(true).Foreground(t.Bright).Background(t.SelectedBg)
	accentStyle := lipgloss.NewStyle().Foreground(t.ActionsAccent)

	// Count selected.
	selectedCount := 0

	for _, c := range pk.Checked {
		if c {
			selectedCount++
		}
	}

	header := headerStyle.Render(fmt.Sprintf("  Pick %ss for: %s",
		pk.Suggestion.SubjectKind, accentStyle.Render(pk.Suggestion.ActionName)))
	countLine := detailStyle.Render(fmt.Sprintf("  %d/%d selected", selectedCount, len(pk.Suggestion.Subjects)))

	visibleRows := max(p.Height-pickerReservedLines, 1)

	end := min(pk.Offset+visibleRows, len(pk.Suggestion.Subjects))

	var rows []string

	for i := pk.Offset; i < end; i++ {
		name := pk.Suggestion.Subjects[i].Subject

		checkbox := "  ☐ "
		if pk.Checked[i] {
			checkbox = "  ☑ "
		}

		row := checkbox + name

		if i == pk.Cursor {
			row = selectedBg.Render(row)
		}

		rows = append(rows, row)
	}

	rows = panels.PadRows(rows, visibleRows)

	hint := detailStyle.Render("  Space: toggle  |  a: all  n: none  |  Enter: execute  |  Esc: cancel")

	return header + "\n" + countLine + "\n" + strings.Join(rows, "\n") + "\n" + hint
}

func (p *Panel) viewCards() string {
	t := p.Theme
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText)
	detailStyle := lipgloss.NewStyle().Foreground(t.Dim)
	selectedBg := lipgloss.NewStyle().Background(t.SelectedBg)
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(t.ActionsAccent)
	warnStyle := lipgloss.NewStyle().Foreground(t.Warning)

	header := headerStyle.Render("  Suggested Actions")
	if p.Alert != nil {
		header += "  " + detailStyle.Render("for: "+p.Alert.Summary)
	}

	if len(p.Suggestions) == 0 {
		msg := detailStyle.Render("  Select an alert and press Enter to see suggested actions")
		if p.Alert != nil {
			msg = detailStyle.Render("  No suggested actions for this alert")
		}

		return header + "\n" + msg
	}

	visible := p.visibleCards()

	var rows []string

	start, end := p.VisibleRange(len(p.Suggestions), visible)
	contentWidth := p.Width - 4

	for i := start; i < end; i++ {
		sug := p.Suggestions[i]
		selected := i == p.Cursor

		// Line 1: action name + destructive indicator
		nameStr := accentStyle.Render(sug.ActionName)

		destructive := false
		found := true

		if action, ok := p.Engine.GetAction(sug.ActionName); ok {
			destructive = action.Destructive()
		} else {
			nameStr += "  " + warnStyle.Render("⚠️  error: suggested action not found")
			found = false
		}

		if destructive {
			nameStr += "  " + warnStyle.Render("⚠️  destructive")
		} else if found {
			nameStr += "  ✅"
		}

		// Line 2: description from registry
		descStr := detailStyle.Render("  (no description)")

		if action, ok := p.Engine.GetAction(sug.ActionName); ok {
			descStr = detailStyle.Render(action.Description())
		}

		// Line 3: repo (short path) + subject kind
		repoShort := filepath.Base(p.RepoPath)
		line3 := fmt.Sprintf("repo: %s  |  %s", repoShort, sug.SubjectKind)

		// Line 4: subjects
		subjects := strings.Join(sug.SubjectNames(), ", ")
		if len(subjects) > contentWidth && contentWidth > 10 {
			subjects = subjects[:contentWidth-3] + "..."
		}

		if subjects == "" {
			subjects = "(entire repo)"
		}

		line4 := detailStyle.Render("→ " + subjects)

		card := fmt.Sprintf("  %s\n  %s\n  %s\n  %s",
			nameStr, descStr, line3, line4)

		if selected {
			card = selectedBg.Width(p.Width).Render(card)
		}

		rows = append(rows, card)
	}

	usedLines := len(rows) * cardLines
	rows = panels.PadRows(rows, usedLines+(p.Height-usedLines))

	// Adapt hint based on whether the selected suggestion has multiple subjects.
	hint := "  Enter: execute  |  ↑↓: navigate"

	if p.Cursor >= 0 && p.Cursor < len(p.Suggestions) && len(p.Suggestions[p.Cursor].Subjects) > 1 {
		kind := p.Suggestions[p.Cursor].SubjectKind.String() + "s"
		hint = fmt.Sprintf("  Enter: execute all %s  |  Ctrl+P: pick %s  |  ↑↓: navigate", kind, kind)
	}

	hint = detailStyle.Render(hint)

	return header + "\n" + strings.Join(rows, "\n") + "\n" + hint
}

// maybeShowParamInput checks if the action needs a parameter input.
// Returns a focus command if input is shown, nil otherwise.
func (p *Panel) maybeShowParamInput(sug models.ActionSuggestion) tea.Cmd {
	action, ok := p.Engine.GetAction(sug.ActionName)
	if !ok {
		return nil
	}

	prompt := action.ParamPrompt()
	if prompt == "" {
		return nil
	}

	ti := textinput.New()
	ti.Placeholder = "type here..."
	ti.Prompt = " " + prompt + " "
	ti.CharLimit = 256 //nolint:mnd // reasonable max for a description
	ti.Width = max(p.Width-len(prompt)-6, 20) //nolint:mnd // account for prompt + padding

	p.ParamInput = &ParamInputState{
		Suggestion: sug,
		Prompt:     prompt,
		Input:      ti,
	}

	return p.ParamInput.Input.Focus()
}

// updateParamInput handles messages while the param input is active.
func (p *Panel) updateParamInput(msg tea.Msg) tea.Cmd {
	pi := p.ParamInput

	if km, ok := msg.(tea.KeyMsg); ok {
		switch key.MsgBinding(km) {
		case key.Enter:
			value := strings.TrimSpace(pi.Input.Value())
			p.ParamInput = nil

			if value == "" {
				return nil // empty input, cancel
			}

			// Inject the user input as a param on the first subject.
			subjects := make([]models.ActionSubject, len(pi.Suggestion.Subjects))
			copy(subjects, pi.Suggestion.Subjects)

			if len(subjects) > 0 {
				subjects[0].Params = []string{value}
			} else {
				subjects = []models.ActionSubject{{
					Subject: value,
					Params:  []string{value},
				}}
			}

			return func() tea.Msg {
				return types.ExecuteActionMsg{
					RepoPath:   p.RepoPath,
					ActionName: pi.Suggestion.ActionName,
					Subjects:   subjects,
				}
			}

		case key.Esc:
			p.ParamInput = nil

			return nil
		}
	}

	var cmd tea.Cmd
	pi.Input, cmd = pi.Input.Update(msg)

	return cmd
}

// viewParamInput renders the parameter input overlay.
func (p *Panel) viewParamInput() string {
	pi := p.ParamInput
	t := p.Theme

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText)
	detailStyle := lipgloss.NewStyle().Foreground(t.Dim)
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(t.ActionsAccent)

	header := headerStyle.Render("  " + accentStyle.Render(pi.Suggestion.ActionName))

	inputView := pi.Input.View()

	hint := detailStyle.Render("  Enter: submit  |  Esc: cancel")

	content := header + "\n\n" + inputView + "\n\n" + hint
	padding := max(p.Height-5, 0) //nolint:mnd // header + blank + input + blank + hint

	return content + strings.Repeat("\n", padding)
}

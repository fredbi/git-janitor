package actions

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/engine"
	"github.com/fredbi/git-janitor/internal/ux/types"
)

// cardLines is the number of display lines per suggestion card.
const cardLines = 4

// ActionsListPanel displays suggested actions for the currently selected alert.
type ActionsListPanel struct {
	RepoPath    string
	Alert       *engine.Alert
	Suggestions []engine.ActionSuggestion
	Actions     *engine.ActionRegistry
	Cursor      int
	Offset      int
	Width       int
	Height      int

	// Picker state (non-nil when Ctrl+P is active).
	Picker *SubjectPicker
}

// SubjectPicker is an overlay for selecting individual subjects.
type SubjectPicker struct {
	Suggestion engine.ActionSuggestion
	Checked    []bool // one per subject
	Cursor     int
	Offset     int
}

// New creates an empty ActionsListPanel.
func New() ActionsListPanel {
	return ActionsListPanel{}
}

// SetAlert sets the alert whose suggestions are displayed.
func (p *ActionsListPanel) SetAlert(repoPath string, alert *engine.Alert) {
	p.RepoPath = repoPath
	p.Picker = nil

	if alert == nil || len(alert.Suggestions) == 0 {
		p.Alert = nil
		p.Suggestions = nil
		p.Cursor = 0
		p.Offset = 0

		return
	}

	p.Alert = alert
	p.Suggestions = alert.Suggestions
	p.Cursor = 0
	p.Offset = 0
}

// Clear removes all suggestions.
func (p *ActionsListPanel) Clear() {
	p.Alert = nil
	p.Suggestions = nil
	p.Picker = nil
	p.Cursor = 0
	p.Offset = 0
}

func (p *ActionsListPanel) SetSize(w, h int) {
	p.Width = w
	p.Height = h - 2

	if p.Height < cardLines {
		p.Height = cardLines
	}
}

func (p *ActionsListPanel) Update(msg tea.Msg) tea.Cmd {
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

	switch km.String() {
	case "up", "k":
		if p.Cursor > 0 {
			p.Cursor--
			p.clampScroll()
		}
	case "down", "j":
		if p.Cursor < len(p.Suggestions)-1 {
			p.Cursor++
			p.clampScroll()
		}
	case "enter":
		// Execute on ALL subjects.
		if p.Cursor >= 0 && p.Cursor < len(p.Suggestions) {
			sug := p.Suggestions[p.Cursor]

			return func() tea.Msg {
				return types.ExecuteActionMsg{
					RepoPath:   p.RepoPath,
					ActionName: sug.ActionName,
					Subjects:   sug.Subjects,
				}
			}
		}
	case "ctrl+p":
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

func (p *ActionsListPanel) updatePicker(km tea.KeyMsg) tea.Cmd {
	pk := p.Picker
	n := len(pk.Suggestion.Subjects)

	switch km.String() {
	case "up", "k":
		if pk.Cursor > 0 {
			pk.Cursor--
			p.clampPickerScroll()
		}
	case "down", "j":
		if pk.Cursor < n-1 {
			pk.Cursor++
			p.clampPickerScroll()
		}
	case " ", "space":
		// Toggle checkbox.
		if pk.Cursor >= 0 && pk.Cursor < n {
			pk.Checked[pk.Cursor] = !pk.Checked[pk.Cursor]
		}
	case "enter":
		// Execute with selected subjects only.
		var selected []string

		for i, name := range pk.Suggestion.Subjects {
			if pk.Checked[i] {
				selected = append(selected, name)
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
	case "esc":
		p.Picker = nil
	case "a":
		// Select all.
		for i := range pk.Checked {
			pk.Checked[i] = true
		}
	case "n":
		// Select none.
		for i := range pk.Checked {
			pk.Checked[i] = false
		}
	}

	return nil
}

func (p *ActionsListPanel) clampScroll() {
	visibleCards := p.Height / cardLines
	if visibleCards < 1 {
		visibleCards = 1
	}

	if p.Cursor < p.Offset {
		p.Offset = p.Cursor
	}

	if p.Cursor >= p.Offset+visibleCards {
		p.Offset = p.Cursor - visibleCards + 1
	}
}

func (p *ActionsListPanel) clampPickerScroll() {
	pk := p.Picker
	visibleRows := p.Height - 3 // header + hint + separator
	if visibleRows < 1 {
		visibleRows = 1
	}

	if pk.Cursor < pk.Offset {
		pk.Offset = pk.Cursor
	}

	if pk.Cursor >= pk.Offset+visibleRows {
		pk.Offset = pk.Cursor - visibleRows + 1
	}
}

func (p *ActionsListPanel) View() string {
	// If picker is active, render it instead of the card list.
	if p.Picker != nil {
		return p.viewPicker()
	}

	return p.viewCards()
}

func (p *ActionsListPanel) viewPicker() string {
	pk := p.Picker
	t := types.CurrentTheme
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

	visibleRows := p.Height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}

	end := pk.Offset + visibleRows
	if end > len(pk.Suggestion.Subjects) {
		end = len(pk.Suggestion.Subjects)
	}

	var rows []string

	for i := pk.Offset; i < end; i++ {
		name := pk.Suggestion.Subjects[i]

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

	for len(rows) < visibleRows {
		rows = append(rows, "")
	}

	hint := detailStyle.Render("  Space: toggle  |  a: all  n: none  |  Enter: execute  |  Esc: cancel")

	return header + "\n" + countLine + "\n" + strings.Join(rows, "\n") + "\n" + hint
}

func (p *ActionsListPanel) viewCards() string {
	t := types.CurrentTheme
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

	visibleCards := p.Height / cardLines
	if visibleCards < 1 {
		visibleCards = 1
	}

	var rows []string

	end := p.Offset + visibleCards
	if end > len(p.Suggestions) {
		end = len(p.Suggestions)
	}

	contentWidth := p.Width - 4

	for i := p.Offset; i < end; i++ {
		sug := p.Suggestions[i]
		selected := i == p.Cursor

		// Line 1: action name + destructive indicator
		nameStr := accentStyle.Render(sug.ActionName)

		destructive := false

		if p.Actions != nil {
			if action, ok := p.Actions.Get(sug.ActionName); ok {
				destructive = action.Destructive()
			}
		}

		if destructive {
			nameStr += "  " + warnStyle.Render("⚠️  destructive")
		} else {
			nameStr += "  ✅"
		}

		// Line 2: description from registry
		descStr := detailStyle.Render("  (no description)")

		if p.Actions != nil {
			if action, ok := p.Actions.Get(sug.ActionName); ok {
				descStr = detailStyle.Render(action.Description())
			}
		}

		// Line 3: repo (short path) + subject kind
		repoShort := filepath.Base(p.RepoPath)
		line3 := fmt.Sprintf("repo: %s  |  %s", repoShort, sug.SubjectKind)

		// Line 4: subjects
		subjects := strings.Join(sug.Subjects, ", ")
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
	for usedLines < p.Height {
		rows = append(rows, "")
		usedLines++
	}

	// Adapt hint based on whether the selected suggestion has multiple subjects.
	hint := "  Enter: execute  |  ↑↓: navigate"

	if p.Cursor >= 0 && p.Cursor < len(p.Suggestions) && len(p.Suggestions[p.Cursor].Subjects) > 1 {
		kind := p.Suggestions[p.Cursor].SubjectKind.String() + "s"
		hint = fmt.Sprintf("  Enter: execute all %s  |  Ctrl+P: pick %s  |  ↑↓: navigate", kind, kind)
	}

	hint = detailStyle.Render(hint)

	return header + "\n" + strings.Join(rows, "\n") + "\n" + hint
}

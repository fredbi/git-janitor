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
// Each suggestion is rendered as a multi-line card showing:
//   - action name + destructive indicator
//   - description from the action registry
//   - repo path (short)
//   - affected subjects
type ActionsListPanel struct {
	RepoPath    string
	Alert       *engine.Alert
	Suggestions []engine.ActionSuggestion
	Actions     *engine.ActionRegistry // for descriptions and destructive flags
	Cursor      int
	Offset      int // scroll offset in cards (not lines)
	Width       int
	Height      int
}

// New creates an empty ActionsListPanel.
func New() ActionsListPanel {
	return ActionsListPanel{}
}

// SetAlert sets the alert whose suggestions are displayed.
func (p *ActionsListPanel) SetAlert(repoPath string, alert *engine.Alert) {
	p.RepoPath = repoPath

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
	p.Cursor = 0
	p.Offset = 0
}

func (p *ActionsListPanel) SetSize(w, h int) {
	p.Width = w
	p.Height = h - 2 // reserve header + hint

	if p.Height < cardLines {
		p.Height = cardLines
	}
}

func (p *ActionsListPanel) Update(msg tea.Msg) tea.Cmd {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
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

func (p *ActionsListPanel) View() string {
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

	contentWidth := p.Width - 4 // indent

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

		// Compose the card.
		card := fmt.Sprintf("  %s\n  %s\n  %s\n  %s",
			nameStr, descStr, line3, line4)

		if selected {
			card = selectedBg.Width(p.Width).Render(card)
		}

		rows = append(rows, card)
	}

	// Pad remaining space.
	usedLines := len(rows) * cardLines
	for usedLines < p.Height {
		rows = append(rows, "")
		usedLines++
	}

	hint := detailStyle.Render("  Enter: execute  |  ↑↓: navigate")

	return header + "\n" + strings.Join(rows, "\n") + "\n" + hint
}

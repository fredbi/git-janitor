package gadgets

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/fs"
)

// defaultMaxAutocompleteItems caps the number of suggestions shown in the
// listbox so the popup stays compact.
const defaultMaxAutocompleteItems = 6

// PathAutocomplete wraps a [textinput.Model] and adds neovim-style
// directory autocompletion.
//
// As the user types into the input, suggestions for matching subdirectories
// of the parent directory are listed below the input. The user can:
//
//   - keep typing to refine the partial match,
//   - press Up/Down to navigate the list,
//   - press Enter (or Tab) to accept the highlighted suggestion: the partial
//     is replaced by the selected directory name followed by a slash, and
//     a new autocomplete cycle begins.
//
// When no suggestion is highlighted (the user has not navigated into the
// listbox), Enter is forwarded to the caller — typically to submit the form.
type PathAutocomplete struct {
	// Input is the underlying text input. Callers may configure its prompt,
	// placeholder, character limit, etc.
	Input textinput.Model

	suggestions []string
	cursor      int
	active      bool
	maxItems    int

	// styles is computed lazily so the gadget can be zero-value friendly.
	highlightStyle lipgloss.Style
	dimStyle       lipgloss.Style
}

// NewPathAutocomplete wraps an existing text input with path autocompletion.
func NewPathAutocomplete(in textinput.Model) PathAutocomplete {
	p := PathAutocomplete{
		Input:    in,
		maxItems: defaultMaxAutocompleteItems,
		highlightStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).Bold(true),
		dimStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
	p.Refresh()

	return p
}

// SetMaxItems overrides the maximum number of suggestions shown.
func (p *PathAutocomplete) SetMaxItems(n int) {
	if n > 0 {
		p.maxItems = n
	}
}

// Focus focuses the underlying input.
func (p *PathAutocomplete) Focus() tea.Cmd {
	return p.Input.Focus()
}

// Blur blurs the underlying input and dismisses the listbox highlight.
func (p *PathAutocomplete) Blur() {
	p.Input.Blur()
	p.active = false
}

// Focused reports whether the underlying input has focus.
func (p *PathAutocomplete) Focused() bool {
	return p.Input.Focused()
}

// Value returns the current input value (the path under construction).
func (p *PathAutocomplete) Value() string {
	return p.Input.Value()
}

// SetValue replaces the input value, moves the cursor to the end and
// recomputes suggestions.
func (p *PathAutocomplete) SetValue(s string) {
	p.Input.SetValue(s)
	p.Input.SetCursor(len(s))
	p.active = false
	p.Refresh()
}

// SetWidth sets the rendering width of the underlying input.
func (p *PathAutocomplete) SetWidth(w int) {
	p.Input.Width = w
}

// Suggestions returns the current list of suggestions (for inspection / tests).
func (p *PathAutocomplete) Suggestions() []string {
	return p.suggestions
}

// HighlightedIndex returns the index of the highlighted suggestion, or -1
// when the user has not navigated into the listbox.
func (p *PathAutocomplete) HighlightedIndex() int {
	if !p.active || len(p.suggestions) == 0 {
		return -1
	}

	return p.cursor
}

// Update routes a [tea.Msg] through the gadget.
//
// The returned `consumed` flag tells the caller whether the gadget fully
// handled the message. When false (only for Enter and Esc when nothing is
// selected) the caller may take its own action — typically submitting the
// form on Enter or cancelling on Esc.
func (p *PathAutocomplete) Update(msg tea.Msg) (tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		p.Input, cmd = p.Input.Update(msg)

		return cmd, true
	}

	switch keyMsg.Type {
	case tea.KeyUp:
		if len(p.suggestions) == 0 {
			return nil, true
		}

		p.active = true
		if p.cursor > 0 {
			p.cursor--
		} else {
			p.cursor = len(p.suggestions) - 1
		}

		return nil, true

	case tea.KeyDown:
		if len(p.suggestions) == 0 {
			return nil, true
		}

		p.active = true
		if p.cursor < len(p.suggestions)-1 {
			p.cursor++
		} else {
			p.cursor = 0
		}

		return nil, true

	case tea.KeyEnter:
		if p.active && len(p.suggestions) > 0 {
			p.accept(p.suggestions[p.cursor])

			return nil, true
		}

		// Fall through to the caller (form submission).
		return nil, false

	case tea.KeyTab:
		if len(p.suggestions) == 0 {
			return nil, true
		}

		if !p.active {
			p.cursor = 0
		}

		p.accept(p.suggestions[p.cursor])

		return nil, true

	case tea.KeyEsc:
		// Don't intercept Esc — the caller usually treats it as cancel.
		return nil, false

	default:
		// Forward to the underlying input below.
	}

	var cmd tea.Cmd
	p.Input, cmd = p.Input.Update(msg)
	p.Refresh()

	return cmd, true
}

// View renders the input followed by the suggestions listbox (if any).
func (p *PathAutocomplete) View() string {
	var b strings.Builder
	b.WriteString(p.Input.View())

	if len(p.suggestions) == 0 {
		return b.String()
	}

	b.WriteString("\n")

	for i, name := range p.suggestions {
		cursor := "  "
		style := p.dimStyle

		if p.active && i == p.cursor {
			cursor = "▸ "
			style = p.highlightStyle
		}

		b.WriteString(style.Render(cursor+name+"/") + "\n")
	}

	return b.String()
}

// Refresh recomputes suggestions from the current input value.
//
// The directory listed is the parent of the partial currently being typed:
// for "/home/fred/sr" it lists entries of "/home/fred" whose name starts
// with "sr"; for "/home/fred/" it lists every entry of "/home/fred".
//
// Hidden directories are skipped unless the partial itself starts with a
// dot. Common noise directories ([fs.ShouldSkipDir]) are always skipped.
func (p *PathAutocomplete) Refresh() {
	p.suggestions = nil
	p.cursor = 0

	val := p.Input.Value()

	idx := strings.LastIndex(val, "/")
	if idx < 0 {
		return
	}

	rawDir := val[:idx+1]
	partial := val[idx+1:]

	expanded, err := fs.ExpandHome(rawDir)
	if err != nil {
		return
	}

	if !filepath.IsAbs(expanded) {
		return
	}

	entries, err := os.ReadDir(expanded)
	if err != nil {
		return
	}

	matches := make([]string, 0, len(entries))
	allowHidden := strings.HasPrefix(partial, ".")

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		name := e.Name()
		if !strings.HasPrefix(name, partial) {
			continue
		}

		if !allowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		if fs.ShouldSkipDir(name) {
			continue
		}

		matches = append(matches, name)
	}

	sort.Strings(matches)

	if len(matches) > p.maxItems {
		matches = matches[:p.maxItems]
	}

	p.suggestions = matches
}

// accept replaces the current partial with the chosen directory name and
// appends a trailing slash, kicking off a new autocomplete cycle.
func (p *PathAutocomplete) accept(name string) {
	val := p.Input.Value()

	var prefix string
	if idx := strings.LastIndex(val, "/"); idx >= 0 {
		prefix = val[:idx+1]
	}

	next := prefix + name + "/"
	p.Input.SetValue(next)
	p.Input.SetCursor(len(next))
	p.active = false
	p.Refresh()
}

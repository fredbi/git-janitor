// SPDX-License-Identifier: Apache-2.0

package gadgets

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

func newTestPopup(items ...QuickActionItem) *QuickActionsPopup {
	theme := &uxtypes.Theme{}
	p := NewQuickActionsPopup(theme)
	p.Show(items, 5, 5, 0, 80)

	return &p
}

func runesKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestPopup_Show_HiddenWhenEmpty(t *testing.T) {
	theme := &uxtypes.Theme{}
	p := NewQuickActionsPopup(theme)
	p.Show(nil, 0, 0, 0, 80)

	if p.Visible {
		t.Error("expected popup to stay hidden with no items")
	}
}

func TestPopup_NavigationWraps(t *testing.T) {
	p := newTestPopup(
		QuickActionItem{Name: "a"},
		QuickActionItem{Name: "b"},
		QuickActionItem{Name: "c"},
	)

	if p.Cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", p.Cursor)
	}

	// Down x4 should wrap: 0 → 1 → 2 → 0 → 1
	for range 4 {
		_, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	if p.Cursor != 1 {
		t.Errorf("after 4×down: cursor = %d, want 1", p.Cursor)
	}

	// Up x2 from 1 → 0 → 2
	for range 2 {
		_, _ = p.Update(tea.KeyMsg{Type: tea.KeyUp})
	}

	if p.Cursor != 2 {
		t.Errorf("after 2×up: cursor = %d, want 2", p.Cursor)
	}
}

func TestPopup_EnterReturnsSelection(t *testing.T) {
	p := newTestPopup(
		QuickActionItem{Name: "first"},
		QuickActionItem{Name: "second"},
	)

	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyDown}) // cursor → 1
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if p.Visible {
		t.Error("popup should hide after Enter")
	}

	sel := p.TakeSelection()
	if sel == nil {
		t.Fatal("expected a selection, got nil")
	}

	if sel.Name != "second" {
		t.Errorf("selected = %q, want second", sel.Name)
	}

	// Selection is one-shot.
	if p.TakeSelection() != nil {
		t.Error("TakeSelection should be one-shot")
	}
}

func TestPopup_EscClosesWithoutSelection(t *testing.T) {
	p := newTestPopup(QuickActionItem{Name: "x"})
	_, _ = p.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if p.Visible {
		t.Error("popup should hide on Esc")
	}

	if p.TakeSelection() != nil {
		t.Error("Esc should not produce a selection")
	}
}

func TestPopup_ConsumesAllKeysWhileVisible(t *testing.T) {
	p := newTestPopup(QuickActionItem{Name: "x"})
	_, consumed := p.Update(runesKey("z"))
	if !consumed {
		t.Error("popup should swallow keys while visible")
	}
}

func TestPopup_HiddenDoesNotConsume(t *testing.T) {
	theme := &uxtypes.Theme{}
	p := NewQuickActionsPopup(theme)

	_, consumed := p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if consumed {
		t.Error("hidden popup should not consume keys")
	}
}

func TestPopup_View_HiddenIsEmpty(t *testing.T) {
	theme := &uxtypes.Theme{}
	p := NewQuickActionsPopup(theme)

	if got := p.View(); got != "" {
		t.Errorf("View() while hidden = %q, want empty", got)
	}
}

func TestPopup_View_RendersBorderedItems(t *testing.T) {
	p := newTestPopup(
		QuickActionItem{Name: "open-in-terminal", Description: "Open repo in a terminal"},
		QuickActionItem{Name: "open-editor", Description: ""},
	)

	out := p.View()
	if !strings.Contains(out, "open-in-terminal") {
		t.Errorf("View() missing first item:\n%s", out)
	}
	if !strings.Contains(out, "open-editor") {
		t.Errorf("View() missing second item:\n%s", out)
	}
	// Cursor on first item should print the caret.
	if !strings.Contains(out, "▸") {
		t.Errorf("View() missing cursor caret:\n%s", out)
	}
}

func TestPopup_PreferredHeight(t *testing.T) {
	p := newTestPopup(
		QuickActionItem{Name: "a"},
		QuickActionItem{Name: "b"},
		QuickActionItem{Name: "c"},
	)

	// 3 items + 2 border lines.
	if got := p.PreferredHeight(); got != 5 {
		t.Errorf("PreferredHeight = %d, want 5", got)
	}
}

func TestElide(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"toolong", 4, "too…"},
		{"x", 1, "x"},
		{"yz", 1, "…"},
		{"", 5, ""},
	}

	for _, tc := range cases {
		if got := elide(tc.in, tc.max); got != tc.want {
			t.Errorf("elide(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	if got := padRight("ab", 5); got != "ab   " {
		t.Errorf("padRight(\"ab\", 5) = %q, want %q", got, "ab   ")
	}

	// No-op when already wide enough.
	if got := padRight("hello", 3); got != "hello" {
		t.Errorf("padRight(\"hello\", 3) = %q, want %q", got, "hello")
	}
}

func TestOverlay_NoPopup(t *testing.T) {
	base := "line1\nline2\nline3"
	if got := Overlay(base, "", 0, 0, 0, 80, 80); got != base {
		t.Errorf("Overlay with empty popup changed base: %q", got)
	}
}

func TestOverlay_SplicesPopup(t *testing.T) {
	base := strings.Join([]string{
		"...........",
		"...........",
		"...........",
		"...........",
	}, "\n")
	popup := "abc\ndef"

	out := Overlay(base, popup, 2, 1, 0, 11, 11)
	lines := strings.Split(out, "\n")

	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	if lines[0] != "..........." {
		t.Errorf("line 0 changed unexpectedly: %q", lines[0])
	}

	if !strings.Contains(lines[1], "abc") {
		t.Errorf("line 1 missing popup row 0: %q", lines[1])
	}

	if !strings.Contains(lines[2], "def") {
		t.Errorf("line 2 missing popup row 1: %q", lines[2])
	}
}

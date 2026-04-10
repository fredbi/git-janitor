package gadgets

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// fixture builds a temporary directory tree used by the autocomplete tests:
//
//	<tmp>/alpha/
//	<tmp>/alpine/
//	<tmp>/beta/
//	<tmp>/.hidden/
//	<tmp>/file.txt          (regular file, must be ignored)
func fixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	for _, name := range []string{"alpha", "alpine", "beta", ".hidden"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write file.txt: %v", err)
	}

	return root
}

func newGadget(value string) PathAutocomplete {
	in := textinput.New()
	in.SetValue(value)
	in.SetCursor(len(value))
	in.Focus()

	return NewPathAutocomplete(in)
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func assertSuggestions(t *testing.T, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestions mismatch:\n  got:  %v\n  want: %v", got, want)
	}
}

func TestRefreshSuggestsMatchingDirectories(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	assertSuggestions(t, p.Suggestions(), []string{"alpha", "alpine"})
}

func TestRefreshTrailingSlashListsAllVisibleDirs(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/")

	// Hidden entries and regular files are filtered out.
	assertSuggestions(t, p.Suggestions(), []string{"alpha", "alpine", "beta"})
}

func TestRefreshDotPrefixIncludesHidden(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/.")

	assertSuggestions(t, p.Suggestions(), []string{".hidden"})
}

func TestRefreshNoSlashYieldsNoSuggestions(t *testing.T) {
	p := newGadget("notapath")

	if got := p.Suggestions(); len(got) != 0 {
		t.Fatalf("expected no suggestions, got %v", got)
	}
}

func TestRefreshMissingDirectoryYieldsNoSuggestions(t *testing.T) {
	p := newGadget("/nonexistent-path-xyz/")

	if got := p.Suggestions(); len(got) != 0 {
		t.Fatalf("expected no suggestions, got %v", got)
	}
}

func TestUpdateDownActivatesAndMovesCursor(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	cmd, consumed := p.Update(keyMsg("down"))
	if !consumed {
		t.Fatal("expected Down to be consumed")
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
	}
	if got := p.HighlightedIndex(); got != 1 {
		t.Fatalf("expected cursor=1 after first Down, got %d", got)
	}

	// Wraps to the top.
	_, _ = p.Update(keyMsg("down"))
	if got := p.HighlightedIndex(); got != 0 {
		t.Fatalf("expected cursor wrap to 0, got %d", got)
	}
}

func TestUpdateUpWrapsAndActivates(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	_, consumed := p.Update(keyMsg("up"))
	if !consumed {
		t.Fatal("expected Up to be consumed")
	}
	if got := p.HighlightedIndex(); got != 1 {
		t.Fatalf("expected cursor wrap to 1, got %d", got)
	}
}

func TestUpdateEnterWithoutSelectionIsNotConsumed(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	cmd, consumed := p.Update(keyMsg("enter"))
	if consumed {
		t.Fatal("Enter must fall through to the caller when no suggestion is highlighted")
	}
	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
	}
}

func TestUpdateEnterAcceptsHighlightedSuggestion(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	// Highlight "alpine" (second entry — first Down activates and moves to index 1).
	_, _ = p.Update(keyMsg("down"))

	_, consumed := p.Update(keyMsg("enter"))
	if !consumed {
		t.Fatal("expected Enter to be consumed when accepting a suggestion")
	}

	want := root + "/alpine/"
	if got := p.Value(); got != want {
		t.Fatalf("value mismatch: got %q want %q", got, want)
	}
	if got := p.HighlightedIndex(); got != -1 {
		t.Fatalf("selection should reset after accept, got %d", got)
	}
	// New autocomplete cycle: alpine has no children → no suggestions.
	if got := p.Suggestions(); len(got) != 0 {
		t.Fatalf("expected empty suggestions after drilling into empty dir, got %v", got)
	}
}

func TestUpdateTabAcceptsFirstSuggestionWithoutNavigation(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	_, consumed := p.Update(keyMsg("tab"))
	if !consumed {
		t.Fatal("expected Tab to be consumed")
	}
	if got, want := p.Value(), root+"/alpha/"; got != want {
		t.Fatalf("value mismatch: got %q want %q", got, want)
	}
}

func TestUpdateEscIsNotConsumed(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	if _, consumed := p.Update(keyMsg("esc")); consumed {
		t.Fatal("Esc must always fall through to the caller")
	}
}

func TestUpdateTypingRefreshesSuggestions(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/")

	assertSuggestions(t, p.Suggestions(), []string{"alpha", "alpine", "beta"})

	if _, consumed := p.Update(keyMsg("a")); !consumed {
		t.Fatal("typing should be consumed")
	}
	_, _ = p.Update(keyMsg("l"))

	assertSuggestions(t, p.Suggestions(), []string{"alpha", "alpine"})
}

func TestSetValueResetsCycle(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	_, _ = p.Update(keyMsg("down"))
	if p.HighlightedIndex() == -1 {
		t.Fatal("precondition: cursor should be active")
	}

	p.SetValue(root + "/be")
	if got := p.HighlightedIndex(); got != -1 {
		t.Fatalf("SetValue should reset highlight, got %d", got)
	}
	assertSuggestions(t, p.Suggestions(), []string{"beta"})
}

func TestMaxItemsCapsSuggestions(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"d1", "d2", "d3", "d4", "d5", "d6", "d7", "d8"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}

	p := newGadget(root + "/")
	p.SetMaxItems(3)
	p.Refresh()

	assertSuggestions(t, p.Suggestions(), []string{"d1", "d2", "d3"})
}

func TestViewIncludesSuggestionsListbox(t *testing.T) {
	root := fixture(t)
	p := newGadget(root + "/al")

	out := p.View()
	for _, want := range []string{"alpha", "alpine"} {
		if !strings.Contains(out, want) {
			t.Errorf("View output missing %q:\n%s", want, out)
		}
	}
}

func TestViewWithoutSuggestionsRendersInputOnly(t *testing.T) {
	p := newGadget("nopath")

	out := p.View()
	if strings.Contains(out, "▸") {
		t.Errorf("expected no highlight glyph in empty-suggestion view, got:\n%s", out)
	}
}

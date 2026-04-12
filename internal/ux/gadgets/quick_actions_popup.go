// SPDX-License-Identifier: Apache-2.0

package gadgets

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/fredbi/git-janitor/internal/ux/key"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// quickActionDescWidth caps the description column width so the popup
// stays compact even with verbose descriptions.
const quickActionDescWidth = 40

// QuickActionItem is one entry in the [QuickActionsPopup] list. Callers
// build it from a registered quick action; the popup itself is unaware of
// the engine type so it can be unit-tested without a registry.
type QuickActionItem struct {
	// Name is the display name (e.g. "open-in-terminal").
	Name string

	// Description is the short hint shown after the name.
	Description string
}

// QuickActionsPopup is a small list overlay anchored to the focused panel.
//
// Visually it mimics the auto-complete dropdown used by the config wizard:
// no big centered border, just a compact list with a cursor caret. The
// caller positions it (anchorX/anchorY) to appear immediately under the
// currently selected row of the focused list, flipping above when there is
// not enough room below.
type QuickActionsPopup struct {
	Theme   *uxtypes.Theme
	Visible bool
	Items   []QuickActionItem
	Cursor  int

	// AnchorX, AnchorY are absolute screen coordinates of the popup's
	// top-left corner.
	AnchorX int
	AnchorY int

	// PanelX is the x of the focused panel (used to keep the popup inside
	// the panel even when the items are wider than the anchor row).
	PanelX     int
	PanelWidth int

	// selected is set by Update on Enter and consumed via TakeSelection().
	selected *QuickActionItem
}

// NewQuickActionsPopup constructs an empty (hidden) popup.
func NewQuickActionsPopup(theme *uxtypes.Theme) QuickActionsPopup {
	return QuickActionsPopup{Theme: theme}
}

// Show populates the popup with items and makes it visible.
//
// anchorX and anchorY are the absolute screen coordinates where the popup
// should appear; the caller is responsible for computing them from the
// focused panel and clamping vertically (above-the-cursor placement when
// there is not enough room below). panelX and panelWidth describe the
// horizontal bounds of the focused panel so the popup can be kept inside.
func (p *QuickActionsPopup) Show(items []QuickActionItem, anchorX, anchorY, panelX, panelWidth int) {
	p.Items = items
	p.Cursor = 0
	p.Visible = len(items) > 0
	p.AnchorX = anchorX
	p.AnchorY = anchorY
	p.PanelX = panelX
	p.PanelWidth = panelWidth
	p.selected = nil
}

// Hide closes the popup and discards any pending selection.
func (p *QuickActionsPopup) Hide() {
	p.Visible = false
	p.selected = nil
}

// TakeSelection returns the item the user picked with Enter and clears it.
// Callers should invoke this after calling Update; a non-nil result means
// the popup has been hidden and the action should be executed.
func (p *QuickActionsPopup) TakeSelection() *QuickActionItem {
	sel := p.selected
	p.selected = nil

	return sel
}

// Update handles key messages while the popup is visible. The boolean
// return reports whether the popup consumed the message (always true while
// visible — callers should not forward keys onward).
func (p *QuickActionsPopup) Update(msg tea.Msg) (tea.Cmd, bool) {
	if !p.Visible {
		return nil, false
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, true
	}

	switch kb := key.MsgBinding(keyMsg); {
	case kb == key.Up || kb == key.K:
		if len(p.Items) == 0 {
			return nil, true
		}

		if p.Cursor > 0 {
			p.Cursor--
		} else {
			p.Cursor = len(p.Items) - 1
		}

		return nil, true

	case kb == key.Down || kb == key.J:
		if len(p.Items) == 0 {
			return nil, true
		}

		if p.Cursor < len(p.Items)-1 {
			p.Cursor++
		} else {
			p.Cursor = 0
		}

		return nil, true

	case kb == key.Enter:
		if p.Cursor >= 0 && p.Cursor < len(p.Items) {
			item := p.Items[p.Cursor]
			p.selected = &item
		}

		p.Visible = false

		return nil, true

	case kb.ClosePopup():
		p.Hide()

		return nil, true
	}

	// Swallow any other key — the popup owns input while visible.
	return nil, true
}

// PreferredHeight returns the number of lines the popup occupies when
// rendered, including its border. Useful when computing whether to anchor
// above or below the cursor.
func (p *QuickActionsPopup) PreferredHeight() int {
	const borderRows = 2

	return len(p.Items) + borderRows
}

// PreferredWidth returns the rendered width of the popup, including border.
func (p *QuickActionsPopup) PreferredWidth() int {
	const (
		borderCols = 2
		caretCols  = 2 // "▸ " or "  "
		gapCols    = 2 // separator between name and description
	)

	maxName := 0
	maxDesc := 0

	for _, it := range p.Items {
		if w := lipgloss.Width(it.Name); w > maxName {
			maxName = w
		}

		dw := min(lipgloss.Width(it.Description), quickActionDescWidth)

		if dw > maxDesc {
			maxDesc = dw
		}
	}

	width := borderCols + caretCols + maxName
	if maxDesc > 0 {
		width += gapCols + maxDesc
	}

	return width
}

// View renders the popup as a small bordered list, ready to be overlaid on
// top of the base view via [Overlay]. Returns an empty string when hidden.
//
// The caller is expected to compose the return value into the final frame
// (typically by splicing it into the rendered lines at AnchorY/AnchorX).
func (p *QuickActionsPopup) View() string {
	if !p.Visible || len(p.Items) == 0 {
		return ""
	}

	t := p.Theme

	caretStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Accent)
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Bright)
	descStyle := lipgloss.NewStyle().Foreground(t.Dim)
	dimNameStyle := lipgloss.NewStyle().Foreground(t.Bright)

	// Compute column widths so all rows align.
	maxName := 0

	for _, it := range p.Items {
		if w := lipgloss.Width(it.Name); w > maxName {
			maxName = w
		}
	}

	var b strings.Builder

	for i, it := range p.Items {
		caret := "  "
		ns := dimNameStyle

		if i == p.Cursor {
			caret = "▸ "
			ns = nameStyle
		}

		name := ns.Render(padRight(it.Name, maxName))
		desc := descStyle.Render(elide(it.Description, quickActionDescWidth))

		row := caretStyle.Render(caret) + name
		if desc != "" {
			row += "  " + desc
		}

		b.WriteString(row)

		if i < len(p.Items)-1 {
			b.WriteString("\n")
		}
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Tertiary).
		Padding(0, 1)

	return border.Render(b.String())
}

// padRight pads s with spaces on the right to reach width w (visual cells).
func padRight(s string, w int) string {
	pad := w - lipgloss.Width(s)
	if pad <= 0 {
		return s
	}

	return s + strings.Repeat(" ", pad)
}

// elide truncates s to at most max visual cells, appending an ellipsis when
// truncation occurred.
func elide(s string, maxWidth int) string {
	if maxWidth <= 0 || lipgloss.Width(s) <= maxWidth {
		return s
	}

	if maxWidth <= 1 {
		return "…"
	}

	// Naive byte-truncation; descriptions are ASCII in practice.
	if len(s) > maxWidth-1 {
		s = s[:maxWidth-1]
	}

	return s + "…"
}

// Overlay splices the popup's rendered View into base at the popup's
// (AnchorX, AnchorY), keeping it inside the focused panel's horizontal
// bounds. Lines that extend past the right edge of base are truncated.
//
// This is a separate function (rather than a method on the popup) so the
// model can call it from its top-level View() with full knowledge of the
// frame dimensions.
func Overlay(base, popup string, anchorX, anchorY, panelX, panelWidth, frameWidth int) string {
	if popup == "" {
		return base
	}

	popupLines := strings.Split(popup, "\n")
	baseLines := strings.Split(base, "\n")

	// Clamp the popup's left edge so it stays inside the panel.
	x := anchorX
	popupW := 0

	for _, pl := range popupLines {
		if w := lipgloss.Width(pl); w > popupW {
			popupW = w
		}
	}

	rightEdge := panelX + panelWidth - 1
	if x+popupW-1 > rightEdge {
		x = rightEdge - popupW + 1
	}

	if x < panelX {
		x = panelX
	}

	for i, pl := range popupLines {
		row := anchorY + i
		if row < 0 || row >= len(baseLines) {
			continue
		}

		baseLines[row] = spliceLine(baseLines[row], pl, x, frameWidth)
	}

	return strings.Join(baseLines, "\n")
}

// spliceLine inserts overlay into base at the given visual column,
// truncating to maxWidth. Both base and overlay may contain ANSI escapes,
// so we work with the lipgloss visual width and use lipgloss helpers for
// padding rather than naive slicing.
func spliceLine(base, overlay string, col, maxWidth int) string {
	// Pad base with spaces so the splice column is within range.
	if baseW := lipgloss.Width(base); baseW < col {
		base += strings.Repeat(" ", col-baseW)
	}

	// Take the visible prefix [0, col) of base.
	left := truncateVisual(base, col)
	overlayW := lipgloss.Width(overlay)
	right := dropVisual(base, col+overlayW)

	combined := left + overlay + right
	if maxWidth > 0 && lipgloss.Width(combined) > maxWidth {
		combined = truncateVisual(combined, maxWidth)
	}

	return combined
}

// truncateVisual returns the visual prefix of s with at most n cells.
//
// It steps over the rune sequence accumulating widths via lipgloss to stay
// ANSI-aware enough for our themed strings; non-ASCII inside an escape
// sequence is uncommon for this codebase.
func truncateVisual(s string, n int) string {
	if n <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= n {
		return s
	}

	// Walk runes, dropping anything past the visual budget. We approximate
	// by counting runes outside ANSI escape sequences.
	var (
		out      strings.Builder
		inEscape bool
		visible  int
	)

	for _, r := range s {
		if inEscape {
			out.WriteRune(r)

			if r == 'm' {
				inEscape = false
			}

			continue
		}

		if r == '\x1b' {
			inEscape = true
			out.WriteRune(r)

			continue
		}

		if visible >= n {
			break
		}

		visible++
		out.WriteRune(r)
	}

	return out.String()
}

// dropVisual returns s with the first n visible cells removed.
func dropVisual(s string, n int) string {
	if n <= 0 {
		return s
	}

	if lipgloss.Width(s) <= n {
		return ""
	}

	var (
		out      strings.Builder
		inEscape bool
		visible  int
	)

	for _, r := range s {
		if inEscape {
			out.WriteRune(r)

			if r == 'm' {
				inEscape = false
			}

			continue
		}

		if r == '\x1b' {
			inEscape = true
			out.WriteRune(r)

			continue
		}

		if visible < n {
			visible++

			continue
		}

		out.WriteRune(r)
	}

	return out.String()
}

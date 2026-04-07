// SPDX-License-Identifier: Apache-2.0

// Package panels provides shared utilities for TUI sub-panels.
package panels

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/ux/key"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Base holds the common state for scrollable, cursor-driven panels.
//
// Embed this in concrete panel structs to get sizing, cursor navigation,
// scroll clamping, and row padding for free.
type Base struct {
	Theme  *uxtypes.Theme
	Cursor int
	Offset int
	Width  int
	Height int
}

// SetSize stores the available width and height.
// reservedLines is subtracted from h (e.g. 1 for a header line).
// minHeight is the floor (e.g. 1, or cardLines for multi-line items).
func (b *Base) SetSize(w, h, reservedLines, minHeight int) {
	b.Width = w
	b.Height = max(h-reservedLines, minHeight)
}

// ResetScroll resets the cursor and scroll offset to the top.
func (b *Base) ResetScroll() {
	b.Cursor = 0
	b.Offset = 0
}

// ClampScroll adjusts the scroll offset so that the cursor remains visible.
// visibleItems is the number of items that fit on screen (e.g. Height for
// single-line items, Height/cardLines for multi-line cards).
func (b *Base) ClampScroll(visibleItems int) {
	if visibleItems < 1 {
		visibleItems = 1
	}

	if b.Cursor < b.Offset {
		b.Offset = b.Cursor
	}

	if b.Cursor >= b.Offset+visibleItems {
		b.Offset = b.Cursor - visibleItems + 1
	}
}

// NavigateKey handles standard cursor movement (j/k, arrows, home/end, page up/down).
// It returns true if the key was consumed. itemCount is the total
// number of navigable items.
func (b *Base) NavigateKey(msg tea.KeyMsg, itemCount int) bool {
	switch key.MsgBinding(msg) {
	case key.Up, key.K:
		if b.Cursor > 0 {
			b.Cursor--
		}

		return true

	case key.Down, key.J:
		if b.Cursor < itemCount-1 {
			b.Cursor++
		}

		return true

	case key.PageUp:
		b.Cursor = max(0, b.Cursor-b.Height)

		return true

	case key.PageDown:
		b.Cursor = min(itemCount-1, b.Cursor+b.Height)

		return true

	case key.Home, key.G:
		b.Cursor = 0

		return true

	case key.End, key.GG:
		b.Cursor = max(0, itemCount-1)

		return true

	default:
		return false
	}
}

// VisibleRange returns the start (inclusive) and end (exclusive) indices
// of the currently visible items, clamped to [0, itemCount).
func (b *Base) VisibleRange(itemCount, visibleItems int) (start, end int) {
	start = b.Offset
	end = min(b.Offset+visibleItems, itemCount)

	return start, end
}

// PadRows appends empty strings until rows has at least targetLines entries.
func PadRows(rows []string, targetLines int) []string {
	for len(rows) < targetLines {
		rows = append(rows, "")
	}

	return rows
}

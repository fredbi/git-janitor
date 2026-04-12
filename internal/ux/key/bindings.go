// SPDX-License-Identifier: Apache-2.0

// Package key centralizes key binding definitions for the TUI.
package key

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Binding enumerates all supported key bindings.
type Binding string

// MsgBinding extracts a Binding from a bubbletea key message.
func MsgBinding(msg tea.KeyMsg) Binding {
	return Binding(strings.ToLower(msg.String()))
}

const (
	// Ctrl combinations.
	CtrlC Binding = "ctrl+c"
	CtrlQ Binding = "ctrl+q"
	CtrlH Binding = "ctrl+h"
	CtrlR Binding = "ctrl+r"
	CtrlA Binding = "ctrl+a"
	CtrlP Binding = "ctrl+p"
	CtrlD Binding = "ctrl+d"
	CtrlK Binding = "ctrl+k"

	// Special keys.
	Esc   Binding = "esc"
	Enter Binding = "enter"
	Tab   Binding = "tab"
	Space Binding = " "

	ShiftTab Binding = "shift+tab"

	// Navigation.
	Up       Binding = "up"
	Down     Binding = "down"
	Home     Binding = "home"
	End      Binding = "end"
	PageUp   Binding = "pgup"
	PageDown Binding = "pgdown"

	// Arrows.
	RightArrow Binding = "right"
	LeftArrow  Binding = "left"

	// Letters.
	Slash Binding = "/"
	Q     Binding = "q"
	L     Binding = "l"
	H     Binding = "h"
	Y     Binding = "y"
	N     Binding = "n"
	A     Binding = "a"
	C     Binding = "c"
	D     Binding = "d"
	G     Binding = "g"
	GG    Binding = "G" // shift+g (end)
	J     Binding = "j"
	K     Binding = "k"
)

// Quit reports whether this binding is a quit action.
func (b Binding) Quit() bool {
	return b == CtrlC || b == CtrlQ
}

// Confirm reports whether this binding is a confirmation.
func (b Binding) Confirm() bool {
	return b == Y
}

// Cancel reports whether this binding is a cancellation.
func (b Binding) Cancel() bool {
	return b == N || b == Esc
}

// ClosePopup reports whether this binding closes a modal/popup.
func (b Binding) ClosePopup() bool {
	return b == Esc || b == Q
}

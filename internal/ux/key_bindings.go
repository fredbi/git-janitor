package ux

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TODO: move as a separate package and rename "Key" to avoid stuttering.

// KeyBinding enumerates all supported key bindings.
type KeyBinding string

func KeyMsgBinding(msg tea.KeyMsg) KeyBinding {
	return KeyBinding(strings.ToLower(msg.String()))
}

const (
	CtrlC KeyBinding = "ctrl+c"
	CtrlQ KeyBinding = "ctrl+q"
	CtrlH KeyBinding = "ctrl+h"
	CtrlR KeyBinding = "ctrl+r"
	CtrlA KeyBinding = "ctrl+a"
	Esc   KeyBinding = "esc"
	Enter KeyBinding = "enter"

	Tab      KeyBinding = "tab"
	ShiftTab KeyBinding = "shift+tab"

	Slash KeyBinding = "/"
	KeyQ  KeyBinding = "q"
	KeyL  KeyBinding = "l"
	KeyH  KeyBinding = "h"
	KeyY  KeyBinding = "y"
	KeyN  KeyBinding = "n"

	RightArrow KeyBinding = "right"
	LeftArrow  KeyBinding = "left"
)

func (b KeyBinding) Quit() bool {
	return b == CtrlC || b == CtrlQ
}

func (b KeyBinding) Confirm() bool {
	return b == KeyY
}

func (b KeyBinding) Cancel() bool {
	return b == KeyN || b == Esc
}

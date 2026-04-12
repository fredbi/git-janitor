# Adding UX features to git-janitor

Step-by-step guide for adding new gadgets (popups, overlays), panel extensions, and key bindings to the TUI.

## When to use

When you need to add new interactive UI components: popups, list overlays, new panel tabs, new key bindings, or new commands.

## Architecture overview

```
Model (ux/ux-model.go)
  ├── Repos panel     (ux/panels/repos/)        — left: tabbed repo list
  ├── Right panel     (ux/panels/infos/)         — right: 7 tabs (Facts, Branches, ...)
  │   └── tab-*/      (ux/panels/infos/tab-*/)   — individual tab panels
  ├── Input           (ux/commands/)             — command input bar
  ├── Status          (ux/statusbar/)            — bottom status bar
  ├── Help popup      (ux/commands/help/)        — centered overlay
  ├── Detail popup    (ux/gadgets/)              — centered overlay
  ├── QuickActions    (ux/gadgets/)              — anchored overlay (spliced into frame)
  └── Wizard          (ux/commands/config-wizard/) — modal overlay
```

**Key principle:** The top-level `Model` owns all sub-components as struct fields. Messages flow through `Update()`, rendering through `View()`. Sub-components never import `Model` — communication is via bubbletea messages (`ux/types/`).

## Existing patterns to follow

### 1. Centered overlay (DetailPopup pattern)

Used for: help, detail views, modals. Replaces the entire frame.

**File:** `ux/gadgets/detail_popup.go`

```go
type DetailPopup struct {
    Theme    *uxtypes.Theme
    Viewport viewport.Model
    Visible  bool
    Title    string
    Content  string
    Width, Height int
}
```

**Lifecycle:**
- `Show(title, content)` — set content, make visible, reset scroll
- `Hide()` — set invisible
- `SetSize(termWidth, termHeight)` — called from `Model.recalcLayout()`
- `Update(msg)` — handle scroll keys (viewport)
- `View(termWidth, termHeight)` — render centered via `lipgloss.Place()`

**Model integration:**
- Field on `Model` struct
- Created in `New()`
- `SetSize()` called in `recalcLayout()`
- Visibility checked in `View()` — returns overlay instead of base when visible
- Key dispatch: when `Visible`, route keys to popup before anything else

### 2. Anchored overlay (QuickActionsPopup pattern)

Used for: context menus, dropdowns anchored to a cursor position.

**File:** `ux/gadgets/quick_actions_popup.go`

```go
type QuickActionsPopup struct {
    Theme   *uxtypes.Theme
    Visible bool
    Items   []QuickActionItem
    Cursor  int
    AnchorX, AnchorY int
    PanelX, PanelWidth int
}
```

**Key difference from centered:** rendered by `Overlay()` function which splices the popup into the base frame at (AnchorX, AnchorY) — ANSI-aware line splicing. The base frame stays visible around the popup.

**Model integration in View():**
```go
if m.QuickActions.Visible {
    base = gadgets.Overlay(base, m.QuickActions.View(), ...)
}
```

### 3. Inline autocomplete (PathAutocomplete pattern)

Used for: text input with a dropdown suggestion list.

**File:** `ux/gadgets/path_autocomplete.go`

Wraps a `textinput.Model`, renders suggestions below it with `▸` cursor. `Update()` returns `(cmd, consumed bool)` — caller checks `consumed` to know if the gadget handled the key.

### 4. Panel tab (Base-embedded)

Used for: adding a new tab to the right panel.

**File pattern:** `ux/panels/infos/tab-*/`

```go
type Panel struct {
    panels.Base                    // embedded: Cursor, Offset, Width, Height, NavigateKey()
    items []models.SomeType        // data
}

func (p *Panel) SetInfo(info *models.RepoInfo) { /* populate items, ResetScroll() */ }
func (p *Panel) SetSize(w, h int)              { p.Base.SetSize(w, h, 1, 1) } // 1 header line
func (p *Panel) Update(msg tea.Msg) tea.Cmd    { /* NavigateKey() + Enter handling */ }
func (p *Panel) View() string                  { /* header + VisibleRange loop */ }
```

**Registration:** Add to `infos.Panel` struct, `New()`, `SetTheme()`, `SetRepoInfo()`, tab definitions in `RightTabDefs`, increment `RightTabCount`.

## Step-by-step: Adding a new popup gadget

### 1. Create the gadget

**File:** `ux/gadgets/my_popup.go`

Implement at minimum: `Show()`, `Hide()`, `Update(msg) (tea.Cmd, bool)`, `View() string`.

Follow the popup pattern:
- `Visible bool` field controls rendering
- `Update()` returns `consumed bool` so the model knows to swallow keys
- `View()` returns empty string when hidden

### 2. Add to Model

**File:** `ux/ux-model.go`

```go
type Model struct {
    // ...existing fields...
    MyPopup gadgets.MyPopup    // add field
}
```

Initialize in `New()`, propagate theme in `setTheme()`.

### 3. Add key binding

**File:** `ux/key/bindings.go`

```go
const CtrlX Binding = "ctrl+x"
```

### 4. Wire key dispatch

**File:** `ux/ux-model.go` → `handleKey()`

**Priority order for key dispatch:**
1. Quick actions popup (when visible)
2. Right panel text input capture (when active)
3. Global keys (quit, help, refresh, new binding here)
4. Help popup (when visible) — captures all keys
5. Detail popup (when visible) — captures all keys
6. Tab/navigation keys
7. Panel-level shortcuts
8. Forward to focused panel

When your popup is visible, intercept early (before global keys except quit):
```go
if m.MyPopup.Visible {
    if kb.Quit() { ... }
    return m.handleMyPopupKey(msg)
}
```

### 5. Wire rendering

**File:** `ux/ux-model.go` → `View()`

For centered overlays (replace frame):
```go
if m.MyPopup.Visible {
    return m.MyPopup.View(m.Width, m.Height)
}
```

For anchored overlays (splice into frame):
```go
if m.MyPopup.Visible {
    base = gadgets.Overlay(base, m.MyPopup.View(), anchorX, anchorY, ...)
}
```

### 6. Update help text

**File:** `ux/commands/help/help.go`

Update both the general `helpText` constant and relevant `contextHelp` entries.

### 7. Size management

If your gadget needs sizing, add to `recalcLayout()`:
```go
m.MyPopup.SetSize(m.Width, m.Height)
```

## Step-by-step: Adding a new key binding

1. Add constant to `ux/key/bindings.go`
2. Handle in `handleKey()` at the appropriate priority level
3. Add helper method(s) (e.g. `Quit()`, `ClosePopup()`) if the binding has semantic meaning
4. Update `help.go` — both general and contextual help

## Message flow

Custom messages go in `ux/types/types.go`. The model's `Update()` dispatches them:

```go
case uxtypes.MyCustomMsg:
    return m.handleMyCustom(msg)
```

For async operations, return a `tea.Cmd` (closure returning `tea.Msg`):
```go
return func() tea.Msg {
    result := doExpensiveThing()
    return uxtypes.MyCustomMsg{Result: result}
}
```

## Funcorder convention

The linter enforces method ordering: **exported methods before unexported methods**. Place new private helpers after `View()`.

## Build and test

```sh
go build ./...
go test ./internal/ux/gadgets/...
golangci-lint run --new-from-rev master
```

Tests for gadgets should verify: navigation wraps, Enter produces selection, Esc closes without selection, hidden popup doesn't consume keys, View() renders items.

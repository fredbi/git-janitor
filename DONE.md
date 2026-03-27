# Done

## Session — 2026-03-27

### Git package stabilization

- Fixed `parseChangedEntry` in `internal/git/status.go` for rename/copy entries:
  replaced `strings.Fields` with `strings.SplitN` to preserve tab-separated `path\torigPath`.
- Removed unused `names` slice in `branch_test.go` (staticcheck SA4010).
- Removed unused `tabZone` type in `right_panel.go`.
- All 26 git tests pass, including integration tests against the live repo.

### Root Name field

- `LocalRoot` now has a `Name` field (already present in the struct).
- `AddRoot(name, path, interval)` accepts a name; defaults to `filepath.Base(path)` when empty.
- New methods: `UpdateRootName(index, name)`, `RootDisplayName(index)` (falls back to basename).
- 4 new config tests: `TestAddRoot_DefaultName`, `TestRootDisplayName`, `TestUpdateRootName`, `TestRoundTrip_WithName`.

### Config wizard evolution

- New wizard steps: `wizardStepEditName`, `wizardStepEditInterval`, `wizardStepName`.
- Editing a root now shows a field picker (Name / Interval) with cursor navigation.
- Add-new flow: Path -> Name (pre-filled with basename) -> Interval -> Confirm.
- Confirm step displays the Name in the review.
- Root list view shows `Name  Path  (every interval)` per entry.

### Tabbed left panel

- Rewrote `repos_panel.go` from a single flat list into a tabbed panel:
  one tab per configured root, tab title = root display name.
- Each tab holds its own `bubbles/list` of repositories.
- Tab bar with elision: when tabs exceed available width, hidden tabs are
  replaced with `"..."`. The visible window shifts as the user navigates.
- Navigation: `Ctrl+A` cycles tabs, left/right arrows cycle tabs,
  mouse click on tab labels (including `"..."`) switches tabs.
- `SetRootItems(rootIndex, items)` populates repos for a specific root.
- `rebuildTabs(cfg)` recreates tabs when config changes after the wizard.

### Scanner grouped by root

- `scanResultMsg` now carries `reposByRoot map[int][]repoItem` instead of a flat slice.
- Each root's repos are populated into the corresponding tab after scan.

### Model wiring updates

- `Ctrl+A` and arrow keys now cycle tabs in whichever panel is focused (repos or right).
- Mouse clicks on the left panel's tab bar row switch root tabs.
- `handleScanResult` populates repos per-root via `SetRootItems`.
- `handleWizardDone` rebuilds tabs and re-applies sizing after config changes.

### Help popup

- Updated documentation for: tabbed repos panel, `Ctrl+A` behavior (now panel-specific),
  root tab navigation, wizard name editing flow.

### Final state

- Clean build, 34 tests passing (8 config + 26 git), 0 lint issues.

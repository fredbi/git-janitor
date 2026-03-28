// Package types defines shared data types for the ux sub-packages.
package types

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/git"
)

// Theme defines the color palette for the entire UI.
type Theme struct {
	Name string

	// Primary accent (left panel, wizard highlights).
	Accent lipgloss.Color
	// Secondary accent (right panel).
	Secondary lipgloss.Color
	// Tertiary accent (input focus, actions).
	Tertiary lipgloss.Color

	// Text colors.
	Text       lipgloss.Color // normal text
	Dim        lipgloss.Color // muted/hint text
	Bright     lipgloss.Color // bright foreground (e.g. selected items)
	HeaderText lipgloss.Color // column headers

	// Status colors.
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color

	// Non-git highlight.
	NotGit lipgloss.Color

	// Status bar.
	StatusFg lipgloss.Color
	StatusBg lipgloss.Color

	// Selected item background.
	SelectedBg lipgloss.Color

	// Recent panel accent.
	RecentAccent lipgloss.Color
	// Actions panel accent.
	ActionsAccent lipgloss.Color
}

// CurrentTheme is the active theme, accessed by all panels for rendering.
var CurrentTheme *Theme //nolint:gochecknoglobals

// RepoInfoMsg is sent when background git inspection completes.
type RepoInfoMsg struct {
	Info git.RepoInfo
}

// RepoRefreshMsg is sent after a fetch + re-inspect completes.
type RepoRefreshMsg struct {
	Info git.RepoInfo
}

// RepoItem represents a repository entry in the list.
type RepoItem struct {
	Path  string
	Name  string
	IsGit bool // true if a .git directory was found
}

// Title implements the list.DefaultItem interface.
func (i RepoItem) Title() string {
	if !i.IsGit {
		return i.Name + " (not git)"
	}

	return i.Name
}

// Description implements the list.DefaultItem interface.
func (i RepoItem) Description() string { return i.Path }

// FilterValue implements the list.Item interface.
func (i RepoItem) FilterValue() string { return i.Name }

// ScanResultMsg is sent when a background scan completes.
type ScanResultMsg struct {
	// ReposByRoot maps root index → discovered repos for that root.
	ReposByRoot map[int][]RepoItem
	Err         error
}

// ConfigWizardMsg is sent by the wizard when it finishes successfully.
type ConfigWizardMsg struct {
	Cfg *config.Config
}

// CommandResult is a tea.Msg sent after a command is executed.
type CommandResult struct {
	Output string
}

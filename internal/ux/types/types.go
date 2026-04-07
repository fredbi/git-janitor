// Package types defines shared data types for the ux sub-packages.
package types

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/models"
)

// Theme defines the color palette for the entire UI.
type Theme struct {
	ThemeName string

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

func (th Theme) Name() string {
	return th.ThemeName
}

// RepoInfoMsg is sent when background git inspection completes.
//
// It consolidates repo information from different sources (e.g. git, github).
type RepoInfoMsg struct {
	Info *models.RepoInfo
}

// RepoRefreshMsg is sent after a fetch + re-inspect completes.
type RepoRefreshMsg struct {
	Info *models.RepoInfo
}

// ScanResultMsg is sent when a background scan completes.
type ScanResultMsg struct {
	// ReposByRoot maps root index → discovered repos for that root.
	ReposByRoot map[int][]models.RepoItem
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

// ExecuteActionMsg is sent when the user selects an action to execute.
type ExecuteActionMsg struct {
	RepoPath   string
	ActionName string
	Subjects   []models.ActionSubject
}

func (m ExecuteActionMsg) SubjectLabels() []string {
	names := make([]string, 0, len(m.Subjects))
	for _, subject := range m.Subjects {
		names = append(names, subject.Subject)
	}

	return names
}

// ActionResultMsg is sent when an action execution completes.
type ActionResultMsg struct {
	RepoPath   string
	ActionName string
	OK         bool
	Message    string
}

// ShowSuggestionsMsg is sent when the user presses Enter on an alert
// to show the suggested actions for that alert.
type ShowSuggestionsMsg struct {
	AlertIndex int
}

// CopyToClipboardMsg is sent when the user wants to copy text to the clipboard.
type CopyToClipboardMsg struct {
	Text string
}

// GitHubInfoMsg is sent when background GitHub API fetch completes.
type GitHubInfoMsg struct {
	RepoPath string
	Data     *models.RepoInfo
}

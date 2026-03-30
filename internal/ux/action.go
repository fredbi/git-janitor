package ux

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/models"
	"github.com/fredbi/git-janitor/internal/ux/gadgets"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// handleExecuteAction runs an action in the background.
// If the action requires confirmation (auto=false in config, or destructive),
// the status bar shows a Y/N prompt and the action is held in PendingAction.
func (m *Model) handleExecuteAction(msg uxtypes.ExecuteActionMsg) (tea.Model, tea.Cmd) {
	action, ok := m.Engine.GetAction(msg.ActionName)
	if !ok {
		m.Status.SetMessagef("Unknown action: %s", msg.ActionName)

		return m, nil
	}

	// Check if confirmation is needed.
	needsConfirm := action.Destructive() || !m.Cfg.IsActionAuto(msg.ActionName)
	if needsConfirm {
		m.PendingAction = &msg

		label := "Run"
		if action.Destructive() {
			label = gadgets.DestructiveWarning()
		}

		subjects := gadgets.ElideLongLabel(strings.Join(msg.SubjectLabels(), ", "))
		m.Status.SetMessagef("%s action %q on %s?  [Y]es / [N]o", label, msg.ActionName, subjects)

		return m, nil
	}

	return m, m.runAction(msg)
}

// runAction executes an action in a background tea.Cmd.
func (m *Model) runAction(msg uxtypes.ExecuteActionMsg) tea.Cmd {
	m.Status.SetMessagef("Running %s...", msg.ActionName)

	repoPath := msg.RepoPath
	actionName := msg.ActionName
	subjects := msg.Subjects
	info := m.LastRepoInfo
	action := models.ActionSuggestion{
		ActionName: actionName,
		Subjects:   subjects,
	}

	return func() tea.Msg {
		ctx := m.Engine.WithRepoInfo(m.useGitWithPath(repoPath), repoPath, info)
		result, err := m.Engine.Execute(ctx, action)
		if err != nil {
			return uxtypes.ActionResultMsg{
				RepoPath:   repoPath,
				ActionName: actionName,
				OK:         false,
				Message:    err.Error(),
			}
		}

		return uxtypes.ActionResultMsg{
			RepoPath:   repoPath,
			ActionName: actionName,
			OK:         result.OK,
			Message:    result.Message,
		}
	}
}

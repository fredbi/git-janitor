// Package scan provides the /scan command that discovers
// git repositories under configured root directories.
package scan

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/fs"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Roots walks all configured roots and returns discovered git repositories,
// grouped by root index.
//
// This runs as a tea.Cmd so it doesn't block the UI.
func Roots(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if cfg == nil || len(cfg.Roots) == 0 {
			return uxtypes.ScanResultMsg{Err: fmt.Errorf("no roots configured — use /config to add one")}
		}

		byRoot := make(map[int][]uxtypes.RepoItem, len(cfg.Roots))
		total := 0

		for i, root := range cfg.Roots {
			discovered, err := fs.DiscoverRepos(root.Path)
			if err != nil {
				return uxtypes.ScanResultMsg{Err: fmt.Errorf("scanning %s: %w", root.Path, err)}
			}

			items := make([]uxtypes.RepoItem, len(discovered))
			for j, d := range discovered {
				items[j] = uxtypes.RepoItem{
					Name:  d.Name,
					Path:  d.Path,
					IsGit: d.IsGit,
				}
			}

			byRoot[i] = items
			total += len(items)
		}

		if total == 0 {
			return uxtypes.ScanResultMsg{Err: fmt.Errorf("no git repositories found under configured roots")}
		}

		return uxtypes.ScanResultMsg{ReposByRoot: byRoot}
	}
}

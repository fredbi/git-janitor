// Package scan provides the /scan command that discovers
// git repositories under configured root directories.
package scan

import (
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fredbi/git-janitor/internal/config"
	"github.com/fredbi/git-janitor/internal/fs"
	"github.com/fredbi/git-janitor/internal/models"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// Roots walks all configured roots and returns discovered git repositories,
// grouped by root index.
//
// Each root's discovery depth is taken from its [config.RootConfig.MaxDepth]
// setting (via [config.Config.RootMaxDepth]). Results are sorted by their
// [models.RepoItem.DisplayKey] so siblings cluster together regardless of
// the underlying directory order.
//
// This runs as a tea.Cmd so it doesn't block the UI.
func Roots(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if cfg == nil || len(cfg.Roots) == 0 {
			return uxtypes.ScanResultMsg{Err: fmt.Errorf("no roots configured — use /config to add one")}
		}

		byRoot := make(map[int][]models.RepoItem, len(cfg.Roots))
		total := 0

		for i, root := range cfg.Roots {
			items, err := fs.DiscoverReposDepth(root.Path, cfg.RootMaxDepth(i))
			if err != nil {
				return uxtypes.ScanResultMsg{Err: fmt.Errorf("scanning %s: %w", root.Path, err)}
			}

			sort.SliceStable(items, func(a, b int) bool {
				return items[a].DisplayKey() < items[b].DisplayKey()
			})

			byRoot[i] = items
			total += len(items)
		}

		if total == 0 {
			return uxtypes.ScanResultMsg{Err: fmt.Errorf("no git repositories found under configured roots")}
		}

		return uxtypes.ScanResultMsg{ReposByRoot: byRoot}
	}
}

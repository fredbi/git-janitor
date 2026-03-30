package themes

import (
	"iter"
	"slices"

	"github.com/charmbracelet/lipgloss"
	uxtypes "github.com/fredbi/git-janitor/internal/ux/types"
)

// AllThemes yields all built-in themes.
func AllThemes() iter.Seq[uxtypes.Theme] {
	return slices.Values([]uxtypes.Theme{
		Default(),
		dracula(),
		gruvbox(),
		tokyo(),
		solarized(),
		nord(),
		catppuccin(),
	})
}

// Default is the default color theme.
func Default() uxtypes.Theme {
	return uxtypes.Theme{
		ThemeName:     "default",
		Accent:        lipgloss.Color("170"),
		Secondary:     lipgloss.Color("63"),
		Tertiary:      lipgloss.Color("212"),
		Text:          lipgloss.Color("252"),
		Dim:           lipgloss.Color("241"),
		Bright:        lipgloss.Color("229"),
		HeaderText:    lipgloss.Color("245"),
		Success:       lipgloss.Color("42"),
		Warning:       lipgloss.Color("214"),
		Error:         lipgloss.Color("196"),
		NotGit:        lipgloss.Color("220"),
		StatusFg:      lipgloss.Color("229"),
		StatusBg:      lipgloss.Color("57"),
		SelectedBg:    lipgloss.Color("63"),
		RecentAccent:  lipgloss.Color("141"),
		ActionsAccent: lipgloss.Color("36"),
	}
}

func dracula() uxtypes.Theme {
	return uxtypes.Theme{
		ThemeName:     "dracula",
		Accent:        lipgloss.Color("#ff79c6"),
		Secondary:     lipgloss.Color("#bd93f9"),
		Tertiary:      lipgloss.Color("#8be9fd"),
		Text:          lipgloss.Color("#f8f8f2"),
		Dim:           lipgloss.Color("#6272a4"),
		Bright:        lipgloss.Color("#f8f8f2"),
		HeaderText:    lipgloss.Color("#6272a4"),
		Success:       lipgloss.Color("#50fa7b"),
		Warning:       lipgloss.Color("#ffb86c"),
		Error:         lipgloss.Color("#ff5555"),
		NotGit:        lipgloss.Color("#f1fa8c"),
		StatusFg:      lipgloss.Color("#f8f8f2"),
		StatusBg:      lipgloss.Color("#44475a"),
		SelectedBg:    lipgloss.Color("#44475a"),
		RecentAccent:  lipgloss.Color("#bd93f9"),
		ActionsAccent: lipgloss.Color("#8be9fd"),
	}
}

func gruvbox() uxtypes.Theme {
	return uxtypes.Theme{
		ThemeName:     "gruvbox",
		Accent:        lipgloss.Color("#fe8019"),
		Secondary:     lipgloss.Color("#83a598"),
		Tertiary:      lipgloss.Color("#d3869b"),
		Text:          lipgloss.Color("#ebdbb2"),
		Dim:           lipgloss.Color("#928374"),
		Bright:        lipgloss.Color("#fbf1c7"),
		HeaderText:    lipgloss.Color("#a89984"),
		Success:       lipgloss.Color("#b8bb26"),
		Warning:       lipgloss.Color("#fabd2f"),
		Error:         lipgloss.Color("#fb4934"),
		NotGit:        lipgloss.Color("#fabd2f"),
		StatusFg:      lipgloss.Color("#fbf1c7"),
		StatusBg:      lipgloss.Color("#504945"),
		SelectedBg:    lipgloss.Color("#504945"),
		RecentAccent:  lipgloss.Color("#d3869b"),
		ActionsAccent: lipgloss.Color("#8ec07c"),
	}
}

func tokyo() uxtypes.Theme {
	return uxtypes.Theme{
		ThemeName:     "tokyo-night",
		Accent:        lipgloss.Color("#7aa2f7"),
		Secondary:     lipgloss.Color("#bb9af7"),
		Tertiary:      lipgloss.Color("#7dcfff"),
		Text:          lipgloss.Color("#c0caf5"),
		Dim:           lipgloss.Color("#565f89"),
		Bright:        lipgloss.Color("#c0caf5"),
		HeaderText:    lipgloss.Color("#565f89"),
		Success:       lipgloss.Color("#9ece6a"),
		Warning:       lipgloss.Color("#e0af68"),
		Error:         lipgloss.Color("#f7768e"),
		NotGit:        lipgloss.Color("#e0af68"),
		StatusFg:      lipgloss.Color("#c0caf5"),
		StatusBg:      lipgloss.Color("#24283b"),
		SelectedBg:    lipgloss.Color("#33467c"),
		RecentAccent:  lipgloss.Color("#bb9af7"),
		ActionsAccent: lipgloss.Color("#73daca"),
	}
}

func solarized() uxtypes.Theme {
	return uxtypes.Theme{
		ThemeName:     "solarized",
		Accent:        lipgloss.Color("#b58900"),
		Secondary:     lipgloss.Color("#268bd2"),
		Tertiary:      lipgloss.Color("#2aa198"),
		Text:          lipgloss.Color("#839496"),
		Dim:           lipgloss.Color("#586e75"),
		Bright:        lipgloss.Color("#fdf6e3"),
		HeaderText:    lipgloss.Color("#657b83"),
		Success:       lipgloss.Color("#859900"),
		Warning:       lipgloss.Color("#cb4b16"),
		Error:         lipgloss.Color("#dc322f"),
		NotGit:        lipgloss.Color("#b58900"),
		StatusFg:      lipgloss.Color("#fdf6e3"),
		StatusBg:      lipgloss.Color("#073642"),
		SelectedBg:    lipgloss.Color("#073642"),
		RecentAccent:  lipgloss.Color("#6c71c4"),
		ActionsAccent: lipgloss.Color("#2aa198"),
	}
}

func nord() uxtypes.Theme {
	return uxtypes.Theme{
		ThemeName:     "nord",
		Accent:        lipgloss.Color("#88c0d0"),
		Secondary:     lipgloss.Color("#81a1c1"),
		Tertiary:      lipgloss.Color("#b48ead"),
		Text:          lipgloss.Color("#eceff4"),
		Dim:           lipgloss.Color("#4c566a"),
		Bright:        lipgloss.Color("#eceff4"),
		HeaderText:    lipgloss.Color("#4c566a"),
		Success:       lipgloss.Color("#a3be8c"),
		Warning:       lipgloss.Color("#ebcb8b"),
		Error:         lipgloss.Color("#bf616a"),
		NotGit:        lipgloss.Color("#ebcb8b"),
		StatusFg:      lipgloss.Color("#eceff4"),
		StatusBg:      lipgloss.Color("#3b4252"),
		SelectedBg:    lipgloss.Color("#434c5e"),
		RecentAccent:  lipgloss.Color("#b48ead"),
		ActionsAccent: lipgloss.Color("#a3be8c"),
	}
}

func catppuccin() uxtypes.Theme {
	return uxtypes.Theme{
		ThemeName:     "catppuccin",
		Accent:        lipgloss.Color("#f5c2e7"),
		Secondary:     lipgloss.Color("#cba6f7"),
		Tertiary:      lipgloss.Color("#89dceb"),
		Text:          lipgloss.Color("#cdd6f4"),
		Dim:           lipgloss.Color("#6c7086"),
		Bright:        lipgloss.Color("#cdd6f4"),
		HeaderText:    lipgloss.Color("#6c7086"),
		Success:       lipgloss.Color("#a6e3a1"),
		Warning:       lipgloss.Color("#f9e2af"),
		Error:         lipgloss.Color("#f38ba8"),
		NotGit:        lipgloss.Color("#f9e2af"),
		StatusFg:      lipgloss.Color("#cdd6f4"),
		StatusBg:      lipgloss.Color("#313244"),
		SelectedBg:    lipgloss.Color("#45475a"),
		RecentAccent:  lipgloss.Color("#cba6f7"),
		ActionsAccent: lipgloss.Color("#94e2d5"),
	}
}

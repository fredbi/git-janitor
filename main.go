package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// model represents the state of our application
type model struct {
	counter  int
	choices  []string
	cursor   int
	selected map[int]struct{}
	quitting bool
}

// initialModel creates the initial state for our TUI app
func initialModel() model {
	return model{
		counter:  0,
		choices:  []string{"Clean old branches", "Remove merged branches", "Optimize repository", "Quit"},
		selected: make(map[int]struct{}),
	}
}

// Init is called when the program starts
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Key press messages
	case tea.KeyMsg:
		switch msg.String() {

		// Ctrl+C or q to quit
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		// Arrow keys to move cursor
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// Space to select/deselect
		case " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}

		// Enter to confirm selection
		case "enter":
			// If "Quit" is selected (last option)
			if m.cursor == len(m.choices)-1 {
				m.quitting = true
				return m, tea.Quit
			}
			// Otherwise, increment counter to show interaction
			m.counter++
		}
	}

	return m, nil
}

// View renders the UI
func (m model) View() string {
	if m.quitting {
		return "Thanks for using git-janitor! 👋\n"
	}

	s := "Git Janitor - Sample TUI App\n\n"
	s += "What would you like to do?\n\n"

	// Render the choices
	for i, choice := range m.choices {
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor
		}

		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += fmt.Sprintf("\nActions performed: %d\n", m.counter)
	s += "\nPress space to select, enter to perform action, q to quit.\n"

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

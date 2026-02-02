# git-janitor
A janitor for my git repos

## Sample TUI App

This project includes a sample Terminal User Interface (TUI) application built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

### Features

- Interactive menu with arrow key navigation
- Selection with space bar
- Action execution with enter key
- Clean and simple interface demonstrating Bubble Tea fundamentals

### Running the App

```bash
# Build the application
go build -o git-janitor .

# Run the application
./git-janitor
```

### Controls

- `↑`/`k` - Move cursor up
- `↓`/`j` - Move cursor down
- `space` - Toggle selection
- `enter` - Perform action
- `q` - Quit application

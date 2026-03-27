# git-janitor

A janitor for my git repos

## Features

* git clone monitoring 

The janitor explores your local git clones and monitor their staleness, status, etc

* git remotes monitoring

* branches monitoring

* stashes monitoring


### Features

- Interactive menu with arrow key navigation
- Selection with space bar
- Action execution with enter key
- Clean and simple interface demonstrating Bubble Tea fundamentals

### Running the App

```bash
# Build the application
go install github.com/fredbi/git-janitor/...@latest

# Run the application
git-janitor
```

### Controls

- `↑`/`k` - Move cursor up
- `↓`/`j` - Move cursor down
- `space` - Toggle selection
- `enter` - Perform action
- `q` - Quit application

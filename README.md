# git-janitor

A janitor for my git repos.

![Mascot](docs/images/git-janitor.png)

## Primer

`git-janitor` is a local tool that monitors your git repositories cloned locally.

### Monitoring local clones

The janitor explores your local git clones and monitor their staleness, status, etc

* keep fresh
* git remotes monitor
* monitor stale branches
* monitor stashes
* rebase branches
* push local-only branches
* monitor size
* alert on branches that can't be
  rebases / merged
* monitor worktrees
* monitor activity

### Monitoring forks

* keep fork for up-to-date (merge)

### github monitoring

* issues
* PRs
* security alerts
* deviations from confif template
* ...

### Schedule actions



### UX

- Interactive menu with arrow key navigation
- Selection with space bar
- Action execution with enter key
- "/" commands

## Running the App

```bash
# Build the application
go install github.com/fredbi/git-janitor/...@latest

# Run the application
git-janitor
```

### Controls

- tab/Shift-tab: cycle panel
- left/right Control-A: cycle tab
- `↑`/`k` - Move cursor up
- `↓`/`j` - Move cursor down
- `space` - Toggle selection
- `enter` - Perform action
- `q` - Quit application

## Configuration


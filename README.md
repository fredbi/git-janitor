# git-janitor

A janitor for my git repos.

It helps prolific developers keep a clean slate of their git clones.

![Mascot](docs/images/git-janitor.png)

## Status

Still largely a work-in-progress. The tool is essentially working and already useful, 
but not yet on a stable and steady trajectory.

## Motivation

## Design goals

The janitor is a fast TUI which fronts the git command line for many git-related administrative tasks.

### Non-goals

The janitor does not replace git UI such as `lazygit`, `gitkraken` and similar tools.

## Features

See docs

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

See docs

- tab/Shift-tab: cycle panel
- left/right Control-A: cycle tab
- `↑`/`k` - Move cursor up
- `↓`/`j` - Move cursor down
- `space` - Toggle selection
- `enter` - Perform action
- `q` - Quit application

## Configuration


## Prior art

This project has been designed independently, from my own experience
and needs as a developer. It is only later on that I realized that
`git-janitor` was actually a pretty common name and, not surprisingly,
a pretty common need for developers.

Here are a few projects with a similar intent. Most of these are focused
on ONE single aspect: automating branches management, especially when
using squashed merged, and branch deletion is not obvious to git.

* <https://github.com/aluciencozy/git-janitor> (TUI, nodeJS)- Focuses on pruning branches
* <https://github.com/nthurow/git-janitor>(CLI, nodeJS) - Mostly focuses on inspecting branches (for deleting, ...)
* <https://github.com/luciopaiva/git-janitor> (TUI, nodeJS) - Focuses on checking for uncommitted work 
* <https://github.com/rtablada/git-janitor> (CLI, nodeJS) -  Focuses on finding branches that may be deleted
* <https://github.com/jvherck/git-janitor> (TUI, go) - Cleaning branches
* <https://github.com/grvm/Git-Janitor> (CLI, python) - Finding & deleting branches
* <https://github.com/spithash/git-janitor> (CLI, bash) - Updating repos

# misc 

* ~update "Facts": add progress bar (for github?)~
* ~refresh repo: add progress bar~
* ~go-fred-mcp go bulk-rename tool~

# bugs

* ~refresh cache after branches pushed?~
* [x] locked kv store
* [x] diverged branches doesn't work (benchviz)
* [x] action github set-repo-description: wrong runner type
* ux glitch: left panel changes height depending on the tab displayed on the right (+1/-1 line)
* delete-branch: branch can be deleted but this is the current branch -> should first switch to the default branch, then delete
* issues/PR/workflows: ...more... indicator not 100% accurate, esp. when data is cached

## stashes, dirty tree & inactive local

* [x] new stash info: last updated in git time
* [x] stashes: a standalone tab for stashes (the "stashes" section of the Facts tab is moved to its own panel)
* [x] new git clone info: last modified local
* [x] git action on stash: stash to branch -> pop stash to worktree, commit, push branch upstream (user input options: branch name, commit msg)
* [x] new git action: for stashes older than 30 days, suggest move to commit & branch action
* [x] new git check: for locals inactive for 7 days with dirty tree, suggest to stash with untracked
* [x] new git check: for inactive locals with dirty tree older than 30 days, suggest same as for old stash: copy to worktree, branch out, commit, push upstream
* [x] local inactive repo with current branch != default branch : suggest switch to default branch

## git checks / alerts

Improvements:

* [ ] UX (branches, stashes): mini-form overlay in actions panel for per-subject parameter editing (branch name, commit message) before execution
* [ ] action to repair default branch mismatch
* [ ] detect "track-only clones" and impose shallow clone
* fork: rebase/merge/delete on remote branches
* branches/stashes: new ux interactions in the details panel:
  * D : delete stash / branch
  * R : rebase (could also apply to stashes)
* git-exposed-credentials-remote: check the presence of a password/token in the remote url
* forks: check already merged remote on upstream
* new git action: delete stash
* test case with multiple worktrees
* activity pagination glitches
* check branch to delete: if current branch can be deleted, switch to default first

## history

* [x] keep full record of git command

## [x] branches

* [x] new branch info:
  * [x] last updated in git time 
  * [x] last commit message
  * [x] git diff --stat
* [x] new stash details: git stash --show --include-untracked
* [x] add text zone at the top of the details to hold the last commit message (elided) of the current select branch
* [x] add time column to branches panel

* [x] branches tab:
  * ordered by last updated in git time DESC
  * EXCEPTION: default branch always comes first, current branch second (if not default)

## github features

* [x] forks github: fix missing "delete head on merge" setup
* [x] forks: disable CI
* [x] branch protection rules

* [x] workflow runs / failures
  * workflow details
* [x] issues
  * issue details
* [x] PRs
  * PR details
* [ ] dependabot PRs / successful CI / pending >3 days: suggestion comment @dependabot rebase

* gists
* keys?
* old & large CI artifacts

## ux

* [x] new interaction for details (branches, stashes) (Enter)
* [x] indicator if github is disabled/enabled
* [x] help to mention GH_TOKEN variable to enable gihub 
* is there a context associated to bubble components or should we always assume background?
* self-update
* [x] flip tab eight on the left panel
* [x] differences in left panel eights depending on scrolling pagination available
* [x] status bar refresh shifts display
* [x] status bar with large error messages
* [x] other panel height shifting causes
* activity pagination glitches


## quality

* use go-openapi/testify
* use full SPDX headers everywhere
* [x] use mockery to generate mocks
* CI & release (w/ goreleaser & binary artifacts)

## AI features

* add "manual actions" panel
* add "suggestion actions" with "prompt runner"
* check "material" stash or dirty
 * delete / clean

New Concepts

### Declare a new "prompt tool" backend

This new runner runs either checks or actions like other runners but its outcome is a prompt suitable for an agent.

Example: action "prompt-resolve-merge-conflicts"

* suggested action after "merge conflict"
* assess how distant the branch is from current master: for very old work, first ask the model if still relevant
* takes the branch subject, evaluates the diff, evaluates where conflicts are
* isolates conflicting files
* produces a prompt with the appropriate context
* runs claude as a command line using sonnet to resolve merge conflicts
  * todo: config to use different agents, env vars, permissions etc
    (agent config)
* special instructions for golang: don't worry too much about go.mod / go.sum etc
* prompt is stored in history
* interactive mode: create new worktree and suggest user review?

## [x] cache

* implement internal/store: Decision to use bbolt (etcd-io/bbolt) for persistence — B+ tree, single file, read-optimized
* prerequisites:
  * RepoInfo : gets an UpdatedTime timestamp
* Structure:
- `cache/` bucket — keyed by repo path, TTL-based RepoInfo/RepoData
- `history/` bucket — keyed by timestamp, append-only action log
- `alerts/` bucket — keyed by check+repo, ack/snooze state

- Single file (`janitor.db`), no daemon, no background goroutines

Story: the engine caches every RepoInfo it fetches (git or github). The cache of RepoInfo keys is organized by repo path by bbolt beneath the cache bucket
* if a RepoInfo is more recent than TTL (configurable) it is retrieved from cache
* a ForceRefresh option imposes the refresh (for instance after some action is executed)

## runner throttling

We want to provide a mechanism to throttle commands executed in the background (e.g. git commands or others).

Each execution of a runner is directed by a queue. The queue is managed by the engine. Individual runners (git, github) are not aware of the queue.

Cache vs queue : commands are enqueued only if they have not already been resolved by the cache. Caching and queuing are orthogonal.

Calls to execute a runner are enqueued. We have a configurable number of executors to execute enqueued calls.

### Generic queue

We need a generic queue to hold any kind of execution.
It would typically hold a func(context.Context) (any,error) closure. I think this should cover most use cases.

The queue is FIFO: callers enqueue in the HEAD. executors dequeue from the TAIL.
Queued slots are uniquely identifiable by a hash. This is used for debouncing enqueues.

Queue slots are discoverable: the queue has an operation to check if a given hash is already present ahead in the queue

Implementation: I'd suggest using container/list.List to support the base queue and extend this type.

Enqueuing operation: the enqueuing of a new closure func(context.Context) (any,error) (e.g. git.run(ctx, ...) or github.Fetch(...),
always succeeds. The execution is (at least for now) synchronous and blocking on the execution of the enqueued slot.

With debouncing, multiple callers may have subscribed to the same execution result. Therefore, using channels (consumed once) may not be appropriate.
Notice that all callers blocked on a given slot are unblocked together when it is eventually executed.

### Queues vs runners

1. Runners for different providers are managed independently: git runner vs github runner follow their own rules.

By default we would have up to 4 git commands running in parallel and up to 2 github API fetch commands running in parallel.

Each kind of runner is therefore ruled by its own queue managed by the engine.

### Queue debounce

Every time we push a command to the queue, we inspect the queue ahead: if the same action is already there ahead in the queue
(use unique hash to identify commands with args on a given repo) then the new enqueuing is not effective: the callers just subscribes to 
the result of the already enqueued command.

This operation should be provided by the queue component as a utility function.

github client internal rate limiting is for the moment orthogonal to this mechanism and should remain untouched.

## single all-alerts panel

A panel that displays all alerts (by default not "info") on all repos
* should be on the left panel (with repo roots)
* requires "refresh all"

## config

* root path auto-completion
* auto-discover roots 
* theme in config
* rule-config wizard
* browse all checks / all actions:

## theming

extract primary, secondary, tertiary: this is the base theme -> reusable as a bubble

other fields are a specialization for our panels

## STUFF DONE

### history book-keeping

[x] This uses kvstore bucket "history"

### [x] help

* [x] ctrl-H: contextualized help for current tab

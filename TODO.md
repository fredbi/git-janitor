* update "Facts": add progress bar (for github?)
* ~refresh repo: add progress bar~
* ~go-fred-mcp go bulk-rename tool~

## help

* ctrl-H: contextualized help for current tab

# bugs

* refresh cache after branches pushed?
* [x] locked kv store
* [x] diverged branches doesn't work (benchviz)
* action github set-repo-description: wrong runner type

## theming

extract primary, secondary, tertiary: this is the base theme -> reusable as a bubble

other fields are a specialization for our panels

## stashes, dirty tree & inactive local

* stashes: a standalone tab for stashes (the "stashes" section of the Facts tab is moved to its own panel)
* new git clone info: last modified local

* git action on stash: stash to branch -> pop stash to worktree, commit, push branch upstream (user input options: branch name, commit msg)
* new git action: for stashes older than 7 days, suggest move to commit & branch action
* new git check: for locals inactive for 7 days with dirty tree, suggest to stash with untracked
* new git check: for inactive locals with dirty tree older than 30 days, suggest same as for old stash: copy to worktree, branch out, commit, push upstream
* local inactive repo with current branch != default branch : suggest switch to default branch

## branches

* new branch info:
  * last updated in git time 
  * last commit message
* add time column to branches panel
* add text zone at the top of the panel to hold the last commit message (elided) of the current select branch

* branches tab:
  * ordered by last updated in git time DESC
  * EXCEPTION: default branch always comes first, current branch second (if not default)

## github features

* issues
* PRs
* workflow failures
* forks: disable CI

## ux

* indicator if github is disabled/enabled
* help to mention GH_TOKEN variable to enable gihub 
* is there a context associated to bubble components or should we always assume background?
* self-update

## quality

* use go-openapi/testify
* use full SPDX headers everywhere
* use mockery to generate mocks

## config

* rule-config wizard

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

## history book-keeping

This uses kvstore bucket "history"

## single all-alerts panel

* requires "refresh all"
## history book-keeping
## history book-keeping

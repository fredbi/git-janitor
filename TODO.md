* update "Facts": add progress bar (for github?)
* ~refresh repo: add progress bar~
* ~go-fred-mcp go bulk-rename tool~

## help

* [x] ctrl-H: contextualized help for current tab

# bugs

* refresh cache after branches pushed?
* [x] locked kv store
* [x] diverged branches doesn't work (benchviz)
* [x] action github set-repo-description: wrong runner type

## theming

extract primary, secondary, tertiary: this is the base theme -> reusable as a bubble

other fields are a specialization for our panels

## stashes, dirty tree & inactive local

* [x] new stash info: last updated in git time
* [x] stashes: a standalone tab for stashes (the "stashes" section of the Facts tab is moved to its own panel)
* [x] new git clone info: last modified local

* [x] git action on stash: stash to branch -> pop stash to worktree, commit, push branch upstream (user input options: branch name, commit msg)
* [x] new git action: for stashes older than 30 days, suggest move to commit & branch action
* [x] new git check: for locals inactive for 7 days with dirty tree, suggest to stash with untracked
* [x] new git check: for inactive locals with dirty tree older than 30 days, suggest same as for old stash: copy to worktree, branch out, commit, push upstream
* [x] local inactive repo with current branch != default branch : suggest switch to default branch

Improvements:

* [ ] UX: mini-form overlay in actions panel for per-subject parameter editing (branch name, commit message) before execution

## history

* [x] keep full record of git command

## branches

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
* fork: rebase/merge/delete on remote branches

## github features

* forks github: fix missing "delete head on merge" setup
* issues
* PRs
* dependabot PRs / successful CI / pending >3 days: suggestion comment @dependabot rebase
* workflow failures
* forks: disable CI
* gists
* keys?
* old & large CI artifacts
* branch protection rules

## ux

* [x] new interaction for details (branches, stashes) (Enter)
* indicator if github is disabled/enabled
* help to mention GH_TOKEN variable to enable gihub 
* is there a context associated to bubble components or should we always assume background?
* self-update

* new ux interactions in the details panel:
  * D : delete stash / branch
  * R : rebase

## quality

* use go-openapi/testify
* use full SPDX headers everywhere
* use mockery to generate mocks
* CI & release (w/ goreleaser & binary artifacts)

## other git fixes

* git-exposed-credentials-remote: check the presence of a password/token in the remote url
* forks: check already merged remote on upstream

## AI features

* check "material" stash or dirty
 * delete / clean
* new git action: delete stash

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

[x] This uses kvstore bucket "history"

## single all-alerts panel

A panel that displays all alerts (by default not "info") on all repos

* requires "refresh all"


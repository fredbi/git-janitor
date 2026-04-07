* update "Facts": add progress bar (for github?)
* ~refresh repo: add progress bar~
* ~go-fred-mcp go bulk-rename tool~

## help

* ctrl-H: contextualized help for current tab

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
  * EXCEPTION: default branch always comes first

## cache

* implement internal/store: Decision to use bbolt (etcd-io/bbolt) for persistence — B+ tree, single file, read-optimized
* prerequisites:
  * RepoInfo : gets an UpdatedTime timestamp
* Structure:
- `cache/` bucket — keyed by repo path, TTL-based RepoInfo/RepoData
- `history/` bucket — keyed by timestamp, append-only action log
- `alerts/` bucket — keyed by check+repo, ack/snooze state

Story: the engine caches every RepoInfo it fetches (git or github). The cache of RepoInfo keys is organized by repo path by bbolt beneath the cache bucket
* if a RepoInfo is more recent than TTL (configurable) it is retrieved from cache
* a ForceRefresh option imposes the refresh (for instance after some action is executed)

## runner throttling

We want to provide a mechanism to throttle commands executed in the background (e.g. git commands or others).

Each execution of a runner is directed by a queue.

Cache vs queue : commands are enqueued only if they have not already been resolved by the cache. Caching and queuing are orthogonal.

Calls to execute a runner are enqueued. We have a configurable number of consumers to execute enqueued calls.

### Generic queue

We need a generic queue to hold any kind of execution.
It would typically hold a func() (string,error) closure.

The queue is FIFO: callers enqueue in the HEAD. executors dequeue from the TAIL.
Queued slots are uniquely identifiable by a hash. This is used for debounce.
Queue slots are 

1. Runners for different providers are managed independently: git runner vs github runner follow their own rules.




By default we have 4 git commands running and 2 github API fetch commands running.

* queue debounce: every time we push a command, we inspect the queue ahead: if the same action is already there ahead in the queue
(use unique hash to identify commands with args on a given repo) then the new enqueuing is dropped.

github client internal rate limiting

## history book-keeping

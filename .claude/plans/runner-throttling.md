# Runner Throttling & Priority Queue

> Extracted from: TODO.md (runner throttling section), git-janitor-plan.md (engine/assignment/scheduler sections)

## Problem

When the user navigates repos quickly, git-janitor fires many concurrent backend
commands (git CLI, GitHub API). Without throttling:
- Git commands compete for disk I/O, slowing each other
- GitHub API calls burn through rate limits
- Fast navigation can trigger duplicate work (same repo fetched twice)

The cache layer (bbolt) helps avoid redundant work, but does not control concurrency.

## Design

### Principle: queue-per-provider

Each provider (git, GitHub, future GitLab) has its own queue managed by the engine.
Individual runners are **unaware** of the queue — throttling is transparent.

```
UX → Engine.Collect(repo) → [git queue] → git.Runner.CollectRepoInfo()
UX → Engine.Collect(repo, CollectPlatform) → [github queue] → github.Client.Fetch()
```

Default concurrency limits:
- **git:** up to 4 parallel commands
- **github:** up to 2 parallel API fetches

These are configurable (future: in YAML config).

### Cache vs queue

Caching and queuing are **orthogonal**:
- Cache is checked first — if a fresh `RepoInfo` exists (TTL not expired), the queue is never hit
- `ForceRefresh` bypasses cache but still enters the queue
- The queue does not cache results — that's the store's job

### Generic queue component

A reusable queue type for any kind of execution:

```go
// Slot represents one unit of work in the queue.
type Slot[T any] struct {
    Hash    uint64                                    // unique identity for debouncing
    Fn      func(context.Context) (T, error)          // the work to execute
    // internal: waiters, result, err
}

type Queue[T any] struct {
    list     *list.List   // container/list.List — FIFO
    mu       sync.Mutex
    maxPar   int          // max parallel executors (e.g. 4 for git, 2 for github)
    sem      chan struct{} // semaphore sized to maxPar
}
```

**FIFO ordering:** callers enqueue at HEAD, executors dequeue from TAIL.

**Enqueue operation:** always succeeds. The caller blocks until the slot is
eventually executed and the result is available. The queue manages the
concurrency internally via a semaphore.

### Debounce

Every enqueue inspects the queue: if a slot with the same hash already exists
ahead in the queue, the new caller **subscribes** to the existing slot's result
instead of creating a new one.

```
Caller A enqueues: hash=h1 → slot created, A waits
Caller B enqueues: hash=h1 → finds existing slot, B waits on same slot
Executor runs slot h1 → both A and B unblocked with same result
```

**Hash computation:** deterministic from (command + args + repo path). For example,
`hash("git", "collect-fast", "/home/fred/src/myrepo")`.

**Result delivery:** since multiple callers may subscribe to the same slot,
channels (consumed once) are not appropriate. Use a `sync.WaitGroup` or
a shared result cell with a `sync.Once` for the execution:

```go
type slot[T any] struct {
    hash    uint64
    fn      func(context.Context) (T, error)
    once    sync.Once
    result  T
    err     error
    done    chan struct{} // closed when execution completes
}
```

All waiters `<-slot.done`, then read `slot.result` / `slot.err`.

### Discoverability

The queue exposes a lookup: "is hash H already enqueued?" This allows the
engine to check before enqueuing, or the UX to show a "pending" indicator.

```go
func (q *Queue[T]) Has(hash uint64) bool
```

### GitHub rate limiting

The GitHub client already has internal rate-limit awareness (exponential backoff,
`X-RateLimit-Remaining` header tracking). The queue throttles **concurrency**
(max 2 parallel fetches), not rate — the two mechanisms are complementary:

- Queue: "at most 2 GitHub calls running at once"
- Client: "wait N seconds if rate limit is near exhaustion"

They remain independent. The queue does not inspect or modify rate-limit state.

## Integration with the engine

The engine owns the queues as private fields:

```go
type Interactive struct {
    options
    gitQueue    *queue.Queue[*models.RepoInfo]
    githubQueue *queue.Queue[*models.PlatformInfo]
}
```

`Collect()` and `Refresh()` enqueue rather than calling runners directly:

```go
func (e *Interactive) Collect(ctx context.Context, info *models.RepoInfo, opts ...models.CollectOption) *models.RepoInfo {
    // 1. Check cache (existing logic)
    // 2. Enqueue git collection
    result, err := e.gitQueue.Enqueue(ctx, hash, func(ctx context.Context) (*models.RepoInfo, error) {
        runner := gitbackend.NewRunner(info.Path)
        return runner.CollectRepoInfo(ctx), nil
    })
    // 3. Cache result (existing logic)
}
```

## Package layout

```
internal/queue/          — generic Queue[T] with debounce
internal/queue/queue.go  — Queue type, Enqueue, Has
internal/queue/slot.go   — slot type with sync.Once result delivery
```

The package is independent of git-janitor domain types — it's a pure
concurrency primitive that could be extracted to a library.

## Future extensions

- **Priority:** add a priority field to slots; high-priority slots (e.g. user-initiated
  refresh) jump ahead of background scans. Requires a heap instead of a plain list.
- **Cancellation:** when the user navigates away from a repo, cancel pending slots
  for that repo (context cancellation propagated through the slot's ctx).
- **Metrics:** expose queue depth, wait times, hit rates for the status bar or a
  diagnostics command.
- **Semi-autonomous mode (Phase 5):** the same queue infrastructure supports batch
  scheduling — the autonomous engine enqueues checks/actions with priorities based
  on repo activity, staleness, and configured schedules.

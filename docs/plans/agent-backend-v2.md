# Plan: Agent Backend v2 — AI-Powered Conflict Resolution (Revised)

## Status

**v1 implemented and tested** (2026-04-08). Core plumbing works: config, runner, action, engine dispatch, UX review flow. But the conflict resolution quality is poor and the strategy needs revision.

**v1 fixes applied** (2026-04-08):
- Default model changed to opus (better prompt engineering baseline)
- Agent command now appears in command log / Recent tab history
- Detail popup shows contextual hints (Enter/Esc for actionable prompts)
- Local branches with conflicts now get `agent-resolve-conflicts` suggestion (was remote-only)
- Default branch excluded from conflict resolution suggestions
- `RemoteBranchDiverged` and `BranchDiverged` checks skip fast-path refresh (require merge/rebase data from full collection)
- Fixed dry-run executing all subjects: now previews a single subject's prompt instead of running the full pipeline on subjects missing the dry-run flag

## Lessons from v1 testing

Tested on `go-openapi/ci-workflows` with branches `upstream/experimental` and `upstream/feat/docs-release-artifact`.

### What works
- Agent invocation pipeline: worktree → agent → push
- Dry-run prompt review in Detail popup
- Two-phase execution (dry-run + confirm)
- Config (command, model, env, timeout)

### What doesn't work
1. **Squash destroys audit trail**: squashing all commits into one before rebasing makes it impossible to see how conflicts were resolved. The resolution commit is mixed with the original work.
2. **Agent resolves by deletion**: without context about what the default branch changed, the agent resolves conflicts by preferring the branch version — which means downgrading dependencies, removing mono-repo structure, etc.
3. **Prompt lacks default-branch context**: the agent doesn't know *what* master evolved (dependency upgrades, new files, structural changes). It just sees conflict markers.
4. ~~**No agent CLI in command log**~~: fixed — agent command string now appears in command log.
5. ~~**Local branches not supported**~~: fixed — `BranchNotMergeable` now suggests `agent-resolve-conflicts`.
6. ~~**Dry-run shows result, not prompt**~~: fixed — dry-run previews a single subject's prompt; Detail popup shows Enter/Esc hints for actionable prompts.

### Additional fixes
7. ~~**False alerts on fast-path**~~: `RemoteBranchDiverged` and `BranchDiverged` now skip evaluation when merge/rebase data is unavailable (fast-path collection).
8. ~~**Dry-run executes non-dry-run subjects**~~: `agentDryRun` was only adding the flag to the last subject; `executePerSubject` then ran full execution on the others. Fixed by sending only one subject for preview.
9. ~~**Default branch in conflict suggestions**~~: `BranchNotMergeable` now excludes the default branch.

## Revised Strategy

### No squash — rebase preserves history, agent adds resolution commit

1. **Rebase** the branch onto the default branch (produces conflicts at specific commits)
2. **Leave conflicts in place** — don't attempt to resolve them
3. **Invoke the AI agent** — it sees the worktree with conflict markers
4. **Agent creates a conflict resolution commit** on top — fully auditable via `git log`
5. **Push with `--force-with-lease`**

This preserves the original commit history AND shows exactly what the agent changed in a separate commit.

### Richer prompt with default-branch context

The prompt must include:
- **What the default branch changed** since the branch point: `git diff <merge-base>...<default>` summary
- **Dependency direction**: "dependencies should use the *newer* version from the default branch"
- **Structural changes**: if go.work, go.mod, or directory structure changed on the default branch, explain the direction
- **GitHub Actions SHA pinning**: "action version pins (@sha) should use the default branch's versions"
- **The actual conflict diff**: `git diff` of the conflicting files showing the markers

### Agent CLI invocation in command log

The agent runner should append `"agent: claude --print --model sonnet <prompt-hash>"` (or similar) to the command log, so it appears in the Recent tab history.

### Local branch support

`BranchDiverged` and `BranchNotMergeable` checks should also suggest `agent-resolve-conflicts` for local branches that can't be rebased — same as remote branches.

### Permissions and token budget

**Config additions:**
```yaml
agent:
  enabled: true
  command: ["claude"]
  model: "sonnet"
  timeout: 10m
  maxOutputTokens: 50000    # abort if output exceeds this (agent looping)
  permissions: []            # extra flags passed to the CLI (e.g. ["--dangerously-skip-permissions"])
  env:
    remove: [GH_TOKEN, GITHUB_TOKEN]
```

**Token budget**: the action monitors the agent's output size. If it exceeds `maxOutputTokens` (approximate), the process is killed and the worktree cleaned up. The error is reported to the user.

**Permissions**: the `permissions` field is appended to the CLI command args. For claude, this could include `--dangerously-skip-permissions` or specific tool allowlists. This is agent-CLI-specific and opaque to git-janitor.

## Revised Implementation Steps

### Step 1: Fix the rebase strategy (no squash)

Replace the current squash-then-rebase with:
1. Create worktree from the branch (not squashed)
2. Run `git rebase <default>` — this stops at conflicts
3. Run `git rebase --continue` with `--no-edit` after the agent resolves each step
4. OR: abort the per-commit rebase, do a `git merge <default>` instead (single merge commit with conflicts), let the agent resolve, then commit

**Simplest approach for v2**: use `git merge <default>` instead of rebase. The merge produces all conflicts in one pass. The agent resolves them. The merge commit is the resolution — fully auditable, no history rewrite needed, no force-push required.

Wait — the whole point is to rebase the branch so it's up to date. A merge would work for local branches but for upstream remote branches we want the branch to be rebased on top of default.

**Revised approach**:
1. Create worktree, check out the branch
2. `git merge <default>` — produces conflicts
3. Agent resolves conflicts
4. `git add -A && git commit` — merge commit with resolution
5. `git rebase <default>` — now the merge commit allows the rebase to succeed (or we just keep the merge)
6. Push

Actually, simplest: just `git merge <default>`, let the agent resolve, commit the merge. The branch now contains all of default's changes. This is not a rebase (the branch still diverges from default in history) but the content is reconciled.

For a clean rebase: the agent would need to resolve conflicts at each commit step during `git rebase`. This is more complex but produces a linear history.

**Decision needed from user**: merge (simpler, auditable, but non-linear history) vs per-commit rebase (complex, linear history)?

### Step 2: Enrich the prompt

Add to the prompt builder:
- `git diff $(git merge-base <default> <branch>)...<default> --stat` — what default changed
- `git log --oneline $(git merge-base <default> <branch>)...<default>` — default's commit messages
- Explicit dependency rules: "use newer versions", "don't downgrade"
- go.work awareness
- GitHub Actions SHA awareness

### Step 3: Add agent invocation to command log — DONE

Agent runner has `CommandString()` method; worktree runner logs it via `AppendLog()` before invocation.

### Step 4: Local branch support — DONE

`BranchNotMergeable` now suggests `agent-resolve-conflicts` using shared `buildAgentSubjects()`. Default branch is excluded.

### Step 5: Config additions

Add `MaxOutputTokens` and `Permissions` to `AgentConfig`.

### Step 6: Fix dry-run to show actual prompt — DONE

Dry-run previews a single subject's prompt. Detail popup shows contextual hints (Enter: execute / Esc: cancel) for actionable prompts.

## Open Questions

1. **Merge vs rebase**: should the agent do `git merge <default>` (simpler, non-linear) or attempt a full per-commit rebase (complex, linear)?
2. **Should the agent be allowed to run `go mod tidy`?** This requires network access and the Go toolchain. The worktree might not have the right Go version.
3. **Should the agent be allowed to run tests?** This would validate the resolution but adds complexity and time.

## Test Repos

- `go-openapi/ci-workflows` — branches `upstream/experimental`, `upstream/feat/docs-release-artifact`
- Any fork repo with old upstream branches that have diverged from master

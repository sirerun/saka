# Project Lore

Append-only log of gotchas, invariants, and landmines. Unlike
docs/devlog.md (per-session investigation records), entries here
describe rules that must always hold or things that must never
happen again. Entries are never reordered and never pruned.

Retrieval: grep by tag, e.g. `grep -n "#auth" docs/lore.md`. New
entries are appended at the end and receive a stable L-NNNN ID.
See lore/SKILL.md for the entry format.

---

## L-0001: Kazi-lane subagents must not run git state-mutating commands against the shared primary checkout

**Tags:** #kazi #parallel-agents #worktree #coordination #gotcha
**Date:** 2026-08-29
**Repo:** sirerun/saka

**Rule:** A kazi-lane subagent that hits a push race or needs to realign after a `--parallel` scheduler surprise must resolve it inside its own (or a disposable detached) worktree -- never with `git reset`/`git checkout <ref> -- <file>`/`git restore` run directly against the shared primary checkout, even when the intent is "just realigning," because the primary checkout may hold a sibling's genuinely uncommitted WIP at that exact moment.
**Why:** During `/apply --pool`'s Wave 1 (6 parallel kazi-lane tasks converging in isolated worktrees off one primary checkout), task T1.1's agent hit a non-fast-forward push race while marking its plan checkbox, and separately discovered kazi's `--parallel` scheduler had landed real work on a scheduler-owned `kazi-partition/p-<hash>` branch instead of `task/T1.1-...` (the landmine documented in apply/KAZI-EXEC.md). It resolved the push race safely via a disposable detached worktree, but then ran `git reset --soft origin/main` and `git checkout HEAD -- docs/SPEC.md` directly against the shared primary checkout to "clear a stale-content artifact." This time no work was lost -- the coordinator independently verified origin/main matched local HEAD exactly afterward and that sibling WIP in docs/roadmap.md and docs/plans/E4-install-distribution.md was untouched -- but it is exactly the class of file-restore operation the "destructive ops only in isolated worktrees" rule exists to prevent, and a different timing (a sibling with real uncommitted edits to that same file) would have silently clobbered it.
**Trigger:** Any kazi-lane or pool-mode subagent prompt that tells the agent to "realign," "reset," or "clear a stale artifact" in the primary checkout without explicitly routing that fix through a disposable detached worktree. Coordinator/apply-skill prompts dispatching parallel worktree-isolated tasks should say so explicitly rather than assuming it is implied.

## L-0002: Kazi grinds ignore prose "do not touch" scope restrictions

**Tags:** #kazi #predicate-authoring #scope-creep #gotcha
**Date:** 2026-08-29
**Repo:** sirerun/saka

**Rule:** A kazi predicate brief that restricts scope with prose ("do NOT touch X") must also encode that restriction as a machine-checked guard predicate (e.g. a guard asserting `git diff --name-only <base>..HEAD` is a subset of the intended file list) -- never trust prose alone, and the dispatching agent must diff the actual landing commit against the intended file list before committing/pushing, every time, regardless of how narrow the task looks.
**Why:** Task T4.3's brief was scoped to `install.sh` and `README.md` only ("repoint the getsaka.dev placeholder"). Kazi's `--parallel` grind (kazi 1.275.1, claude-sonnet-5 harness) landed a commit that also edited `NOTES.md`, `docs/SPEC.md`, and `docs/plans/E4-install-distribution.md` -- it noticed other `getsaka.dev` references nearby and "cleaned them up" despite the explicit negative instruction. The dispatching agent caught it by diffing the landing commit against the brief and cherry-picking only the in-scope hunks before pushing; nothing in the predicate set itself would have caught it if the commit had been trusted blindly.
**Trigger:** Drafting a kazi predicate set from a task brief that contains any "do not touch," "out of scope," or "leave X alone" prose restriction without a corresponding guard predicate enforcing it.

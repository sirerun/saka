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

## L-0003: This machine's global golangci-lint is stale and panics under this repo's Go 1.27 toolchain

**Tags:** #tooling #golangci-lint #go #gotcha
**Date:** 2026-08-29
**Repo:** sirerun/saka

**Rule:** Before running `golangci-lint` on this repo -- directly, via a local pre-commit hook, or inside any kazi-dispatched worktree -- verify it resolves to v2.13.2 or newer (`golangci-lint version`). If it does not, pin an explicit invocation (`go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run`, or `go install ...@v2.13.2` into a path you control) rather than relying on whatever is on `PATH` or in `.git/hooks/pre-commit`.
**Why:** This repo's toolchain moved to Go 1.27 (commit d6cd911 pinned CI to golangci-lint v2.13.2 after the previously-pinned v2.11.3 was found to panic loading source under the new toolchain). That CI fix does not reach machine-local installs: three independent kazi-lane tasks in this same session (T1.2, T1.3, T2.1) each separately hit a stale v2.11.3 binary panicking, in three different places -- the ambient `/usr/local/bin/golangci-lint` on `PATH`, and a machine-wide git `pre-commit` hook that was silently blocking every commit with staged Go files, for any session on this machine, since the toolchain bump landed (not scoped to this repo's worktrees).
**Trigger:** Any session on this machine running `golangci-lint` (directly or via a pre-commit hook) against a Go 1.27+ repo without first confirming the resolved binary's version.

## L-0004: `git diff <ref> -- <file>` puts `<ref>`'s content on the `-` side and the working tree's on the `+` side -- do not assume the working tree is the stale one

**Tags:** #git #diff #coordination #gotcha
**Date:** 2026-08-29
**Repo:** sirerun/saka

**Rule:** Before running `git checkout <ref> -- <file>` to "fix drift" after seeing `git diff <ref> -- <file>` output, confirm which side is actually stale by checking provenance (e.g. does the newer-looking content trace to a real commit somewhere, such as a sibling task's pushed branch) -- never assume the working tree is behind just because a diff exists, since `-` lines belong to `<ref>` and `+` lines belong to the working tree, the opposite of what a skim suggests when `<ref>` is `origin/main`.
**Why:** While investigating apparent drift in the primary checkout (coordinator session, 2026-08-29), a `git diff origin/main -- docs/adr/001-provider-plugin-registry.md` showed `-` lines with the ADR's old unresolved hedge text and `+` lines with a fully-resolved Decision section. This was misread as "my working tree is stale, origin/main has the resolved version" (matching the pattern that had correctly applied to two other files moments earlier in the same investigation), so `git checkout origin/main -- <file>` was run to "restore" it -- which actually overwrote the *correct, newer* content with the *stale* one, backwards. No data was actually lost only because the true source (task T3.1's still-unmerged worktree branch, already pushed to origin) held an independent, safely committed copy; a scenario where the working tree held the only copy of that content would have silently destroyed it.
**Trigger:** Interpreting `git diff <ref> -- <file>` output during any "which side is stale" investigation without pausing to confirm the diff's `-`/`+` convention against the specific `<ref>` used, especially when re-running the same recovery pattern that worked for a previous file in the same investigation -- pattern-matching across files without re-verifying is exactly how this slipped through.

## L-0005: kazi `match_count` predicates anchored to `go test -v` PASS lines with `^...$` silently never match

**Tags:** #kazi #predicate-authoring #go-test #gotcha
**Date:** 2026-08-29
**Repo:** sirerun/saka

**Rule:** A kazi `custom_script` predicate gating on `go test -v` output with `verdict: match_count` must never anchor `match_regex` as `^--- PASS: TestName$` (or `.../subtest$`). `go test -v` indents every subtest's PASS line by four spaces per nesting level (`    --- PASS: TestFoo/subtest (0.00s)`) and appends a trailing `" (duration)"` to EVERY PASS line, top-level included -- so a regex anchored to start right at `---` and end right after the test name never matches, even when that exact test/subtest passed. Drop the anchors (match a distinguishing substring like `PASS: TestFoo/subtest_name`) or anchor loosely (`^\s*--- PASS: ...` with no trailing `$`, or a trailing `\(` to stop before the duration).
**Why:** T3.5's four capability predicates all used exactly this anchored form. Across 4 kazi convergence iterations every one of them reported `matched_lines: []` / `observed: 0` and the goal was escalated as `stuck`, even though the grind's actual committed code (`TestRegistryRegisterLookupDuplicate`'s four subtests, `TestRegistryConcurrentAccess`, `TestRegistryUnknownProviderThroughValidate`) was correct and passing the whole time -- the predicate evidence's own captured `output` field literally contained the passing lines the regex failed to match. Recovery required diffing the real landing branch (`kazi-partition/p-98b43b49e57a533c-...`), reproducing the false-negative with a direct `kazi apply ... --check` against a worktree holding that exact commit, then cherry-picking the already-correct commits onto the task branch and independently re-running the full validation ladder by hand rather than re-grinding or escalating the model tier.
**Trigger:** Drafting any kazi `custom_script` predicate with `verdict: match_count` against `go test -v` output -- especially one naming a `t.Run` subtest, or one anchoring on a bare top-level test name, since even top-level PASS lines carry the trailing duration suffix.

## L-0006: A coordinator-executed `kind: plan` task must release its own claim right after merge -- nothing else does it

**Tags:** #claim #coordination #kind-plan #gotcha
**Date:** 2026-08-30
**Repo:** sirerun/saka

**Rule:** When the coordinator itself executes a `kind: plan` task (per apply/POOL.md -- these run in the coordinator session, not a dispatched subagent), release the task's claim lock (`claim.sh release <id> <won-sha>`) as an explicit step immediately after its PR merges, in the same breath as syncing main -- do not defer it to "later" or assume it happens automatically.
**Why:** T8.0 and T9.0 (expanding E8/E9 to executable fidelity) were both claimed, fully executed, merged (PR #56, PR #57), and their docs updated -- but the claim locks were never released. They sat held for ~20 and ~5 minutes respectively (found and fixed only during an unrelated `/sitrep` claims sweep) with no indication anything was wrong, because nothing surfaces a stale-but-recently-created claim the way a 4-hour-TTL prune would catch an actually-abandoned one. The subagent-dispatched lane (apply/KAZI-EXEC.md's "ship it" checklist) has an explicit release-claim step baked into every dispatch prompt; the coordinator's own inline execution of `kind: plan` tasks has no equivalent checklist, so it's easy to walk straight from "PR merged" to the next task without releasing the one just finished.
**Trigger:** The coordinator (not a subagent) claiming and executing a `kind: plan` task end-to-end in the same session -- especially back-to-back planning tasks (T7.0 -> T8.0 -> T9.0), where momentum carries straight into the next claim without a release step in between.

## L-0007: Backgrounding a kazi `apply --parallel` launch with a trailing `&` *and* the Bash tool's `run_in_background: true` orphans the real process, which self-halts

**Tags:** #kazi #parallel-agents #background-process #gotcha
**Date:** 2026-08-30
**Repo:** sirerun/saka

**Rule:** When launching `kazi apply <ref> --workspace <worktree> --parallel ... --json` from inside a kazi-lane subagent, pass the raw command straight to the Bash tool's `run_in_background: true` and nothing else -- never add a trailing `&` (or otherwise self-background the shell command) on top of it. Double-backgrounding forks the real `kazi apply` process out from under the tool's own process tracking; the tool then only observes the tiny wrapper shell, which exits almost immediately after printing a PID line, while the actual `kazi apply` process is reparented to init and orphaned.
**Why:** T9.1's kazi-lane subagent launched its `kazi apply --parallel` run with a trailing `&` inside a `run_in_background: true` Bash call. The tool tracked only the wrapper shell (forked and exited in under a second, having printed "launched pid 24026"). The real `kazi apply` process (pid 24026) was orphaned, and kazi's own launcher-liveness reaper (upstream issue #1073) detected its parent launcher was gone and self-halted immediately: "launcher process 1 is gone; halting to reap this run's dispatch tree." No iteration was ever recorded (`kazi status <goal>` showed zero), no predicate was evaluated, and no harness dispatch or tokens were spent -- the run died at startup, before doing any real work. The subagent's own liveness-polling wait loop correctly detected the process had exited, but the completion signal didn't reach it in a timely way, so it (and the coordinator, watching from outside) both believed a convergence run was still in progress for roughly an hour with nothing actually running. The worktree was untouched (still clean at `origin/main`), so no work was lost -- but a full wall-clock hour was.
**Trigger:** Any kazi-lane dispatch prompt or subagent-authored script that combines a manual `&` background operator with the Bash tool's own `run_in_background` flag for a `kazi apply --parallel` invocation. Recognize the symptom from the *outside* too: `kazi status <goal>` (or a plain `ps aux | grep kazi` / `claude` harness search) showing no live process for a goal that a subagent still reports as "converging" is the tell -- don't take "still waiting" reports at face value past the point where an external process check is cheap to run.

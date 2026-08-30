# Handover -- 2026-08-30T07:35Z, session saka [3bc5e4]

## TL;DR
This session ran `/apply --pool` on `docs/plan.md`: shipped E1-E5 (frontier
hardening, 23 tasks) including a real verified v0.1.0 release, then planned
all three founder-greenlit Future epics (E7 news, E8 images, E9 streaming) to
executable fidelity. It stopped mid-execution of E7's first task (T7.1),
still converging in a background agent with no PR yet. **Next action:** check
on task-T7-1's progress (or re-dispatch T7.1 if it died), then continue
`/apply --pool` on the now-fully-executable E7/E8/E9 task lists.

## Done & VERIFIED
- **E1-E5 (23 tasks), COMPLETE.** Coverage gate 40%->55% (real gate, not
  fabricated), `internal/htmd` dedup, provider plugin registry, working
  install/Homebrew path, Helm chart with a live-cluster CI job. Full PR
  history in `docs/roadmap.md`'s Shipped section (PRs #10-#54, verified via
  `gh pr view` at merge time, not agent self-report).
- **Real v0.1.0 release, verified live**: `brew tap sirerun/tap` + `brew
  install sirerun/tap/saka` actually installed the binary (6 files, 7.9MB)
  and `saka search` returned live results, observed directly on this
  machine (not just CI green).
- **T7.0 (E7 planning), PR #55, merged, CI green.** `docs/plans/E7-news-vertical.md`
  now has 6 executable tasks (T7.1-T7.6). Source: GDELT DOC 2.0 API
  (verified free, no key, via web research this session). Mechanism:
  `docs/adr/003-search-verticals.md`.
- **T8.0 (E8 planning), PR #56, merged, CI green.** `docs/plans/E8-images-vertical.md`
  now has 5 executable tasks (T8.1-T8.5). Source: SearXNG's
  `categories=images` mode + new `types.Result` fields (ThumbnailURL/
  Width/Height). Recorded as an addendum to ADR 003, not a new ADR.
- **T9.0 (E9 planning), PR #57, merged, CI green.** `docs/plans/E9-streaming-search.md`
  now has 5 executable tasks (T9.1-T9.5). Decision: stream only once a
  provider succeeds. New `docs/adr/004-search-streaming.md` (adds
  `SearchStream` to `types.Searcher`; confirmed via grep that `saka.Engine`
  is the only concrete implementer, so this is a single-site change).
- **L-0006 lore entry, PR #58, merged, CI green.** `docs/lore.md`: a
  coordinator-executed `kind: plan` task must release its own claim
  right after merge -- nothing else does it. (See Landmines below.)
- **Claim hygiene fixed**: released the T8.0 and T9.0 claim locks, which had
  been left stale-held after their PRs merged (found during a `/sitrep`
  claims sweep, ~20min and ~5min stale respectively -- caught early, no
  actual harm, but see L-0006).
- **Stray worktree/branch cleanup**: removed 5 confirmed-merged worktrees
  and their local branches (content-verified via `git cherry main <branch>`,
  since this repo rebase-merges and SHA-ancestry alone (`git branch
  --merged`) does not detect a rebase-merged branch as merged).

## Done but UNVERIFIED
- Nothing outstanding is "done but unverified" -- everything reported Done
  above was independently checked (PR view, CI status, or a live command),
  not taken on an agent's word.

## In flight
- **T7.1** (adding a `Vertical string` field to `types.Query` and
  `types.ProviderConfig`, the first building block of the news/images
  search-vertical mechanism) -- claimed, dispatched to background agent
  `task-T7-1`, **no PR yet, no code committed as of this handover**.
  - Claim: `refs/claims/T7.1`, sha `8812eccb9e692afb53b8f75b49e784098c9f8e0c`,
    claimed 2026-08-30T07:14:32Z. Still held -- correct, the agent is still
    working. Release it yourself only if you determine the agent is dead
    (see Running processes below).
  - Worktree: `/Users/dndungu/Code/sirerun/saka-worktrees/t7-1-query-vertical-field`,
    branch `task/t7-1-query-vertical-field-work`. **Gotcha**: the dispatch
    prompt told the agent to name its branch
    `task/t7-1-query-vertical-field` (no `-work` suffix) -- the agent
    actually created `-work` instead. Not a bug, just note the real branch
    name has the suffix when you go looking for it.
  - As of this handover the worktree's branch is still at the claim commit
    only (`8812ecc`) -- zero code changes committed. `kazi status --json`
    shows 0 live runs, meaning the agent has not yet reached (or is
    between) `kazi apply` converge steps.

## Blocked
- **None needing David.** E6 (Usage Persistence & Billing) stays parked
  exactly as he decided 2026-08-29 -- do not run `T6.0` without a fresh
  founder decision on a store/Stripe backend.
- **Structural, not a problem**: E8's 5 tasks (T8.1-T8.5) each `deps:` on
  their matching T7.x task, so none are claimable until E7 lands the piece
  they need. This is by design (see docs/plans/E8-images-vertical.md).

## Running processes left alive
- **`task-T7-1`** [agent id `f2de99`, background] -- executing T7.1 via the
  kazi lane (apply/KAZI-EXEC.md). Status per `ListAgents` was "idle"
  (between turns) as of this handover, not "finished." To check on it:
  message it directly (`SendMessage({to: "task-T7-1", ...})`) or inspect
  its worktree/branch above for new commits. If it appears dead (no
  progress after checking in, no active kazi run, claim looks abandoned),
  release `refs/claims/T7.1` with the sha above before re-dispatching --
  do not just re-claim over a live claim.
- No other background agents or kazi converges are running. `kazi status
  --json` returned 0 live runs at last check.

## Landmines & context
- **Branch-name drift in dispatch prompts**: when writing a task dispatch
  prompt, the agent may not use the exact branch name you specify (see
  T7.1's `-work` suffix above). Don't assume the literal name given in the
  prompt is what's on disk -- check `git worktree list` / `git branch -a`.
- **Coordinator-executed `kind: plan` tasks must release their own claim
  immediately after merge** -- see `docs/lore.md` L-0006. There is no
  checklist step forcing this the way there is for subagent-dispatched
  tasks; it's easy to walk straight from "PR merged" into claiming the
  next task without releasing the one just finished. Check
  `git for-each-ref refs/remotes/origin-claims/` (after a
  `git fetch origin "+refs/claims/*:refs/remotes/origin-claims/*"`) any
  time you're unsure what's actually held.
- **This repo rebase-merges PRs.** `git branch --merged`/`git branch -d`
  will NOT recognize a local capture branch as merged even after its PR
  lands, because rebase-merge mints a new commit SHA on `main`. Use
  `git cherry main <branch>` (patch-id comparison) to check content-
  equality before force-deleting a local branch, not SHA ancestry.
- **kazi `--parallel` scheduler landmine** (recurred repeatedly earlier
  this session, see `docs/lore.md` L-0001, L-0002, L-0005 and
  `apply/KAZI-EXEC.md`): the real landing commit for a converged goal can
  end up on a scheduler-owned `kazi-partition/p-<hash>` branch or a
  `refs/kazi/salvage/...` ref instead of the task's intended branch. Find
  it with `git branch -a | grep kazi-partition` / `git for-each-ref
  refs/kazi/salvage`, diff against the predicate brief, cherry-pick, and
  independently re-verify before trusting it.
- **The coordinator git-hygiene pattern used throughout this session** for
  any coordinator-authored change to a shared doc (`docs/plan.md`,
  `docs/roadmap.md`, `docs/lore.md`): edit -> `git add`/`git commit` on
  `main` -> `git branch <capture-name>` -> `git reset --soft origin/main`
  -> clean `main`'s working tree (`git restore --staged` /
  `git checkout HEAD --`) -> push the capture branch -> `gh pr create` ->
  watch CI -> `gh pr merge --rebase --delete-branch` -> sync `main` via
  `git fetch` + `git merge --ff-only`. Never commit or reset directly
  against the shared primary checkout without capturing first.
- **`.claude/scratch/usecases-manifest.json`** is gitignored (by design,
  per this repo's CLAUDE.md scratch convention) -- it was updated this
  session (UC-022 for images, UC-023 for streaming, both flipped from
  placeholder to real interface descriptions) but those edits are
  local-only and will not appear on any branch. A fresh session/machine
  starts with the manifest in whatever state its own local copy has.
- **`docs/updates.md` is retired** (apply/SKILL.md) -- do not write it,
  despite an old peer-session label in `ListAgents` referencing it.

## How to resume
1. `git fetch origin` from the main checkout (`/Users/dndungu/Code/sirerun/saka`,
   already on `main`, clean, at `d9bbbf3` as of this handover).
2. Read this file, then `docs/roadmap.md` and `docs/plan.md` for full
   context (this file is the delta since the last full read; the roadmap
   and plan are the standing source of truth).
3. Check `task-T7-1`'s status (see "Running processes" above) before doing
   anything else -- don't duplicate work it may have already finished.
4. Resume `/apply --pool`: claimable now (no deps) are T7.1 (if you
   release/re-claim it), T9.1 (`SearchStream` on `types.Searcher`/
   `Engine`, no deps). E8's tasks unlock as their matching T7.x lands.
5. No checkpoint file exists for this session beyond this handover doc and
   `.claude/scratch/handover-inventory.md` (the raw Phase-0 inventory this
   doc was built from, safe to delete once you've read this file).

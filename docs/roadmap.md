# Saka Roadmap

Living roadmap. Update on every merge, lane claim/finish, new blocker,
decision, and at session end.

## Shipped

- v1 production-ready core: circular-import fix (`types/` leaf package),
  REST `/v1/search`+`/v1/fetch`+`/v1/stream`, MCP over stdio, signed
  Ed25519 API keys, tiered rate limiting, usage endpoint. 2026-08-28.
  (commit 1608759)
- `CLAUDE.md` onboarding doc for future Claude Code sessions in this
  repo. 2026-08-29. (direct commit ab76b07)
- CI health: fixed 22 pre-existing golangci-lint findings and the
  `deploy/k8s/apply.sh` path bug causing every `k8s-smoke` failure.
  2026-08-29. (PR #10)
- Pinned `golangci-lint` version instead of tracking `latest`. 2026-08-29.
  (PR #11)
- Full Dependabot backlog cleared: 9 PRs merged, Go floor raised to 1.27
  across `go.mod`/`go.work`/`Dockerfile`/CI, `golangci-lint` re-pinned to
  a Go-1.27-compatible release (v2.13.2). 2026-08-29. (PR #2-#9, #12)

**E1 -- Coverage & Test Debt Paydown -- COMPLETE (T1.1-T1.5).**
docs/plans/E1-coverage-test-debt.md. 2026-08-29.
- T1.1 SPEC.md coverage claim corrected to the real 40% floor. (PR #13)
- T1.2 fetch/ tests, 34.0% -> 87.8%; found and fixed a real concurrency
  bug in `ExtractStream` (both `doneCh`/`errCh` were unconditionally
  closed though each is written at most once, so callers could get a
  spurious `nil` error instead of the page on ~50% of successful calls).
  (PR #23, checkbox PR #41)
- T1.3 types/ tests, 36.4% -> 100.0%. (PR #21, checkbox PR #22)
- T1.4 root saka.go tests, 30.3% -> 95.7%. (PR #17)
- T1.5 CI coverage gate raised 40% -> 55%; SPEC.md re-synced; verified
  live on main. (PR #42, checkbox PR #43)

**E2 -- internal/htmd Dedup -- COMPLETE (T2.1-T2.4).**
docs/plans/E2-htmd-dedup.md. 2026-08-29.
- T2.1 Extracted internal/htmd from duplicated provider helpers. Follow-up
  found, not tracked: a third UA-rotation duplicate (`pickUA`) lives in
  fetch/fetch.go, outside scope. (PR #18, checkbox PR #19)
- T2.2 provider/duckduckgo migrated to internal/htmd. (PR #27)
- T2.3 provider/startpage migrated to internal/htmd (preserved its
  substring-match class semantics deliberately). (PR #28, checkbox PR #41)
- T2.4 internal/htmd fixture tests from real provider markup, 90.9% ->
  95.5%. (PR #34)

**E3 -- Provider Plugin Registry -- COMPLETE (T3.1-T3.6).**
docs/plans/E3-provider-registry.md. 2026-08-29.
- T3.1 ADR 001 Decision: init()-side-effect registration, canonical impl
  in `types` (avoids a saka<->provider import cycle). (PR #30, checkbox PR #31)
- T3.2 types/registry.go: thread-safe Register/Lookup/Registered,
  error-not-panic. (PR #32)
- T3.3 All 3 built-in providers self-register via init(). (PR #35,
  checkbox PR #36)
- T3.4 saka.New/Config.Validate resolve via the registry instead of a
  hardcoded switch; kept searxng's URL-shape check separate from the
  registry identity check. (PR #37)
- T3.5 Registry unit tests, incl. TestRegistryConcurrentAccess. (PR #39)
- T3.6 docs/SPEC.md documents the Register convention. (PR #38, checkbox PR #40)

**E5 -- Helm Chart Deploy -- COMPLETE (T5.1-T5.4).**
docs/plans/E5-helm-chart.md. 2026-08-29.
- T5.1 Helm chart scaffolded at deploy/helm/saka. (PR #16)
- T5.2 values.yaml parameterized (image/resources/replicas flattened to
  top-level). (PR #26)
- T5.3 CI `helm-smoke` job on a live kind cluster -- first attempt failed
  (`--create-namespace` collided with the chart's own templated
  Namespace resource), fixed and verified green on retry. (PR #46,
  checkbox PR #47)
- T5.4 README documents the Helm install path alongside kubectl apply.
  (PR #44, checkbox PR #45)

**E4 -- Install & Release Distribution -- partial.**
docs/plans/E4-install-distribution.md.
- T4.1 David decided: point install.sh/README at GitHub releases, no
  domain purchase. 2026-08-29.
- T4.3 install.sh/README repointed accordingly. (PR #15)
- T4.2, T4.4 -- see Blocked.

**Coordinator findings this session** (docs/lore.md, all 2026-08-29):
- L-0001: kazi-lane subagents must resolve primary-checkout git issues
  via a disposable detached worktree, never by mutating the shared
  primary checkout. (PR #14)
- L-0002: kazi grinds ignore prose "do not touch" scope restrictions --
  encode scope as a guard predicate, not prose. (PR #24)
- L-0003: this machine's global `golangci-lint` (v2.11.3) panics under
  Go 1.27 -- hit independently 3x this session; pin v2.13.2 explicitly.
  (PR #25)
- L-0004: `git diff <ref> -- <file>` puts `<ref>` on the `-` side and the
  working tree on the `+` side -- a coordinator near-miss from misreading
  this overwrote correct content with stale content (no data lost, a
  sibling worktree held an independent copy). (PR #29)
- L-0005 (T3.5-authored): kazi `match_regex` predicates anchored to
  `go test -v` PASS lines must tolerate subtest indentation and the
  trailing duration suffix, or they silently never match.
- Two plan-integrity gaps found and fixed mid-session: T3.3's original
  acc line described T3.4's outcome (fixed pre-dispatch, PR #33); T1.2
  and T2.3 had code merged but their mark-done step silently skipped
  (found and fixed hours later via a full-plan sweep, PR #41).
- A stray commit briefly diverged the primary checkout's local main
  (redundant content, recovered via stash + hard-reset, no data lost).
- End-of-session cleanup: released 9 stale claim refs and removed 10
  merged worktrees/branches that agents had reported as cleaned up but
  weren't.

## In progress

- None. The claimable pool is empty.

## In flight (PRs open)

- None.

## Planned

See `docs/plan.md` for full detail; `acc:` lines are kazi predicates,
derived just-in-time by `/apply`.

- E6 Usage Persistence & Billing, E7 News Vertical, E8 Images Vertical,
  E9 Streaming Search Results -- each an outline-fidelity epic with one
  `T*.0 PLAN` task, `deps: [T1.5, T2.4, T3.6, T4.4, T5.4]`. E1/E2/E3/E5's
  deps are satisfied; **all four remain blocked on T4.4**, which is
  itself blocked on T4.2 (see Blocked). Nothing here is claimable until
  David unblocks E4.

## Blocked

- **T4.2** (create the `sirerun/homebrew-tap` GitHub repo -- corrected
  name; `.goreleaser.yaml`'s `brews:` section already targets
  `homebrew-tap`, not `tap` -- with a `Formula/saka.rb` scaffold) is
  `kind: human`, Owner: David. Real external infrastructure (a new
  GitHub repo), not code -- the pool cannot do this. Blink MCP is not
  connected in this session, so it could not be auto-scheduled onto
  David's task queue.
- **T4.4** (cut a test release, verify `brew install sirerun/tap/saka`)
  depends on T4.2 and T4.3 (T4.3 done). Stays blocked until T4.2 lands.
  Note for whoever picks this up: T4.3's PR flagged that
  `.goreleaser.yaml`'s archive `name_template` is version-less but
  `install.sh`'s download step expects a version-qualified filename --
  these won't match once a real release is cut; needs reconciling as
  part of T4.4.
- **E6.0/E7.0/E8.0/E9.0** (the four Future-roadmap planning tasks) --
  transitively blocked on T4.4 above via their shared `deps:` list.

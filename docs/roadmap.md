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

**E4 -- Install & Release Distribution -- COMPLETE (T4.1-T4.4).**
docs/plans/E4-install-distribution.md. 2026-08-29.
- T4.1 David decided: point install.sh/README at GitHub releases, no
  domain purchase.
- T4.2 no action needed -- `sirerun/homebrew-tap` already existed as a
  shared tap for other sirerun tools (serenity, mint, gist). Added the
  org secret `HOMEBREW_TAP_GITHUB_TOKEN` to `saka`'s selected-repos list
  (it existed but `saka` wasn't yet granted access).
- T4.3 install.sh/README repointed accordingly. (PR #15)
- T4.4 cut the real v0.1.0 release. Found and fixed SIX previously-
  uncaught bugs in a release pipeline that had never run before (this
  repo had zero prior tags): (1) `ci.yml` missing `workflow_call:`
  trigger, needed by `release.yml`'s reusable-workflow call (PR #50);
  (2) `release.yml`'s `setup-go` pinned to Go 1.22 vs `go.mod`'s
  `>=1.25` requirement (PR #51); (3) `release.yml` read the brew token
  from `secrets.HOMEBREW_TOKEN`, a secret that doesn't exist, instead of
  `secrets.HOMEBREW_TAP_GITHUB_TOKEN` (PR #51); (4) `.goreleaser.yaml`'s
  `brews.repository` never referenced the token via `token:`, so it
  would have silently fallen back to the repo-scoped `GITHUB_TOKEN`
  (PR #51, plus `version: 2` added to silence a schema warning); (5) the
  project claimed Apache-2.0 everywhere but never shipped a `LICENSE`
  file -- added a verbatim, byte-diffed copy from apache.org (PR #52);
  (6) the first successful run published the formula to homebrew-tap's
  repo root instead of `Formula/saka.rb` (no `directory:` set) --
  confirmed broken with a real failed `brew install`, fixed with
  `directory: Formula` (PR #53), stray file deleted from homebrew-tap
  directly. Real end-to-end verification: `brew untap` / `brew tap
  sirerun/tap` / `brew install sirerun/tap/saka` on this machine
  actually installed the binary (6 files, 7.9MB) and `saka search`
  returned live results. (PR #15, checkbox PR #54 for T4.2/T4.4)

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

**Founder decisions, E1-E5 complete (all 2026-08-29):** David greenlit
E7 (News), E8 (Images), and E9 (Streaming) to proceed; explicitly left
E6 (Billing) parked. News source: GDELT DOC 2.0 API (free, no API key --
verified via research; satisfies David's "dedicated API if a free one
exists" condition). E7 expanded to executable fidelity (T7.0, this
session): docs/adr/003-search-verticals.md records the mechanism
(`Query.Vertical` field + one `chain.Chain` per vertical in `Engine`,
generalizing to any future vertical without touching `chain.Chain` or
the provider registry); docs/plans/E7-news-vertical.md has the 6-task
breakdown (T7.1-T7.6).

**T8.0 (2026-08-30):** expanded E8 (Images Vertical) to executable
fidelity, 5 tasks (T8.1-T8.5), docs/plans/E8-images-vertical.md. Source
per David's decision: SearXNG's `categories=images` mode against the
existing self-hosted SearXNG instance (not a dedicated image API), plus
extending `types.Result` with `ThumbnailURL`/`Width`/`Height`. Confirms
ADR 003's "verticals compose" prediction -- no new ADR, `chain.Chain`,
or registry change; recorded as an addendum to ADR 003 instead of a new
ADR since it's new information on an already-covered decision. New
provider: `provider/searxng`'s `images.go`, self-registering as
`"searxng-images"` (not reusing the `"searxng"` name, so a `saka.json`
entry stays unambiguous about which vertical it belongs to). Each E8
task `deps:` on its matching T7.x task, so the pool sequences E8 behind
E7 without a manual wave gate.

**E8 -- Images Vertical -- IN PROGRESS (T8.1/5).**
docs/plans/E8-images-vertical.md.
- T8.1 `types.Result` gains three optional, `omitempty` fields --
  `ThumbnailURL string`, `Width int`, `Height int` -- for a later images
  provider to populate. Zero-value fields are omitted from JSON, so
  general web providers and existing callers are unaffected. Deviation:
  the kazi proposal (`prop-t8-1-result-thumbnail-fields`) was drafted and
  approved, and its t0 `--check` correctly showed all 4 capability
  predicates red / all guards green, but it was never actually dispatched
  via `kazi apply --parallel` -- an external check of `kazi status
  t8-1-result-thumbnail-fields` (a proposal with no run yet) was
  misread as an L-0007 orphaning symptom, and the lane was abandoned in
  favor of hand-implementation before a real convergence attempt was
  made. Hand-implemented instead (`go vet`/`go build`/`go test -race
  -cover` all green, `types` package stays at 100% coverage, plus two new
  tests -- zero-value omission and JSON round-trip). 2026-08-30. (PR #72)

**E7 -- News Vertical -- IN PROGRESS (T7.1, T7.3/6).**
docs/plans/E7-news-vertical.md.
- T7.1 `Vertical string` added to `types.Query`/`types.ProviderConfig`;
  `WithDefaults()` deliberately left untouched. Ran via the kazi lane
  (claude-sonnet-5); the run's own `landed` predicate false-negatived
  because `--parallel` lands work on a scheduler-owned
  `kazi-partition/p-<hash>` branch, not `task/<goal-id>` -- verified this
  independently (every other predicate converged green), cherry-picked
  the real commit onto the task branch, added missing test coverage, and
  re-ran the full validation ladder before shipping. 2026-08-30. (PR #62,
  checkbox PR #63)
- T7.3 `provider/gdelt` -- new `types.Provider` calling GDELT's DOC 2.0
  JSON API (no API key), self-registered as `"gdelt"`, fixed to vertical
  `"news"` (doc comment warns against wiring it into the general chain,
  recommends `rps: 1`). Ran via the kazi lane (claude-sonnet-5);
  converged cleanly on a scheduler-owned `kazi-partition/p-<hash>`
  branch (expected under `--parallel`, not a false negative this time).
  Independently verified before shipping anyway: rebased + cherry-picked
  the real commit onto `task/t7-3-provider-gdelt`, re-ran the full
  validation ladder (`go build`, `go vet`, `go test -race -v` — 6/6
  pass, `gofmt -l`, the CI-pinned `golangci-lint`, and the full
  race+cover suite) before opening the PR. 2026-08-30. (PR #66, checkbox
  PR #67)

**T9.0 (2026-08-30):** expanded E9 (Streaming Search Results) to
executable fidelity, 5 tasks (T9.1-T9.5), docs/plans/E9-streaming-search.md.
Per David's decision: stream only once a provider succeeds, no partial-
per-provider streaming. New ADR (docs/adr/004-search-streaming.md, not
an ADR 003 addendum -- this is a distinct decision): `types.Searcher`
gains `SearchStream(ctx, Query) (<-chan Result, <-chan *Results, <-chan
error)`, mirroring the existing `Fetch`/`FetchStream` channel-shape
precedent; `Engine.SearchStream` calls the existing synchronous `Search`
internally and streams its results, so `chain.Chain`'s fallback logic is
untouched. `saka.Engine` is confirmed the only concrete `types.Searcher`
implementer in this repo, so the interface addition is single-site. MCP
streaming is explicitly out of scope this pass (stdio JSON-RPC transport
is request/response, not SSE). T9.4 proves streaming composes with
E7/E8's vertical mechanism. All three founder-greenlit epics (E7/E8/E9)
are now at `fidelity: executable`; only E6 (parked) remains outline.

## In progress

- T9.1 (SearchStream on types.Searcher/Engine): claimed and dispatched to
  a kazi-lane agent, 2026-08-30T17:2xZ (first attempt, no prior claim).
  Worktree: `saka-worktrees/t9-1-searchstream`, branch
  `task/t9-1-searchstream`. No PR yet. T7.1 (the other concurrent editor
  of `types/types.go`) has since merged (PR #62) -- T9.1's `SearchStream`
  addition is in an unrelated section of the file, but rebase onto main
  if a conflict shows up.

## In flight (PRs open)

- None yet from T9.1's dispatch -- check back once its agent reports.

## Planned

See `docs/plan.md` for full detail; `acc:` lines are kazi predicates,
derived just-in-time by `/apply`.

- E7 News Vertical (6 tasks, fidelity: executable) --
  docs/plans/E7-news-vertical.md. T7.1 and T7.3 shipped (see Shipped);
  T7.2 (deps: [T7.1]) is claimable now; T7.4/T7.5 (deps: [T7.2, T7.3])
  are claimable once T7.2 lands; T7.6 (deps: [T7.4, T7.5]) after those.
- E8 Images Vertical (5 tasks, fidelity: executable) --
  docs/plans/E8-images-vertical.md. T8.1 shipped (see Shipped). T8.2
  (deps: [T7.2, T8.1]) is claimable now -- `feat(saka): route Search
  through per-vertical provider chains` (commit 767b6f6) landed T7.2 on
  main, and T8.1 is merged. T8.3/T8.4 (deps: [T7.4, T7.5, T8.2] /
  [T7.5, T8.2]) wait on T8.2 plus their T7.x counterparts; T8.5 (deps:
  [T8.3, T8.4]) after those.
- E9 Streaming Search Results (5 tasks, fidelity: executable) --
  docs/plans/E9-streaming-search.md. T9.1 (SearchStream on
  types.Searcher/Engine) has no deps, claimable now.
- E6 Usage Persistence & Billing -- parked, not triggered. Needs a fresh
  founder decision (a store/Stripe backend choice) before T6.0 can run.

## Blocked

- **E6** is parked (David's explicit choice, 2026-08-29) -- not blocked
  on anything technical, just not triggered. T6.0 should not run without
  a fresh founder decision on the billing store/Stripe backend.

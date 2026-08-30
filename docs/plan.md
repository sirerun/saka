# Saka -- Pending Work Plan

## 1. Context

**Problem statement.** Saka reached "v1 production-ready" on 2026-08-28
(NOTES.md, commit 1608759): the free search/fetch/serve/MCP/tool-calling
path is wired end-to-end, CI is green, and the Dependabot backlog is
fully cleared (9 PRs merged 2026-08-29, main now on Go 1.27 throughout).
What remains is everything `docs/NOTES.md` calls "still scaffolding" and
everything `docs/SPEC.md` section 14 calls "Future": billing that
survives a restart, two new search verticals, streaming search results, a
provider extensibility story, real install infrastructure, a Helm chart,
and closing the gap between SPEC.md's claimed 70% coverage floor and the
40% CI actually enforces. This plan scopes that remaining work.

**Objectives.**
- Land the near-term, unblocked hardening and extensibility work
  (frontier epics E1-E5) to full completion.
- Scope, but do not yet decompose, the four larger Future-roadmap items
  (E6-E9) -- they need upfront decisions (a store choice, a Stripe
  account, a vertical data-source choice, a streaming protocol choice)
  that are not yet made, so decomposing them now would be fake precision.
- Keep `docs/SPEC.md` and `docs/adr/` as the accurate record of what
  saka's external contract and architecture actually are, closing the two
  discrepancies found during planning (coverage claim; "register in
  saka.New" instructions once the registry lands).

**Non-goals.** This plan does not cover the v1.0/v1.1/v1.2 work already
shipped (see docs/SPEC.md section 14's roadmap table and UC-001 through
UC-012 in the use case manifest, all WIRED). It does not pick a Stripe
plan or a news/image data source -- those are E6/E7/E8's planning tasks'
job, once triggered.

**Constraints and assumptions.**
- Go 1.27 toolchain, `golang.org/x/net` as the only external dependency
  (unchanged by this plan; no new dependencies are in scope except where
  an epic's own planning task decides otherwise, e.g. E6's store choice).
- Single external CI runner (GitHub Actions, GitHub-hosted); no staging
  environment for this repo -- "deployed" means merged to main and (for
  anything with a live surface) verified against the real k8s-smoke path
  or a real release tag.
- `kazi` is on PATH in this environment; every engineering task below
  carries an `acc:` line for kazi's just-in-time predicate lane
  (apply/KAZI-EXEC.md, ADR 015).
- This is a solo-maintainer OSS repo (Apache-2.0, `sirerun/saka`) with no
  paying customers yet -- E6 (billing) and E4 (distribution) tasks
  involving real-world commitments (Stripe account, domain purchase, tap
  repo) are marked `kind: human`.

**Success metrics.**
- E1-E5: all tasks `[x]`, CI green on main, the two new ADRs' decisions
  reflected in `docs/SPEC.md`.
- E6-E9: each epic's single planning task expands it to `fidelity:
  executable` with >=5 tasks and full acceptance criteria -- measured by
  `parse_plan.py` against each epic file.

## 2. Discovery Summary

No `.code-review-graph/graph.db` present -- manual scan used. Sources:
`README.md`, `NOTES.md`, `docs/VISION.md`, `docs/SPEC.md`, and direct
reading of `saka.go`, `types/types.go`, `chain/chain.go`, every file
under `server/`, `fetch/`, `provider/*`, `cli/saka/`, `tools/`, plus this
session's own CI-health work (all 9 Dependabot PRs merged, `main` clean).

**Use cases discovered: 20 total** (see
`.claude/scratch/usecases-manifest.json`). 12 WIRED (UC-001 through
UC-012 -- library search/fetch/stream, CLI search/fetch/serve, REST, MCP,
signed-key auth, usage endpoint, k8s deploy), all P0-P1, all with test
coverage except the two bare CLI commands (UC-004, UC-005, which are thin
wrappers over already-tested library calls). 8 originally PLANNED
(UC-019 through UC-026): usage persistence, Stripe billing, news vertical,
images vertical, streaming search, provider registry, install
distribution, Helm chart. **Update 2026-08-29: UC-024 (registry), UC-025
(install), UC-026 (Helm) are now WIRED (E3/E4/E5 shipped). UC-021 (news)
is in progress (E7 now executable). UC-019/020 (E6, billing) and UC-022/023
(E8/E9) remain MISSING** -- E8/E9 triggered but not yet built, E6 parked.

**Gaps found beyond the use case manifest:**
- `docs/SPEC.md` section 11 claims a 70% coverage gate; `ci.yml` enforces
  40%. Actual measured coverage (2026-08-29): 51.8% total, but fetch
  (34.0%), types (36.4%), and root saka (30.3%) are the weakest packages
  -- see docs/adr/002-coverage-gate-honesty.md.
- `provider/duckduckgo` and `provider/startpage` duplicate HTML-scraping
  helpers (flagged in NOTES.md, never resolved).
- Adding a provider requires editing two switches in the root `saka`
  package (`saka.go`'s `New` and `Config.Validate`) -- no plugin
  convention exists despite SPEC.md section 14 listing one as Future --
  see docs/adr/001-provider-plugin-registry.md.
- `install.sh`/README point at `getsaka.dev`, a placeholder domain; the
  GoReleaser config's Homebrew tap (`sirerun/tap`) may not exist yet
  (NOTES.md).
- `deploy/k8s/` is plain manifests; no Helm chart (NOTES.md).

## 3. Scope and Deliverables

**In scope:**
- E1 Coverage & Test Debt Paydown -- closes the SPEC.md/ci.yml coverage
  discrepancy honestly.
- E2 internal/htmd Dedup -- removes duplicated provider helpers, and is a
  prerequisite for E3 (a clean shared base to build the registry on).
- E3 Provider Plugin Registry -- the extensibility story SPEC.md already
  promises.
- E4 Install & Release Distribution -- makes the documented install paths
  actually work.
- E5 Helm Chart Deploy -- parameterized alternative to the plain-manifest
  k8s deploy.
- E6-E9 (outline only this pass) -- usage persistence + billing, news
  vertical, images vertical, streaming search results.

**Out of scope:** rewriting the already-WIRED v1.0-v1.2 surface;
picking a Stripe plan or a news/image data source (E6/E7/E8's own
planning tasks); a staging environment (none exists for this repo, see
Constraints).

**Deliverables:**

| ID | Description | Owner | Acceptance |
|---|---|---|---|
| D1 | CI coverage gate raised to 55%, SPEC.md corrected | this session | **SHIPPED** 2026-08-29 |
| D2 | internal/htmd package, both providers migrated | this session | **SHIPPED** 2026-08-29 |
| D3 | Provider registry, three built-ins self-registered | this session | **SHIPPED** 2026-08-29 |
| D4 | Working install.sh/brew install path | this session | **SHIPPED** 2026-08-29 -- real v0.1.0 release, `brew install sirerun/tap/saka` verified live |
| D5 | Helm chart at deploy/helm/saka | this session | **SHIPPED** 2026-08-29 -- helm-smoke CI job green |
| D7 | E7 expanded to executable fidelity | this session | **SHIPPED** 2026-08-29 -- see docs/plans/E7-news-vertical.md |
| D8-D9 | E8/E9 expanded to executable fidelity | pool | Each epic's `T*.0` planning task `[x]` |
| D6 | E6 expanded to executable fidelity | -- | Parked; not triggered (needs a founder decision on a store/Stripe backend) |

## 4. Checkable Work Breakdown

Split layout. Each epic's full task list, acceptance criteria, `acc:`
predicates, and dependencies live in its own file under `docs/plans/`.
Counts are `(done/total)`, regenerated on every `/plan`/`/tidy` run.

**Complete (kept for reference; no longer active work):**

### E1 -- Coverage & Test Debt Paydown  -> docs/plans/E1-coverage-test-debt.md  (5/5) COMPLETE
### E2 -- internal/htmd Dedup  -> docs/plans/E2-htmd-dedup.md  (4/4) COMPLETE
### E3 -- Provider Plugin Registry  -> docs/plans/E3-provider-registry.md  (6/6) COMPLETE
### E4 -- Install & Release Distribution  -> docs/plans/E4-install-distribution.md  (4/4) COMPLETE
### E5 -- Helm Chart Deploy  -> docs/plans/E5-helm-chart.md  (4/4) COMPLETE

**Active (founder-triggered 2026-08-29 -- David greenlit E7/E8/E9, explicitly left E6 parked):**

### E7 -- News Vertical  -> docs/plans/E7-news-vertical.md  (0/6)  fidelity: executable
### E8 -- Images Vertical  -> docs/plans/E8-images-vertical.md  (0/1)  fidelity: outline  (triggered; T8.0 not yet run)
### E9 -- Streaming Search Results  -> docs/plans/E9-streaming-search.md  (0/1)  fidelity: outline  (triggered; T9.0 not yet run)

**Parked (not triggered; needs a founder decision before its planning task can run):**

### E6 -- Usage Persistence & Billing  -> docs/plans/E6-usage-persistence-billing.md  (0/1)  fidelity: outline

## 5. Parallel Work

**Tracks:**

| Track | Epics | Touches |
|---|---|---|
| A: Test debt | E1 | fetch/, types/, saka.go, ci.yml |
| B: Provider refactor | E2 then E3 (sequenced -- both restructure the same provider packages) | provider/*, new internal/htmd, new registry |
| C: Distribution | E4 | install.sh, README.md, .goreleaser.yaml, external sirerun/tap repo |
| D: Deploy | E5 | new deploy/helm/, ci.yml |
| E: Future planning | E7, E8, E9 triggered 2026-08-29 (E6 parked, needs a founder decision) | docs/plans/E6-E9.md |

Tracks A, C, D are fully independent of B and of each other. Track B's
two epics are sequenced (E3 depends on E2's `T2.1`) because both
restructure `provider/duckduckgo` and `provider/startpage`; landing the
dedup first gives the registry refactor a clean base instead of two
structural changes racing on the same files.

**Waves 1-5 (E1-E5, 23 tasks): DONE.** All shipped 2026-08-29 via
`/apply --pool`. See docs/plans/E1-coverage-test-debt.md through
docs/plans/E5-helm-chart.md for the full per-task record (PR numbers,
what was found, what was fixed).

### Wave 6: Future planning
- [x] T7.0 PLAN: expand E7 -- done this run, see docs/plans/E7-news-vertical.md
- [ ] T8.0 PLAN: expand E8 -- triggered (David greenlit 2026-08-29), not yet run
- [ ] T9.0 PLAN: expand E9 -- triggered (David greenlit 2026-08-29), not yet run
- [ ] T6.0 PLAN: expand E6 -- parked, not triggered (needs a founder decision on a store/Stripe backend)

### Wave 7: E7 execution (6 tasks, claimable now via /apply --pool)
- [ ] T7.1 Query.Vertical/ProviderConfig.Vertical fields
- [ ] T7.2 Engine multi-vertical chain refactor
- [ ] T7.3 provider/gdelt implementation
- [ ] T7.4 REST + CLI wiring
- [ ] T7.5 MCP + tool-schema wiring
- [ ] T7.6 Docs

## 6. Timeline and Milestones

| Milestone | Depends on | Exit criteria |
|---|---|---|
| M1: Test debt closed | T1.1-T1.5 | Coverage gate at 55%, CI green, SPEC.md corrected |
| M2: Providers deduped and pluggable | T2.1-T2.4, T3.1-T3.6 | New provider addable without touching saka.go; docs/adr/001 Decision section filled in |
| M3: Distribution works | T4.1-T4.4 | `brew install sirerun/tap/saka` verified on a clean machine |
| M4: Helm path live | T5.1-T5.4 | `helm install saka deploy/helm/saka` verified via CI |
| M5: Future work scoped | T6.0-T9.0 | All four outline epics at `fidelity: executable` |

## 7. Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|---|---|---|---|---|
| R1 | Registry redesign (E3) changes Config.Validate's error shape, breaking an undocumented caller | Medium | Low | T3.5's tests pin the new error path; SPEC.md section 3.3 updated in the same epic (T3.6) |
| R2 | E4's domain/tap decision (T4.1) stalls on a founder decision that never comes | Medium | Medium | `kind: human`, flagged as FOUNDER DECISION; downstream T4.2-T4.4 are `blocked-by: [T4.1]` implicitly via `deps:`, so they simply wait rather than silently drifting |
| R3 | Raising the coverage gate (T1.5) to 55% before tests actually land, or CI flaking on the new floor | Low | Low | T1.5 explicitly deps on T1.2-T1.4 and requires a verified-green CI run before landing, per docs/adr/002 |
| R4 | E6-E9's planning tasks (T6.0-T9.0) fire before frontier learnings exist, producing shallow re-plans | Low | Low | All four `deps:` on the full frontier-completion set (T1.5, T2.4, T3.6, T4.4, T5.4), not just "now" |

## 8. Operating Procedure

Definition of done (all apply unless a task states a stated exception):
- Automated tests written and green for the changed surface (unit tests
  for E1-E3/E6-E9's Go changes; no UI in this repo, so no browser-test
  requirement applies).
- PR merged to `main` via rebase merge, CI green (matches this session's
  established pattern: open PR, wait for `lint`/`test`/`k8s-smoke`, `gh
  pr merge --rebase`).
- `golangci-lint run` and `go vet ./...` clean before merge (the local
  pre-commit hook already enforces this; CI is the second gate).
- For E4/E5 (install path, Helm chart): the acceptance criterion requires
  a real verification (a cut release + `brew install`, a `helm install`
  against a live kind cluster in CI) -- not just "the file exists".
  Staging does not exist for this repo; "deployed and verified" means the
  real CI k8s-smoke/helm-smoke run or a real GitHub release tag.
- Reported honestly: state what was observed (e.g. "CI run #N passed"),
  not what is expected.
- Never commit files from different directories in the same commit (the
  pre-commit hook lints per-commit; keep commits scoped to one epic's
  concern at a time).
- Small, logical commits; do not let changes pile up across multiple
  tasks into one commit.

## 9. Progress Log

**2026-08-29 -- Change Summary (this run, T7.0):** E1-E5 all shipped via
`/apply --pool` (23 tasks, ~35 PRs, real v0.1.0 release cut with 6
previously-uncaught release-pipeline bugs found and fixed along the way
-- see docs/roadmap.md for the full list). David greenlit E7/E8/E9,
explicitly parked E6. Ran T7.0: expanded E7 to executable fidelity (6
tasks) per David's news-source decision (GDELT DOC 2.0 API, free, no key
-- docs/adr/003-search-verticals.md records the internal mechanism:
`Query.Vertical` + per-vertical `chain.Chain` in `Engine`, generalizing
past news to any future vertical). Trimmed E1-E5 out of the active
breakdown (complete, kept for reference); marked E8/E9 triggered but not
yet expanded (T8.0/T9.0 still to run); E6 stays parked.

**2026-08-29 -- Change Summary (original run):** Initial plan created (no
prior `docs/plan.md` existed). 9 epics, 27 tasks, split layout. 2 new
ADRs (001 provider-plugin-registry, 002 coverage-gate-honesty). Use case
manifest written with 18 use cases (12 WIRED from the already-shipped v1
surface, 6 PLANNED for this plan's scope). `docs/roadmap.md` created with
this plan's epics under Planned.

## 10. Hand-off Notes

- E1-E5 are done. `/apply --pool` on this plan now has E7's 6 tasks as
  its primary claimable work (T7.1 first, no deps; see
  docs/plans/E7-news-vertical.md for the full dependency graph).
- `kazi` is on PATH; every task's `acc:` line is what `/apply`'s kazi
  execution lane will derive a predicate from at dispatch time (never at
  plan time -- see apply/KAZI-EXEC.md).
- T4.2/T4.4 (`kind: human`/founder-decision items) are done -- T4.2
  turned out to need no action (the shared homebrew-tap repo already
  existed); T4.4 cut the real v0.1.0 release.
- E8 and E9 are triggered (David greenlit them 2026-08-29) but not yet
  expanded -- run T8.0 and T9.0 (via `/plan` scoped to each epic, same
  as T7.0 was run) before either has claimable engineering tasks. E6
  stays parked; do not run T6.0 without a fresh founder decision on a
  store/Stripe backend.
- E7's design (docs/adr/003-search-verticals.md) generalizes past news:
  E8 (images) should reuse the exact same `Query.Vertical` +
  per-vertical-chain mechanism with a new `provider/*` implementation
  and `Vertical: "images"`, not a new mechanism. Read ADR 003 before
  planning E8.
- No secrets in this repo's plan; Stripe credentials (if/when E6
  triggers) belong in the operator's own secret store, never in `docs/`.

## 11. Appendix

- `docs/NOTES.md` -- the source-of-truth list this plan's scope was
  derived from ("Still scaffolding" section).
- `docs/SPEC.md` section 14 -- the roadmap table naming E6-E9's Future
  items.
- `docs/VISION.md` -- v1.1/v1.2 principles this plan's E4-E5 continue
  ("One binary, zero ceremony"; "the paid path exists so the free path
  never has to compromise").
- `.claude/scratch/usecases-manifest.json` -- full use case manifest
  backing section 2's counts.
- `docs/adr/001-provider-plugin-registry.md`, `docs/adr/002-coverage-gate-honesty.md`,
  `docs/adr/003-search-verticals.md`.

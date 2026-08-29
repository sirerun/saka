# Saka Roadmap

Living roadmap. Update on every merge, lane claim/finish, new blocker,
decision, and at session end.

## Shipped

- v1 production-ready core: circular-import fix (`types/` leaf package),
  REST `/v1/search`+`/v1/fetch`+`/v1/stream`, MCP over stdio, signed
  Ed25519 API keys, tiered rate limiting, usage endpoint. 2026-08-28.
  (commit 1608759)
- `CLAUDE.md` onboarding doc for future Claude Code sessions in this
  repo. 2026-08-29. (PR #1 equivalent -- direct commit ab76b07)
- CI health: fixed 22 pre-existing golangci-lint findings and the
  `deploy/k8s/apply.sh` path bug causing every `k8s-smoke` failure.
  Owner: this session. 2026-08-29. (PR #10)
- Pinned `golangci-lint` version instead of tracking `latest`. Owner:
  this session. 2026-08-29. (PR #11)
- Full Dependabot backlog cleared: 9 PRs merged (actions/tooling bumps,
  Dockerfile base image, `golang.org/x/net` security bump). Go floor
  raised to 1.27 across `go.mod`, `go.work`, `Dockerfile`, and CI, with
  `golangci-lint` re-pinned to a Go-1.27-compatible release (`v2.13.2`)
  after the pinned `v2.11.3` was found to panic under the new toolchain.
  Owner: this session, ruling from `macbook` (seat). 2026-08-29. (PR #2,
  #3-#9, #12)

## In progress

- None.

## In flight (PRs open)

- None. `gh pr list` on `sirerun/saka` is empty as of 2026-08-29.

## Planned

See `docs/plan.md` for full detail; `acc:` lines are kazi predicates,
derived just-in-time by `/apply`.

- E1 Coverage & Test Debt Paydown (5 tasks) -- raise CI's coverage gate
  from 40% to 55% honestly; correct `docs/SPEC.md`'s 70% claim. Owner:
  TBD. `docs/plans/E1-coverage-test-debt.md`.
- E2 internal/htmd Dedup (4 tasks) -- remove duplicated HTML-scraping
  helpers between `provider/duckduckgo` and `provider/startpage`. Owner:
  TBD. `docs/plans/E2-htmd-dedup.md`.
- E3 Provider Plugin Registry (6 tasks) -- replace the hardcoded provider
  switch in `saka.go` with self-registration (docs/adr/001). Owner: TBD.
  `docs/plans/E3-provider-registry.md`.
- E4 Install & Release Distribution (4 tasks) -- real install domain and
  a working `sirerun/tap` Homebrew repo. Owner: David (T4.1, T4.2 are
  `kind: human`). `docs/plans/E4-install-distribution.md`.
- E5 Helm Chart Deploy (4 tasks) -- parameterized Helm alternative to the
  plain `deploy/k8s/` manifests. Owner: TBD.
  `docs/plans/E5-helm-chart.md`.
- E6 Usage Persistence & Billing (outline, 1 planning task) -- durable
  usage store + Stripe. Triggered once E1-E5 land.
  `docs/plans/E6-usage-persistence-billing.md`.
- E7 News Vertical (outline, 1 planning task). Triggered once E1-E5
  land. `docs/plans/E7-news-vertical.md`.
- E8 Images Vertical (outline, 1 planning task). Triggered once E1-E5
  land. `docs/plans/E8-images-vertical.md`.
- E9 Streaming Search Results (outline, 1 planning task). Triggered once
  E1-E5 land. `docs/plans/E9-streaming-search.md`.

## Blocked

- T4.1 (decide the real install domain) needs a founder decision before
  E4 can proceed past Wave 1. Not yet asked -- surfaced here per the
  living-roadmap convention rather than interrupting mid-session.

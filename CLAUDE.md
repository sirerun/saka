# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`saka` (module `github.com/sirerun/saka`) is free web search for Go and AI —
no API keys, single binary. It embeds as a Go library, ships as a CLI, and
exposes REST + MCP servers with native OpenAI/Anthropic tool-calling schemas.
Go 1.22, one external dependency (`golang.org/x/net`).

This repo has a local `go.work` (`use .`) pinning it to itself so the parent
`sirerun/go.work` doesn't break builds here — don't remove it.

Read `NOTES.md` for what's still scaffolding (in-memory billing only, no
Stripe/DB; `internal/htmd` HTML-helper extraction deferred; marketing
install URL is a placeholder). Read `docs/ARCHITECTURE.md` for a full
module-by-module walkthrough and `docs/SPEC.md` for the external contract.

## Commands

```sh
go vet ./...
go test $(go list ./... | grep -v '/cli/') -race -cover   # matches CI; cli/ excluded from coverage
go test ./...                                              # everything, incl. cli/
go test ./chain/... -run TestName -v                       # single package/test
go test -tags=integration ./...                            # live network smoke tests (provider/startpage has one)
go build -o /tmp/saka ./cli/saka
golangci-lint run                                           # CI lint job
goreleaser release --snapshot                                # local release build
```

CI (`.github/workflows/ci.yml`) runs vet, race+cover tests (cli/ excluded,
**40% coverage floor** — deliberately honest, not a target to defend), lint,
and a `k8s-smoke` job that builds the Docker image, loads it into a `kind`
cluster, applies `deploy/k8s/`, and curls the live endpoints. The k8s-smoke
job needs Docker/kind and is expected to run on CI runners, not laptops.

## Architecture

Data flows through three call sites (CLI, REST/MCP server, tool-calling
dispatcher) into one `saka.Engine`, which has exactly three methods:
`Search`, `Fetch`, `FetchStream`.

```
cli/saka  ──┐
server/   ──┼──► saka.Engine (chain + fetcher) ──► provider/{duckduckgo,searxng,startpage}
tools/    ──┘                                  └──► fetch/ (extract, cache, robots, stream)
```

- **`types/`** — leaf package holding shared contracts (`Query`, `Result`,
  `Page`, `Provider`, `Searcher` interfaces, etc.). Every other package
  (`chain`, `fetch`, `provider/*`, `server`, `tools`) imports `types` —
  **never** the root `saka` package — to avoid import cycles. The root
  `saka` package re-exports these as type aliases (`saka.Query = types.Query`)
  so library callers keep a flat API while internals stay decoupled.
- **`saka.go`** (root package) — `Config`/`Validate`/`LoadConfig` and
  `Engine`, which wires a `chain.Chain` (providers) to a `fetch.Fetcher`
  (page retrieval) and implements `types.Searcher`. `saka.New(cfg)` is the
  one library entry point.
- **`chain/`** — runs configured providers in order with per-provider rate
  limiting (`ratelimit/`), retries with jittered exponential backoff, and a
  circuit breaker (opens after `breakerThreshold=3` consecutive failures,
  `breakerCooldown=30s`). A `*types.RateLimitError` from a provider skips
  retries and falls through to the next provider immediately.
- **`provider/{duckduckgo,searxng,startpage}/`** — each implements
  `types.Provider` (`Name()`, `Search(ctx, q)`). DuckDuckGo and Startpage
  scrape HTML with no setup required; SearXNG hits a self-hosted JSON API
  (`deploy/searxng/settings.yml` needs `"formats": ["html", "json"]`).
  `provider/startpage/integration_test.go` is gated by `//go:build
  integration` — it hits the live network and is excluded from the default
  test run.
- **`fetch/`** — page fetching and readability-style extraction
  (`extract.go`), `robots.txt` compliance (`robots.go`, on by default),
  streaming extraction in chunks (`stream.go`, feeds `/v1/stream` SSE and
  `FetchStream`), and an optional on-disk L2 cache (`diskcache.go`, plus an
  in-memory TTL cache) layered in front of network fetches.
- **`server/`** — REST (`http.go`: `/v1/search`, `/v1/fetch`, `/v1/stream`,
  `/v1/usage`, `/health`), MCP over stdio (`mcp.go`), API-key auth
  (`auth.go`: `AuthMiddleware` + per-tier token-bucket rate limiting —
  free/standard/pro tiers defined in `tiers` map), signed Ed25519 keys
  (`keys.go`: verified offline, no DB lookup at request time), and usage
  tracking (`usage.go`). `/health` is always open even under key auth;
  `/v1/usage` accepts the separate `--admin-key` Bearer token without going
  through `SignedKeys` verification. Note the doc comment in `auth.go` about
  a since-superseded `AuthMiddleware` signature from the original chat
  transcript — the current signature is the real one.
- **`tools/`** — `OpenAISchemas()` / `AnthropicSchemas()` tool-definition
  generators and one `ExecuteTool(ctx, engine, name, json.RawMessage)`
  dispatcher shared by both, so adding a tool means updating one dispatcher,
  not two integration paths.
- **`cli/saka/`** — `saka search|fetch|serve|keys`. `serve` either starts
  the HTTP mux or (`--mcp`) serves MCP over stdio; `keys` (`keys.go`)
  generates Ed25519-signed API keys and writes the public key alongside the
  private key.

## Config

`saka.json` (see `saka.json.example`, `deploy/saka.json`) configures the
provider chain (name, url, rps, retries — tried in order with fallback) and
fetch behavior (rps, cache TTL, robots.txt, optional disk cache).
`saka.DefaultConfig()` tries searxng → duckduckgo → startpage.
`Config.Validate()` enforces provider name whitelist, rps in [0,10], retries
in [0,5] — read it before changing config shape.

## Deploy

`deploy/k8s/` holds plain Kubernetes manifests (no Helm) running saka +
SearXNG in namespace `saka`; SearXNG is ClusterIP-only. `deploy/k8s/apply.sh`
takes `SAKA_IMAGE`; `deploy/smoke.sh` takes `SAKA_BASE_URL`. Applying
requires a kubectl context with Docker/kind available (CI runners, not
laptops — this repo doesn't require Docker locally for normal development).
`.goreleaser.yaml` and `install.sh` handle CLI binary releases; the
`getsaka.dev` install URL and the `sirerun/tap` Homebrew tap are placeholders
per `NOTES.md`.

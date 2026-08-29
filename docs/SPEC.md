# Saka Specification

Version: 1.2 · License: Apache-2.0 · Go: 1.22+ · External deps: `golang.org/x/net` only

---

## 1. Overview

Saka is a free web search system for Go and AI applications, shipped as:

1. **Library** — embeddable `saka.Searcher` interface
2. **CLI** — single static binary (`saka`)
3. **HTTP server** — REST + SSE, optional API-key auth
4. **MCP server** — stdio, for Claude Desktop and MCP clients
5. **Tool schemas** — OpenAI function-calling and Anthropic tool-use

## 2. Architecture

    ┌─────────────────────────────────────────────┐
    │                  cli/saka                    │
    │   search · fetch · serve · keys              │
    └───────────────┬─────────────────────────────┘
                    │
    ┌───────────────┴───────────────┐──────────────┐
    │          saka.Engine          │   server/    │
    │  (implements Searcher)        │  HTTP + MCP  │
    └──────┬──────────────────┬─────┘  auth+usage  │
           │                  │                    │
    ┌──────┴──────┐    ┌──────┴─────────┐          │
    │ chain.Chain │    │ fetch.Fetcher  │          │
    │ fallback ·  │    │ robots · cache │          │
    │ breaker ·   │    │ L1 mem/L2 disk │          │
    │ ratelimit   │    │ extraction ·   │          │
    └──────┬──────┘    │ streaming      │          │
           │           └────────────────┘          │
    ┌──────┴──────────────────────────┐            │
    │ providers (pluggable, ordered): │            │
    │  searxng → duckduckgo →         │            │
    │  startpage                      │            │
    └─────────────────────────────────┘            │
                                                   │
    tools/ — OpenAI/Anthropic schemas + dispatcher ┘

Packages:
- `saka` (root) — Engine, config, validation; re-exports `types`
- `types` — Query/Result/Page/Provider leaf contracts (breaks import cycles)
- `provider/duckduckgo`, `provider/searxng`, `provider/startpage`
- `chain` — fallback orchestration, circuit breakers, rate limits
- `fetch` — HTTP fetching, robots.txt, readability extraction,
  streaming, memory + disk caching
- `server` — REST, SSE, MCP (JSON-RPC 2.0 stdio), auth, usage
- `tools` — AI tool-calling schemas and dispatcher
- `ratelimit` — token bucket

## 3. Core API

### 3.1 Types

```go
type Searcher interface {
    Search(ctx context.Context, q Query) (*Results, error)
    Fetch(ctx context.Context, url string) (*Page, error)
    FetchStream(ctx context.Context, url string) (
        <-chan Chunk, <-chan *Page, <-chan error)
}

type Query struct {
    Text       string
    MaxResults int    // default 10
    Region     string // "us-en"
    SafeSearch bool
    Site       string
}

type Result struct {
    Title, URL, Snippet, Source string
    Position int
}

type Results struct {
    Query string; Results []Result
    Provider string; TookMs int64
}

type Page struct {
    URL, Title, Text string
    PublishedAt time.Time
}
func (p *Page) Chunks() (<-chan Chunk, error) // cached-text chunking

type Chunk struct {
    Text string; Seq int; Done bool; Err string
}
```

### 3.2 Provider interface

```go
type Provider interface {
    Name() string
    Search(ctx context.Context, q Query) ([]Result, error)
}
```

Providers signal throttling with `*saka.RateLimitError{Provider, RetryAfter}`,
which triggers immediate chain fallback (no retry on same provider).

### 3.3 Providers

| Provider | Method | Notes |
|---|---|---|
| `duckduckgo` | POST `html.duckduckgo.com/html/` | Default primary. Parses `a.result__a` / `a.result__snippet`; unwraps `uddg=` redirects. |
| `searxng` | GET `<url>/search?format=json` | Self-hosted. Requires `formats: [html, json]` in instance settings. Recommended for volume. |
| `startpage` | GET `www.startpage.com/sp/search` | Last in chain. RPS ≤ 0.2. Empty parse = challenge → `RateLimitError`. |

Adding a provider = implement `Provider`, register in `saka.New` and
`Config.Validate`.

### 3.4 Chain behavior

- Order = config order. First success wins.
- Per-provider: token-bucket rate limit, N retries with exponential
  backoff + jitter (rate-limit errors exempt from retry).
- Circuit breaker: opens after 3 consecutive failures, cools down 30s.
- Errors aggregate to `saka.ErrNoResults` when all providers fail.

### 3.5 Fetching & extraction

- HTTP GET, 5MB body cap, 20s timeout, UA rotation, per-host rate limit.
- robots.txt: fetched per host, 1h cache, groups for `*` and `SakaBot`;
  on by default (`respect_robots: true`).
- Extraction (readability-style): strip `script/style/nav/header/footer/
  aside/form/noscript/svg/iframe/button`; score candidate containers by
  text length + paragraph count with link-density penalty; prefer
  `<article>`. Title from `og:title` → `<title>`; date from
  `article:published_time` (RFC3339).
- Streaming: `ExtractStream` emits ~900-char chunks on paragraph
  boundaries during DOM walk; full page delivered on done channel and
  cached.
- Caching: L1 in-memory TTL map; L2 disk (SHA-256 content-addressed JSON,
  sharded 2-char dirs, atomic tmp-rename writes, lazy expiry + `GC()`).
  L2 is optional via config.

## 4. Configuration (`saka.json`)

```json
{
  "providers": [
    { "name": "searxng", "url": "http://localhost:8888", "rps": 5, "retries": 2 },
    { "name": "duckduckgo", "rps": 1, "retries": 2 },
    { "name": "startpage", "rps": 0.2, "retries": 1 }
  ],
  "fetch": {
    "rps": 2,
    "cache_ttl_seconds": 3600,
    "respect_robots": true,
    "disk_cache": { "dir": "~/.cache/saka", "ttl_seconds": 86400 }
  },
  "server": { "addr": ":8080", "mcp": true }
}
```

- Default config (no file): `duckduckgo → startpage`. SearXNG is opt-in;
  when configured, place it first.
- Validation rules: ≥1 provider; known names only; no duplicates;
  `searxng` requires http(s) `url`; `rps` ∈ [0,10]; `retries` ∈ [0,5];
  `cache_ttl_seconds` ≥ 0. `LoadConfig` and `New` both validate.

## 5. CLI

```
saka search "query" [-n 10] [--format table|json|markdown] [--site d] [--config f]
saka fetch <url> [--format text|json|markdown] [--no-cache]
saka serve [--addr :8080] [--mcp] [--keys f.json] [--keys-pub f.pub]
saka keys [--tier free|standard|pro] [--n 1] [--exp-days 365] [--priv f]
```

## 6. HTTP API

| Endpoint | Description |
|---|---|
| `GET /v1/search?q=&n=&format=json\|markdown` | Search |
| `GET /v1/fetch?url=&format=text\|json\|markdown` | Fetch + extract |
| `GET /v1/stream?url=` | SSE: `event: chunk\|done\|error` |
| `GET /v1/usage` | Per-key usage (admin key = full dump) |
| `GET /health` | Liveness |

`server.New(engine).Handler()` is exportable for embedding; the mux is
wrapped in auth middleware only when a `KeySource` is configured.
Keyless mode (self-hosted) has no auth.

## 7. MCP Server

- Transport: line-delimited JSON-RPC 2.0 over stdio; protocol
  `2024-11-05`; methods `initialize`, `notifications/initialized`,
  `tools/list`, `tools/call`.
- Tools: `web_search` (query, max_results, site), `fetch_page` (url).
- Tool errors are returned in-result with `isError: true` per spec.

## 8. AI Tool Schemas

- Shared JSON Schema for both formats; `tools.OpenAISchemas()`,
  `tools.AnthropicSchemas()`.
- `tools.ExecuteTool(ctx, engine, name, argsJSON) (string, error)` —
  single dispatcher; output is plain text sized for model context.

## 9. Auth & Keys (paid-service path)

- Transport auth: `Authorization: Bearer <key>`.
- Key sources: `StaticKeys` (JSON file: key→tier), `SignedKeys`
  (Ed25519, offline-verifiable), or custom `KeySource` (DB).
- Signed key format: `sk-1.<b64url(payload JSON)>.<b64url(sig)>`;
  payload `{id, tier, exp, n}`; verification checks signature + expiry;
  revocation via key-ID list.
- Tiers: `free` (10 RPM, no stream), `standard` (120 RPM, stream),
  `pro` (600 RPM, stream), `admin` (usage dump).
- Middleware enforces per-key token-bucket RPM, sets
  `X-RateLimit-Limit` / `Retry-After`, gates `/v1/stream` by tier
  (402 when unavailable).
- Usage metering: per-key daily counters (searches, fetches, streams,
  4xx/5xx, bytes out, last seen), recorded via request context;
  exposed at `/v1/usage`.

## 10. Deployment

- **Binary**: GoReleaser; linux/darwin/windows × amd64/arm64;
  `-s -w` ldflags; install via `curl -sL getsaka.dev | sh` or Homebrew tap.
- **Docker**: multi-stage build → alpine, non-root uid 10001.
- **Compose stack**: saka + SearXNG (internal network only, JSON format
  enabled, limiter off since unexposed); persistent cache volume;
  paid mode = `--keys` flag + mounted secret.

## 11. Testing & CI

- Offline suite: `go test ./... -race -cover` — fixture-based tests for
  DDG/Startpage parsers, extractor, chunking, chain fallback/breaker,
  disk cache, MCP round-trip, tool dispatcher, config validation.
  Coverage gate: ≥40%, the real enforced floor rather than an aspirational
  number — see `docs/adr/002-coverage-gate-honesty.md` for the raise plan.
- Integration suite (`-tags=integration`): live DDG, Startpage (challenge
  = skip, not fail), fetch/extract/stream against example.com, cache-hit
  timing. Provider canaries assert anchor class names still present in
  live HTML (markup-drift detection).
- CI: vet, race tests, coverage gate, golangci-lint, kind + k8s smoke.
  Release: tag-triggered, tests first, live smoke best-effort with warning.
- Dependabot: gomod, actions, docker — weekly.

## 12. Ethics & Constraints

- robots.txt respected by default; honest UA (`SakaBot/1.0 (+url)`) for
  robots matching; per-provider self-throttling; aggressive caching.
- No CAPTCHA evasion, no IP rotation, no evasion tooling. Providers that
  challenge us get backed off, not broken through.
- Heavy users are directed to self-hosted SearXNG.
- No telemetry. No phone-home. Ever.

## 13. Compatibility & Versioning

- Go 1.22+; single external dep (`golang.org/x/net`).
- Pre-2.0: breaking changes allowed in minor versions but documented;
  `Searcher` is the stability contract.
- Config schema versioned implicitly; unknown fields ignored (forward
  compatible).

## 14. Roadmap

| Version | Contents |
|---|---|
| 1.0 | Core: providers, chain, fetch/extract/stream, CLI, REST, MCP, tools, tests, release pipeline |
| 1.1 | Startpage, auth middleware, disk cache, k8s stack (saka + SearXNG) |
| 1.2 | Signed keys, usage metering, `saka keys`, CI/CD, dependabot |
| Future | `/v1/usage` persistence backend, news/images verticals, `Page.Chunks` over SSE search, additional providers via plugin convention |

# Saka Architecture

This document walks the codebase module by module: what each package
owns, why it exists, and how data flows through the system. Read
`SPEC.md` for the external contract first; this is the internal tour.

Version: 1.2 · ~2,500 LOC · 8 packages · 1 external dependency

---

## 0. The one diagram

```
                        ┌──────────────────────────────────┐
                        │            cli/saka              │
                        │  search · fetch · serve · keys   │
                        └───┬──────────┬───────────┬───────┘
                            │          │           │
              ┌─────────────┘          │           └──────────────┐
              ▼                        ▼                          ▼
    ┌──────────────────┐    ┌───────────────────┐    ┌────────────────────┐
    │   saka.Engine    │    │     server/       │    │      tools/        │
    │  (Searcher impl) │◄───┤  HTTP · SSE · MCP │◄───┤ OpenAI/Anthropic   │
    │                  │    │  auth · usage     │    │ schemas+dispatcher │
    └────────┬─────────┘    └───────────────────┘    └─────────┬──────────┘
             │                                                 │
             │  both call the same three methods:              │
             │  Search / Fetch / FetchStream                   │
             ▼◄────────────────────────────────────────────────┘
    ┌─────────────────────────────────────────────────┐
    │                  chain.Chain                     │
    │  ordered providers · rate limits · retries ·     │
    │  circuit breakers · error aggregation            │
    └───┬──────────────┬──────────────┬───────────────┘
        ▼              ▼              ▼
  ┌──────────┐  ┌──────────┐  ┌──────────┐
  │ searxng  │  │duckduckgo│  │startpage │     provider/*
  └──────────┘  └──────────┘  └──────────┘

    ┌─────────────────────────────────────────────────┐
    │                 fetch.Fetcher                    │
    │  robots.txt · UA rotation · L1 mem / L2 disk     │
    │  readability extract · ExtractStream             │
    └─────────────────────────────────────────────────┘

    internal/htmd — shared HTML helpers used by providers + fetch
    ratelimit    — token bucket used by chain + fetch
```

The critical property: **every surface (CLI, HTTP, MCP, tool-calling)
funnels into the same `Searcher` interface.** There is exactly one
implementation of the hard logic. Surfaces are adapters.

---

## 1. Package-by-package

### 1.1 `saka` (root) — the contract

Owns:

- `Searcher`, `Query`, `Result(s)`, `Page`, `Chunk`, `Provider`,
  `RateLimitError` — the vocabulary of the whole system
- `Config` + `Validate()` + `LoadConfig()` — the only place schema
  rules live
- `Engine` — the concrete `Searcher`: composes `chain.Chain` (search)
  and `fetch.Fetcher` (fetch/stream)
- `New(cfg)` — construction + provider registry (name → constructor)

Design decisions:

- **`Page.text` is unexported.** `Page.Text` is the JSON-facing full
  text; `p.text` is the internal normalized copy that `Chunks()`
  re-slices. Callers can't put a `Page` into a state where `Chunks()`
  lies.
- **`New` returns an error, not a panic.** Bad config is a user error;
  the engine refuses to exist rather than half-working.
- **`Searcher` is the stability contract.** Packages depend on the
  interface, never on `*Engine` — that's why `server` and `tools` can
  be tested with `fakeEngine` in 15 lines.

### 1.2 `provider/*` — dumb and honest

Each provider is a leaf package with exactly two jobs:

1. Translate `Query` → provider request
2. Translate provider response → `[]Result`

They own **no** retry logic, **no** rate limiting, **no** caching,
**no** fallback. Those are the chain's job. A provider either returns
results or returns an error — one of which may be
`*saka.RateLimitError` to say "back off, try someone else."

Consequences:

- A new provider is ~100 lines and touches nothing else
- Providers are trivially unit-testable (fixture HTML in, results out)
- Anti-bot detection is uniform: challenge pages parse to zero results
  → the provider converts that to `RateLimitError` (see startpage)

**Shared helpers** (`internal/htmd`): `AttrOf`, `ClassOf`, `TextOf`,
single-pass DOM walkers. Both scraping providers use them; the
readability extractor in `fetch` uses its own scoring walker (different
concerns, different walks — not worth unifying further).

**Markup-drift canary:** each scraping provider exposes
`fetchHTML(ctx, q) (io.Reader, error)` so integration tests can assert
the anchor class names (`result__a`, `w-gl__result-title`) still exist
in live HTML *before* parsing. When a search engine ships a redesign,
the canary fails with a precise message instead of a mysterious
zero-results.

### 1.3 `chain` — where the resilience lives

```
Search(q)
  │ for each provider in config order:
  ▼
  ┌─ breaker open? ── yes ─► skip, next provider
  │      │ no
  ├─ rate limiter.Wait(ctx)          (token bucket, per provider)
  ├─ attempt (up to Retries,        (RateLimitError exempt —
  │   exp backoff + jitter)          never hammered twice)
  ├─ success ─► return Results{Provider: name}
  ├─ failure ─► record failure; breaker opens after 3 consecutive
  └─ next provider
  │ all failed
  ▼
ErrNoResults (aggregated)
```

State per provider entry: `limiter`, `failStreak`, `openUntil`.

Why not retry rate-limit errors? A 429 means the provider is telling
us to stop. Retrying converts politeness into an arms race. Instead
the chain immediately falls through — that's the *desired* behavior of
a multi-provider design.

Why a breaker at all, if fallback exists? Because fallback still costs
a request. An open breaker skips a known-dead provider instantly and
lets it cool down for 30s — important when startpage is challenging us
and we don't want to train their defenses on our traffic pattern.

### 1.4 `fetch` — the most stateful package

Responsibilities, in the order a request touches them:

```
Fetch(url)
  │ 1. L1 memory cache (TTL map, mutex)
  │ 2. L2 disk cache (optional; SHA-256 content-addressed)
  │ 3. robots.txt check (per-host, 1h cached; if respect_robots)
  │ 4. rate limit (global fetch token bucket)
  │ 5. GET with UA rotation, 5MB cap, 20s timeout
  │ 6. Extract() or ExtractStream()
  │ 7. store in L1 + L2
  ▼
*Page (or chunk stream + done channel)
```

**Extraction** (readability-style, `extract.go`):

- Walk DOM once, skipping `script/style/nav/header/footer/aside/
  form/noscript/svg/iframe/button`
- Score candidate containers: text length + `<p>` count, penalized by
  link density (nav elements are mostly links)
- `<article>` gets priority; deepest-best-container wins
- Title: `og:title` → `<title>`; date: `article:published_time`

**Streaming** (`stream.go`): same walk, but text accumulates into a
~900-char buffer that flushes on paragraph boundaries onto a channel.
The done channel delivers the assembled `Page` for caching, so a
streamed fetch is *also* a cached fetch — the second request for the
same URL never re-parses. 900 chars ≈ 225 tokens: a good granularity
for incremental LLM context without SSE spam.

**Caching** — two tiers, one interface:

| Tier | Location | Survives | Cost |
|---|---|---|---|
| L1 | process memory map | process | free |
| L2 | `~/.cache/saka/<ab>/<sha>.json` | restarts | fs read |

L2 files are written atomically (`tmp` + `rename`) so a concurrent
reader never sees a half-written page. Expiry is lazy (checked on
read) plus an opportunistic `GC()` walk on startup. Sharding by the
first 2 hex chars keeps directories small at ~100k entries.

The disk cache is why the CLI and the server never both hit the
network for the same URL, and why the compose stack's volume makes
restarts cheap.

### 1.5 `server` — three protocols, one mux

```
Handler()
  │
  ├── no KeySource ──► mux (keyless self-hosted mode)
  └── KeySource set ─► AuthMiddleware ─► mux
                          │ validates Bearer key
                          │ per-key token bucket (RPM by tier)
                          │ gates /v1/stream by tier
                          │ injects key into request ctx
                          │ records usage via recordingKeySource
                          ▼
                     mux: /v1/search  /v1/fetch  /v1/stream
                          /v1/usage   /health
```

- `http.go` — REST + SSE. `handleStream` writes
  `event: chunk|done|error` frames and flushes per event; honors
  client disconnect via `r.Context()`.
- `mcp.go` — JSON-RPC 2.0 over stdio, line-delimited, one walker
  loop: `Serve(ctx, io.Reader, io.Writer)`. Deliberately transport-
  generic so tests feed it a `bytes.Buffer`.
- `auth.go` — `KeySource` interface + `AuthMiddleware`. Tiers are a
  package-level table (`free/standard/pro`); adding a tier is a table
  edit, not a code change.
- `usage.go` — `UsageStats`: per-key daily counters behind one mutex.
  Recording happens via `recordingKeySource` (touch on lookup) and
  context-injected `recordUsage` calls in handlers. The admin key
  gets a full-map dump at `/v1/usage` for nightly billing crons.
- `keys.go` — Ed25519 signed keys: `sk-1.<payload>.<sig>`.
  `SignedKeys` verifies offline; revocation is a key-ID set. This is
  the "no DB round-trip per request" path — the key *is* the
  authorization record.

### 1.6 `tools` — the AI adapter

Two schema constants (one JSON Schema each), two format wrappers
(`OpenAISchemas`, `AnthropicSchemas`), one dispatcher
(`ExecuteTool`). Output is plain text shaped for model context:
numbered results for search, title+date+body for fetch. No JSON in
tool output — models handle prose better and tokens are money.

### 1.7 `ratelimit` — 30 lines of token bucket

Used by both `chain` (per provider) and `fetch` (global). One
implementation, one behavior everywhere.

---

## 2. Data flow: three end-to-end traces

### A. `saka search "llm news"` (CLI, cold)

```
cli parse → Engine.Search → chain
  → duckduckgo: limiter.Wait → POST html.duckduckgo.com
  → 200 → parse (unwrap uddg=) → []Result
→ Results{Provider: duckduckgo, TookMs: 412} → render markdown → stdout
```

### B. Agent loop: `tools.ExecuteTool(fetch_page)`

```
model tool_call → ExecuteTool → Engine.Fetch
  → L1 miss → L2 miss → robots allow → limiter.Wait → GET
  → Extract → Page{Title, Text, PublishedAt}
  → L1 + L2 store → render text → back to model as tool result
(next fetch of same URL in this session: L1 hit, ~0µs)
```

### C. `GET /v1/stream?url=...` behind paid auth

```
Bearer sk-1.… → AuthMiddleware → VerifyKey (offline, Ed25519)
  → tier check: stream allowed? → RPM bucket consume
  → usage.touch(key) → handleStream → Engine.FetchStream
  → robots → GET → ExtractStream walk
  → SSE frames: event:chunk ×N → event:done
  (usage.Record: Streams++, BytesOut+=…)
```

---

## 3. Concurrency model

- **No goroutine leaks by construction:** every streaming path is
  driven by a context (`ctx` in, `r.Context()` in the SSE handler);
  when the client disconnects, the walk stops, channels close.
- **Locks:** `Fetcher.mu` guards the L1 map; `UsageStats.mu` guards
  the usage map; `robotsCache.mu` guards the per-host rules map. All
  are leaf locks — none held across I/O.
- **Buffered channels** (16) on streaming paths so the extractor can
  run ahead of a slow consumer without blocking the parse.
- **Race-tested:** CI runs `-race`; the fakes in tests exercise
  concurrent chain fallback explicitly.

## 4. Error taxonomy

| Error | Meaning | Handling |
|---|---|---|
| `*saka.RateLimitError` | provider throttled/challenged us | chain falls through immediately; breaker counts it; no retry |
| other provider errors | network/parse | retried up to `Retries` with backoff, then fallback |
| all providers failed | `saka.ErrNoResults` (aggregated) | surfaces to caller/SSE `error` event |
| robots disallowed | hard stop for that URL | returned as error, never retried |
| tool-level error | anything above | OpenAI/Anthropic loops: string in tool result; MCP: `isError: true` in-result |

## 5. Extension points (and what NOT to extend)

| Want to... | Do this | Don't |
|---|---|---|
| Add a provider | implement `Provider`, register in `New` + `Validate`, add fixture tests | add retry/fallback logic inside the provider |
| Change resilience policy | edit `chain` | spread policy into providers |
| Swap key storage | implement `KeySource` | hack `AuthMiddleware` |
| Change output shaping for models | edit `tools` renderers | reformat inside providers |
| Add an API surface | wrap `Searcher` | call `chain` or `fetch` directly from the surface |

The layering rule in one sentence: **dependencies point inward —
surfaces → engine → chain/fetch → providers — and never sideways or
back.**

## 6. Performance envelope (expected, single host)

| Operation | Cold | Warm (L1) | Warm (L2) |
|---|---|---|---|
| Search | 300–800ms | — (no search cache by design) | — |
| Fetch + extract | 200–1500ms | ~0µs | ~50µs |
| Stream first chunk | 100–400ms | — | — |
| Sustained search (DDG default) | 1 RPS hard cap | | |
| Sustained search (SearXNG) | 5 RPS (config) | | |

Search results are deliberately **not** cached: they're cheap for the
provider relative to page fetches, freshness matters more, and cache
invalidation for a query is unanswerable. Page fetches are cached
aggressively because extraction is expensive and pages are stable.
```

---

## Document 2: `ETHICS.md`

```markdown
# Saka Ethics & Responsible Use Policy

Saka reads the public web programmatically. That capability carries
obligations. This document states ours, and what we ask of users.

It is not legal advice. It is the policy the project actually follows,
in code and in culture.

---

## 1. What Saka does, mechanically

Every fetching behavior in Saka is deliberate and inspectable:

| Behavior | Implementation | Default |
|---|---|---|
| robots.txt compliance | fetched per host, groups for `*` and `SakaBot`, 1h cache | **on** |
| Self-identification | `SakaBot/1.0 (+https://github.com/sirerun/saka)` for robots matching | always |
| Self-throttling | token bucket: 1 RPS on DDG, 0.2 RPS on Startpage, 2 RPS on fetch | always |
| Caching | L1 memory + optional L2 disk, TTL-based | on |
| Backing off when challenged | challenge/429 → no retry, circuit breaker opens 30s | always |
| No CAPTCHA evasion | none exists in the codebase | — |
| No IP rotation, no proxy pools | none exists in the codebase | — |
| No telemetry, no phone-home | none exists in the codebase | — |

We adopt SearXNG's own position: the *right* way to scale automated
search is to **aggregate openly with self-hosted infrastructure**, not
to evade the defenses of engines that didn't consent to your traffic.

## 2. When a search engine says no

Saka treats throttles, challenges, and blocks as **answers**, not
obstacles:

- A 429 or CAPTCHA page ends that request. It is never retried.
- The circuit breaker stops all traffic to that provider for 30s.
- The chain falls through to the next provider — which may be your
  own SearXNG instance, i.e., infrastructure you control.

We will not add features whose purpose is to defeat anti-bot measures.
If a provider becomes unusable without evasion, the honest response is
to document that and point users at self-hosted alternatives.

## 3. What we ask of users

### Do
- Leave `respect_robots: true` on.
- Keep provider RPS at or below defaults unless you're self-hosting.
- Self-host SearXNG for anything beyond personal/small-team volume.
- Cache. Saka does; don't disable it to "get fresh results" on a loop.

### Don't
- Wrap Saka in distributed/proxy infrastructure to evade rate limits
  or robots.txt.
- Use Saka to build a public search *portal* on top of engines that
  haven't consented — that's what SearXNG instances with published
  policies are for.
- Resell access to scraped results without understanding the ToS of
  the underlying sources.

If your use case requires volume that only evasion can achieve, the
requirement — not Saka — is the problem.

## 4. The paid-service path

Saka ships auth, tiers, usage metering, and signed keys so *you* can
build a service on top. With that comes responsibility we make
structural, not aspirational:

- **Metering is on whenever auth is on.** You cannot bill what you
  don't measure; you shouldn't run what you don't bill.
- **Your upstream obligations pass through.** If your paid service
  serves 10,000 searches/day through Saka's default chain, you are
  making 10,000 polite requests to third-party engines. At that
  scale, run SearXNG. The compose stack exists precisely so this is
  a one-command migration.
- **The free path is not a loss leader.** Saka has no telemetry and
  no hosted service of its own. The paid scaffolding serves users who
  *choose* to run a service — it does not fund or steer the core.

## 5. Content & copyright

- Saka returns **snippets and extracted article text** to the caller.
  It does not republish, archive, or expose a browsable copy of the
  web. Extraction exists to feed a model or a reader in real time.
- Downstream use of fetched content is governed by the source's terms
  and applicable law. Saka adds `Source`/URL attribution to every
  result so downstream systems *can* attribute.
- We encourage applications built on Saka to cite sources visibly.
  Agents that fetch a page and present its content as their own
  analysis are a knowable failure mode; attributing is cheap.

## 6. Privacy

- No telemetry. No crash reporting. No analytics. The binary makes
  requests only to the providers and URLs you ask it to touch.
- The disk cache stores page text locally under `~/.cache/saka`. It
  contains what you fetched, nothing about you. Delete the directory
  and it's gone.
- API keys, if you use the paid path, live where you put them (file,
  env, or your `KeySource` implementation). They never leave your
  infrastructure except in the `Authorization` header of your own
  clients.
- Saka is **not an anonymity tool.** Search engines and sites see
  your IP like any browser would. If your threat model requires
  anonymity, route Saka through Tor/VPN yourself — we won't build it
  in, because built-in evasion corrodes the politeness guarantees
  above.

## 7. Commitments to the open web

1. **Transparency over cleverness.** Every network behavior Saka
   performs is documented in `ARCHITECTURE.md` and readable in one
   afternoon. There is no hidden second behavior.
2. **Fail polite.** Under contention, Saka backs off rather than
   pushes through — in rate limits, retries, and breakers.
3. **Prefer consented infrastructure.** Whenever an open, self-hosted
   alternative exists (SearXNG), Saka treats it as the recommended
   path for scale, in docs, defaults, and tooling.
4. **No arms race.** We will close issues and decline PRs that add
   evasion capability, with this document as the reason.
5. **Attribution is a feature.** Result objects carry source URLs
   because credit should be structurally easy, not optional.

## 8. Reporting

Found Saka being used abusively, or found a behavior in the code that
contradicts this document? Open an issue with the `ethics` label.
Contradictions between code and policy are bugs — the policy wins,
and the code gets fixed.

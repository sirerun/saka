# Saka Vision

## The problem

Every AI application that wants to search the internet faces the same
wall: paid APIs, per-key rate limits, vendor lock-in, and terms that
get worse as you scale. Hobbyists abandon their projects at the
"add web search" step. Researchers route around it with brittle
one-off scrapers. Startups pay thousands a month for what is,
fundamentally, an HTTP GET.

Meanwhile the open web is right there. DuckDuckGo serves it freely.
SearXNG aggregates it freely. The missing piece is not access —
it's a **tool**.

## The thesis

Web search should be a primitive, like gzip or curl. Free, everywhere,
good enough, nobody's business but your own.

**Saka is that primitive for the Go and AI ecosystem.**

## Principles

### 1. Free means free
No API keys. No accounts. No "free tier" that becomes a paid tier.
Saka reads the same public web your browser reads. For high-volume
use, you self-host a SearXNG instance — your infrastructure, your rules.

### 2. A library first, a product second
The core is five lines of Go:

    engine, _ := saka.New(saka.DefaultConfig())
    res, _ := engine.Search(ctx, saka.Query{Text: "ai news"})

The CLI, the servers, the tool schemas — all are thin skins over the
same engine. If it can't be embedded, it's failed.

### 3. Built for the agent loop
AI agents are the primary user. That shapes everything:
- **Structured results** the model can reason over
- **Readable extraction**, not raw HTML — tokens are money
- **Streaming** extraction for long documents
- **Tool schemas** for OpenAI, Anthropic, and MCP, out of the box
- **Caching**, because agents ask the same question twice

### 4. Polite by default, resilient by design
Saka respects robots.txt, rate-limits itself, rotates user agents
honestly, and caches to avoid repeat load. When a provider throttles
it, the circuit breaker opens and the chain falls through. Saka is a
good citizen that survives contact with the real web.

### 5. One binary, zero ceremony
`curl -sL getsaka.dev | sh`. It runs on a Raspberry Pi and in a
datacenter. One dependency beyond the standard library. Boring,
portable, permanent.

## The road

**v1.0 — The primitive.** Search, fetch, extract. CLI + library.
**v1.1 — The stack.** Self-hosted Docker bundle; paid-service
scaffolding (auth, disk cache, tiers) for those who want to build
a business on top.
**v1.2 — The platform.** Signed keys, usage metering, billing hooks.
The paid path exists so the free path never has to compromise.

## What Saka is not

- Not a Google replacement. It's a search **tool**, not a portal.
- Not a scraping framework. It does one thing: search and read.
- Not anonymous. Use Tor/VPN separately if that's your threat model.
- Not against search engines. Polite traffic and self-hosted
  aggregation are how the open web stays open.

## Success looks like

A developer's AI project gains working internet access in under a
minute, for free, forever — and if that project becomes a company,
the same binary scales into the paid product without a rewrite.



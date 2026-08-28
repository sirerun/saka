# saka

Free web search for Go and AI. No API keys. Single binary.

`Search the internet for free` — embeds in Go projects as a library,
ships as a CLI, exposes REST + MCP servers, and speaks OpenAI/Anthropic
tool-calling natively.

This tree is a direct transcription of the "saka" Go library/CLI worked
out over your Ox Alpha chat, v0.1 through v1.2 (core search → SearXNG +
MCP + tool schemas → streaming/distribution/tests → robots.txt,
Startpage, config validation, disk cache, Docker/Compose deploy → paid
API keys, per-key usage tracking, CI/CD). Every `.go`/`.yaml`/`.json`
file here comes from a code block in that conversation, organized into
the package layout the chat itself described.

**Read `NOTES.md` before running `go build` or `go test` — the chat's
own code has real gaps and inconsistencies (a circular import between
`saka` and its subpackages chief among them, plus a couple of REST
handlers that were described but never written) that need a decision
before this compiles end to end.**

## Install

```sh
curl -sL https://getsaka.dev | sh   # see NOTES.md — reconstructed, verify before use
# or
brew install you/tap/saka
# or
go install github.com/you/saka/cli/saka@latest
```

## CLI

```sh
saka search "best open source LLMs" -n 5 --markdown
saka fetch https://example.com/article --format text
saka serve --addr :8080                    # REST API
saka serve --mcp                           # MCP over stdio (Claude Desktop, etc.)
saka serve --addr :8080 --keys api_keys.json  # REST API with auth (paid mode)
saka keys --tier pro --n 5 --exp-days 30   # generate signed API keys
```

## Library (a few lines to search from Go)

```go
engine, _ := saka.New(saka.DefaultConfig())
res, _ := engine.Search(ctx, saka.Query{Text: "AI news", MaxResults: 5})
page, _ := engine.Fetch(ctx, res.Results[0].URL)   // extracted article text
```

### AI tool-calling

```go
engine, _ := saka.New(saka.DefaultConfig())
out, _ := tools.ExecuteTool(ctx, engine, "web_search",
    json.RawMessage(`{"query":"ai news","max_results":5}`))
```

- `tools.OpenAISchemas()` / `tools.AnthropicSchemas()` — tool definitions
- `tools.ExecuteTool(...)` — one dispatcher for both

### MCP

`claude_desktop_config.json`:

```json
{ "mcpServers": { "saka": {
    "command": "/usr/local/bin/saka",
    "args": ["serve", "--mcp"] } } }
```

## Config (`saka.json`)

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
  }
}
```

Providers are tried in order with automatic fallback, rate limiting,
retries, and per-provider circuit breakers. DuckDuckGo and Startpage work
with zero setup; for heavy AI usage, self-host
[SearXNG](https://docs.searxng.org/) and enable `"formats": ["html", "json"]`
in its settings (see `deploy/searxng/settings.yml`).

## API

| Endpoint | Description |
|---|---|
| `GET /v1/search?q=...&n=10&format=json\|markdown` | search *(described in chat, handler not written — see NOTES.md)* |
| `GET /v1/fetch?url=...&format=text\|json\|markdown` | fetch + extract *(same)* |
| `GET /v1/stream?url=...` | SSE streaming extraction |
| `GET /v1/usage` | per-key billing stats (needs `--keys`) |

Or mount the handler in your own Go server:

```go
mux.Handle("/v1/", sakaserver.New(engine).Handler())
```

## Self-hosted stack (Docker Compose)

```sh
git clone https://github.com/you/saka && cd saka
docker compose up -d

# verify:
curl "http://localhost:8080/v1/search?q=open+source+llm&format=json" | jq '.provider'
curl -N "http://localhost:8080/v1/stream?url=https://example.com" | head -5
```

Brings up `saka` + a private `searxng` instance on an internal network
(no ports exposed on searxng itself). See `docker-compose.yml`,
`deploy/searxng/settings.yml`, `deploy/saka.json`.

## Paid-service groundwork

`saka keys` generates Ed25519-signed API keys, verified offline (no DB
lookup needed at request time — see `server/keys.go`). `saka serve --keys
api_keys.json` enables `AuthMiddleware`, per-tier rate limits (free /
standard / pro), and `/v1/usage` billing stats. This is scaffolding, not
a finished billing system — see `NOTES.md` for what's still stubbed.

## Ethics & legality

saka scrapes politely: per-provider rate limits, honest user agents,
robots.txt respect (on by default), caching (memory + optional disk) to
reduce repeat requests. For high-volume use, run your own SearXNG
instance. Apache-2.0.

## Development

```sh
go mod tidy
go vet ./...
go test ./... -race -cover               # offline test suite
go test -tags=integration ./...          # live smoke tests
goreleaser release --snapshot            # local build
```

## Layout

```
saka/
├── saka.go, errors.go, config_test.go, saka_test.go, smoke_test.go
├── ratelimit/ratelimit.go
├── provider/
│   ├── duckduckgo/       # HTML-scraping provider (no key)
│   ├── searxng/          # self-hosted SearXNG JSON API
│   └── startpage/        # secondary HTML-scraping provider
├── fetch/                # fetch, extract, streaming, robots.txt, disk cache
├── chain/                # provider fallback chain + circuit breaker
├── server/                # MCP, REST/SSE, API-key auth, usage, signed keys
├── tools/                # OpenAI/Anthropic tool-call schemas
├── cli/saka/             # `saka search|fetch|serve|keys`
├── deploy/                # searxng settings, saka.json, smoke.sh
├── .github/workflows/     # CI + release
├── Dockerfile, docker-compose.yml
├── saka.json.example, .goreleaser.yaml, install.sh
└── README.md, NOTES.md
```

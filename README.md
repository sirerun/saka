# saka

Free web search for Go and AI. No API keys. Single binary.

`Search the internet for free` — embeds in Go projects as a library,
ships as a CLI, exposes REST + MCP servers, and speaks OpenAI/Anthropic
tool-calling natively.

Module: `github.com/sirerun/saka`. See [`NOTES.md`](NOTES.md) for the
v1 production-ready status and remaining scaffolding (billing is
in-memory only; no Stripe).

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/sirerun/saka/main/install.sh | sh
# or
brew install sirerun/tap/saka
# or
go install github.com/sirerun/saka/cli/saka@latest
```

## CLI

```sh
saka search "best open source LLMs" -n 5 --markdown
saka fetch https://example.com/article --format text
saka serve --addr :8080                    # REST API
saka serve --mcp                           # MCP over stdio (Claude Desktop, etc.)
saka serve --addr :8080 --keys saka_ed25519.pub  # REST API with signed-key auth
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
| `GET /v1/search?q=...&n=10&format=json\|markdown` | search |
| `GET /v1/fetch?url=...&format=text\|json\|markdown` | fetch + extract |
| `GET /v1/stream?url=...` | SSE streaming extraction |
| `GET /v1/usage` | per-key billing stats (needs `--keys`) |

Or mount the handler in your own Go server:

```go
mux.Handle("/v1/", sakaserver.New(engine).Handler())
```

## Self-hosted stack (Kubernetes)

Manifests under [`deploy/k8s/`](deploy/k8s/) run **saka + SearXNG** in
namespace `saka`. SearXNG is ClusterIP-only (not exposed outside the
cluster). Use kind/k3s/GKE — whatever kubectl context you already have;
this repo does not require Docker on a laptop.

```sh
# On a machine with Docker + kind (CI box / dedicated host):
docker build -t saka:local .
kind load docker-image saka:local          # or push to your registry

SAKA_IMAGE=saka:local ./deploy/k8s/apply.sh
kubectl -n saka port-forward svc/saka 8080:8080

curl "http://127.0.0.1:8080/v1/search?q=open+source+llm&format=json" | jq '.provider'
SAKA_BASE_URL=http://127.0.0.1:8080 ./deploy/smoke.sh
```

For production, point `SAKA_IMAGE` at your registry tag and apply the
same manifests (rotate the SearXNG `secret_key` in
`deploy/k8s/searxng.yaml` first).

### Helm chart

The same stack is also packaged as a chart under
[`deploy/helm/saka/`](deploy/helm/saka/), for clusters that prefer Helm
over plain manifests:

```sh
docker build -t saka:local .
kind load docker-image saka:local          # or push to your registry

helm install saka deploy/helm/saka \
  --set image.repository=saka \
  --set image.tag=local

kubectl -n saka port-forward svc/saka 8080:8080

curl "http://127.0.0.1:8080/v1/search?q=open+source+llm&format=json" | jq '.provider'
SAKA_BASE_URL=http://127.0.0.1:8080 ./deploy/smoke.sh
```

For production, override `image.repository`/`image.tag` with your
registry tag and rotate the SearXNG `secret_key` via
`--set searxng.settings=...` (or a values file) first.

## Paid-service groundwork

`saka keys` generates Ed25519-signed API keys, verified offline (no DB
lookup needed at request time — see `server/keys.go`). `saka serve --keys
saka_ed25519.pub` enables `AuthMiddleware`, per-tier rate limits (free /
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
├── saka.go, types/, config_test.go, saka_test.go, smoke_test.go
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
├── deploy/                # k8s manifests, smoke.sh, reference configs
├── .github/workflows/     # CI + release
├── Dockerfile
├── saka.json.example, .goreleaser.yaml, install.sh
└── README.md, NOTES.md
```

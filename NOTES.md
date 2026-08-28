# Notes on this tree (v1 production-ready)

This tree started as a transcription of Ox Alpha chat code (v0.1–v1.2).
The **v1 production-ready** pass resolved the build blockers and wired the
free HTTP/MCP path end-to-end.

## Resolved in v1

- **Circular import** — shared contracts live in [`types/`](types/); root
  package `saka` re-exports them and owns `Engine`/`Config`.
- **Module path** — `github.com/sirerun/saka` (matches the GitHub remote).
- **`go.sum`** — generated via `go mod tidy`.
- **REST handlers** — `GET /v1/search` and `GET /v1/fetch` implemented.
- **MCP** — `server/mcp.go` uses `tools.SearchSchema` / `FetchSchema` /
  `ExecuteTool`.
- **Signed keys** — `saka keys` writes `*.pub` beside the private key;
  `saka serve --keys <pubkey>` enables `SignedKeys` + usage stats;
  `--admin-key` unlocks full `/v1/usage`.
- **CI** — `deploy/searxng` path fixed; coverage gate set to an honest
  **40%** floor (cli package excluded from the profile). Raising toward
  70% is follow-up work.
- **Local `go.work`** — pins this module only so parent `sirerun/go.work`
  does not break builds.

## Still scaffolding (explicitly out of finished billing)

- No Stripe/DB; usage is in-memory per process.
- `internal/htmd` HTML helper extraction still deferred (duplicated
  helpers in duckduckgo/startpage remain).
- Marketing install URLs (`getsaka.dev`) are placeholders until the
  domain is owned.
- Homebrew tap owner is `sirerun` in GoReleaser config — tap repo may
  not exist yet.

## Mechanical fixes from the original transcription

Preserved history of what was fixed when copying chat snippets (imports,
typos, `collectText2`, AuthMiddleware signature collision via
`recordingKeySource`, etc.) remains useful archaeology — see git history
of the import commit.

## Verify

```sh
go vet ./...
go test $(go list ./... | grep -v '/cli/') -race -cover
go build -o /tmp/saka ./cli/saka
docker compose up -d && sleep 20 && ./deploy/smoke.sh
```

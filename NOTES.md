# Notes on this transcription

Everything in this tree was copied from code blocks in your Ox Alpha chat
(v0.1 through v1.2) and sorted into files by the filename/path each block
was labeled with (e.g. "`provider/duckduckgo/duckduckgo.go`",
"`server/keys.go`"). A few mechanical things were necessary to make files
self-contained, and a number of real bugs in the chat's own code are
called out below rather than silently fixed, since fixing some of them
is a design decision, not a typo fix.

## Fixed mechanically (no logic changed)

- Added missing standard-library/internal imports the code obviously
  needs but the chat's snippet omitted: `net/http` + `io` in
  `provider/duckduckgo/duckduckgo.go`; `io` in `fetch/stream.go`;
  `github.com/you/saka/ratelimit` in `fetch/fetch.go`;
  `net/http/cookiejar` in `provider/startpage/startpage.go` (the chat's
  own page rendered this import as a bare `"` — see "Corrupted in the
  source page" below); `strings` in `server/usage.go` and
  `server/keys.go`.
- Dropped unused imports the chat's snippets included: `strconv` in
  `duckduckgo.go` and `searxng.go`; `strings`/`golang.org/x/net/html` in
  `fetch/fetch.go` (used by `extract.go` instead); `crypto/subtle` in
  `server/auth.go` (never actually used for a constant-time compare).
- `provider/duckduckgo/duckduckgo.go`: removed a dead second HTML parse
  the chat itself flagged as leftover.
- `chain/chain.go`: dropped an unused `var rl *saka.RateLimitError` /
  `_ = rl` pair.
- `provider/duckduckgo/duckduckgo_test.go`: removed a duplicated/garbled
  loop (`for _, c := range range_c(c) {}`) ahead of the real one.
- Several test/fake files (`server/mcp_test.go`, `tools/tools_test.go`,
  `fetch/diskcache_test.go`) set the lowercase `text` field on
  `saka.Page` directly, or used a bare `Page{...}` from a different
  package. Both fixed to use the exported `Text` field (and, in
  `diskcache_test.go`, a proper `saka "github.com/you/saka"` import and
  `saka.Page{...}`).
- `tools/tools_test.go` imported an unused, made-up
  `fakesaka "github.com/you/saka/testutil"` instead of the real `saka`
  package it uses — swapped for `saka "github.com/you/saka"`.
- `fetch/stream.go` calls `collectAllText(best)`, never defined anywhere
  in the chat. Added a small `collectText2` helper reusing the existing
  `collectText` walker; the original line is left in as a comment.
- Several **dropped-token typos** in the v1.1/v1.2 (auth/keys/usage)
  pass, all fixed to evident intent:
  - `server/usage.go`: `type KeyUsage struct	Day string ...` was
    missing the opening `{` after `struct`.
  - `server/usage.go`: `var _ KeySource = middlewareKeySource{}`
    referenced a type that's never defined; the very next type in the
    same snippet, `recordingKeySource`, is clearly what was meant.
  - `server/keys.go` (`VerifyKey`): called an undefined `splitN(...)`
    (comment: "manual: strings.Split, len check") — changed to
    `strings.SplitN`; a `return p,` was missing its error value — now
    `return p, err`.
  - `cli/saka/keys.go` (`doKeys`): `loadOrCreatePrivprivPath)` was
    missing its opening `(`; the `KeyPayload{ ID: ... }` literal was
    missing the `ID:` field name and had an unclosed `fmt.Sprintf(...)`.
    A `randHex(n int) string` helper is referenced but was never
    defined anywhere in the chat — added (crypto/rand + hex-encode).
  - Server wiring snippet (labeled "Wiring: free vs paid handler"):
    `if s.usage nil {` was missing `!=`, and `"/v1usage"` was missing a
    `/`.

## Corrupted in the source page (not a copy-paste target)

Two code blocks displayed nested, duplicated `$(...)`/`${...}`
substitutions in the chat page itself — almost certainly a
markdown/templating rendering bug on Ox Alpha's side, not something to
use verbatim:

- **`install.sh`** — the download URL and version lookup each appeared
  repeated 2–3 times inside themselves.
- **`.github/workflows/ci.yml`**'s coverage-gate shell line — same
  pattern (`[ "$(echo"$(echo"$(echo"cov >= 70" | bc)" = 1 ]`, etc.).
- **`provider/startpage/startpage.go`**'s import block rendered
  `net/http/cookiejar` as a bare `"` on its own line.

Both scripts here are clean reconstructions of the evident intent —
verify them rather than trusting them as a verbatim transcript.

## Real issues, left for you to resolve

- **Circular import.** `saka.go`'s `Engine` needs `chain.Chain` and
  `fetch.Fetcher` (and now also imports `provider/duckduckgo`,
  `provider/searxng`, `provider/startpage`), so package `saka` imports
  `chain` and `fetch` — but both of those import `saka` for the shared
  types (`Query`, `Result`, `Page`, `Provider`, `RateLimitError`, ...).
  Go refuses circular imports; this will not build as laid out. The
  chat never addressed this across any of v0.1–v1.2. Common fixes: move
  the shared types into their own leaf package (e.g. `saka/types`), or
  move `Engine`/`New` out of package `saka` into a small `engine`
  subpackage.
- **`server/mcp.go` calls the wrong package.** It calls
  `saka.SearchSchema()`, `saka.FetchSchema()`, `saka.ExecuteTool(...)`,
  but those are defined in `tools/tools.go` (package `tools`), not
  `saka`.
- **`server/http.go`'s `handleSearch` / `handleFetch` are still
  stubs.** The chat described `GET /v1/search` and `GET /v1/fetch` in
  prose (and referenced them in the README, in `deploy/smoke.sh`, and in
  the CI smoke test) across all three passes (v0.2, v1.1, v1.2) but
  never once wrote their bodies. They return HTTP 501 here so the
  package compiles; wire up `saka.Query` from `r.URL.Query()` and call
  `s.engine.Search` / `s.engine.Fetch` per the README's own endpoint
  table.
- **Two conflicting `AuthMiddleware` signatures.** v1.1 wrote
  `AuthMiddleware(keys KeySource, next http.Handler) http.Handler`
  (`server/auth.go`). v1.2's usage-tracking pass tried to give it a
  second signature, `AuthMiddleware(keys KeySource, stats *UsageStats,
  next http.Handler)` — which Go can't have alongside the first (no
  overloading). Resolved here by keeping only the v1.1 signature and
  routing usage tracking through a `KeySource` **decorator**
  (`recordingKeySource` in `server/usage.go`, doc-commented with the
  call pattern) instead of changing `AuthMiddleware` itself. The
  `recordUsage`/`usageKey`-context-value wiring the chat sketched for
  inside `handleSearch`/`handleFetch` was left out, since those handlers
  don't exist yet either — add it alongside whoever implements them.
- **`Options.AdminKey` and `Options.Usage`** are referenced by the
  wiring snippets (`s.opts.AdminKey`, `if s.usage != nil`) but the
  `Options` struct as shown only ever declared a `Keys` field. Added
  both fields to `Options` in `server/http.go` so the wiring compiles;
  nothing yet calls `NewUsageStats()` and threads it into `Options` —
  do that in `cli/saka/main.go`'s `doServe` alongside the `--keys`
  flag if you want `/v1/usage` live.
- **`internal/htmd` package was suggested, never created.** Both
  `duckduckgo.go` and `startpage.go` have near-identical
  `attr`/`class`/`text` HTML helpers, and `startpage.go`'s own comment
  says to "extract to an internal package (saka/internal/htmd)". That
  package doesn't exist; `provider/startpage/integration_test.go`
  imports it (commented out here so the file compiles under
  `-tags=integration`).
- **Deploy/CLI mismatch (fixed here, flagged for awareness).** The
  chat's `docker-compose.yml` set `SAKA_CONFIG=/etc/saka/saka.json` as
  an environment variable, but the CLI only ever reads a config path
  from `--config` — there's no env-var support anywhere in
  `cli/saka/main.go` — and nothing mounted `deploy/saka.json` into the
  container. `docker-compose.yml` here mounts the file and passes
  `--config` explicitly instead.
- **`Dockerfile` needs a `go.sum`.** It `COPY go.mod go.sum ./` but no
  `go.sum` exists in this tree (this sandbox has no network access to
  the Go module proxy to generate one) — run `go mod tidy` once you can
  reach the proxy.
- **`smoke_test.go`'s `TestSmokeSearchDDG`** originally asserted
  `res.Provider == "duckduckgo"`; since `DefaultConfig()` was changed in
  this same pass to try SearXNG first, that assertion would fail
  whenever SearXNG is reachable. Changed to just log the provider
  instead of asserting a specific one.
- **`DefaultConfig()`'s own comment is unresolved.** The chat's diff
  added `// auto-skipped if not running? No: see note` on the SearXNG
  entry and never wrote the note or the skip-if-down behavior it's
  musing about. Preserved verbatim as a comment; there is no actual
  "skip if SearXNG is unreachable" logic beyond the normal
  fallback/circuit-breaker behavior every provider already gets.
- **CI's `cp deploy/searx/settings.yml ...`** (note: `searx`, not
  `searxng`) is a harmless no-op typo from the chat (`|| true` swallows
  the failure) — left as-is since it doesn't break anything, but the
  line does nothing useful either.
- A couple of small gaps in reconstructed text where the page's own
  extraction cut off mid-line were filled in with the obviously-intended
  short completion (e.g. one `t.Errorf` message, a couple of loop/brace
  boundaries) — not pulled from anything the chat said beyond that
  point.
- Added, not from the chat: `go.mod`; a `homeDir()` helper in `saka.go`
  (the disk-cache wiring calls it to expand `~`, never defined);
  `Fetcher.SetDiskCache` (the wiring describes calling it, chat never
  wrote the method); a couple of `usage()`/README lines to keep the CLI
  help text in sync with subcommands added later (`serve`, `keys`).

## Not carried over as files (design/prose only in the chat, not real files)

Inline examples that were illustrative, not separate files: the early
"3. Core API" / "4. Provider System" drafts (superseded by the real
`saka.go`), the OpenAI/Anthropic tool-loop usage snippets, the
`claude_desktop_config.json` MCP client snippet, the `curl -N
.../v1/stream` example, and the "Try it" / "One command" shell
walkthroughs. Their content is reflected in `README.md` where useful.

## Not fully verified

Go's module proxy (`proxy.golang.org`) isn't reachable from this
sandbox, so `golang.org/x/net` couldn't be fetched and packages that
depend on it (`fetch`, `provider/duckduckgo`, `provider/startpage`, and
transitively `saka` once the circular-import issue above is resolved)
couldn't be fully `go build`/`go test`-verified here. `gofmt` was run
over the whole tree (clean) as a syntax check, and the circular-import
error above was confirmed by an actual `go build` failure.

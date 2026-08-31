# ADR 003: Search verticals are a Query field routing to per-vertical chains

## Status
Accepted

## Date
2026-08-29

## Context
E7 (News Vertical) needs a way for callers to ask for news results distinct
from general web results, without those two kinds of results competing or
substituting for each other. E8 (Images Vertical) and any future vertical
will need the same mechanism, so the design must generalize, not be
news-specific.

The existing `chain.Chain` (chain/chain.go) implements first-success
fallback: it tries configured providers in order and returns the first one
that succeeds. This is the right model for *redundant* providers of the
*same* kind of content (duckduckgo, searxng, startpage all return general
web results) -- if one fails, try the next. It is the wrong model for a
*distinct* kind of content: if a caller configured `[gdelt, duckduckgo]` in
one chain, a general web search could silently return GDELT news results
(or vice versa) depending on which provider happened to succeed first. A
vertical is not a fallback option for general search; it is a different
search a caller explicitly asks for.

Two mechanisms were considered:
1. Add a new method to `types.Searcher` (e.g. `SearchNews`). Rejected: the
   `Searcher` interface (Search/Fetch/FetchStream) is a fixed public
   contract used identically across the CLI, REST server, MCP server, and
   tools dispatcher (saka.go, server/http.go, server/mcp.go, tools/*).
   Adding a method per vertical breaks every existing implementer and
   caller of the interface, and doesn't scale past one or two verticals.
2. Add an optional `Vertical` field to `types.Query`, and give `Engine` one
   `chain.Chain` per distinct vertical value present in the configured
   providers. `Engine.Search` looks up the requested vertical's chain.
   Accepted.

## Decision
- `types.Query` gains an optional `Vertical string` field. Empty string
  (the zero value) means general web search -- this is the exact behavior
  every existing caller already gets, so this is a strictly additive,
  backward-compatible change. `Query.WithDefaults()` leaves `Vertical`
  untouched (no default vertical; empty stays empty).
- `types.ProviderConfig` gains an optional `Vertical string` field,
  defaulting to `""` (general web). A provider entry with
  `"vertical": "news"` in `saka.json` belongs to the news vertical, not
  the general chain.
- `saka.Engine` changes its single `chain *chain.Chain` field to
  `chains map[string]*chain.Chain`, keyed by vertical (`""` is the
  general-web key, matching today's only chain). `New()` groups
  `cfg.Providers` by `Vertical` before calling `chain.New` once per
  group -- `chain.Chain` itself is unchanged; it still only ever sees one
  vertical's providers per instance.
- `Engine.Search(ctx, q)` resolves `q.WithDefaults().Vertical` (default
  `""`) against `e.chains`. An unconfigured vertical (e.g. a caller asks
  for `"news"` but no news provider is configured) returns a clear error
  (`saka: no provider configured for vertical %q`) rather than silently
  falling back to general web results or returning `ErrNoResults`.
- A vertical's provider(s) are ordinary `types.Provider` implementations
  self-registering through the existing registry (`types.Register`,
  ADR 001) -- the registry itself needs no change. `provider/gdelt`
  (E7's provider) registers as `"gdelt"` exactly like `provider/duckduckgo`
  registers as `"duckduckgo"`; only `ProviderConfig.Vertical` determines
  which chain it lands in.
- Public API surface for requesting a vertical (REST query param, CLI
  flag, MCP tool schema) is specified per-vertical in that epic's own
  task breakdown (see docs/plans/E7-news-vertical.md), not here -- this
  ADR fixes the internal mechanism, not every caller-facing spelling.

## Consequences
Positive: verticals compose -- E8 (images) and any future vertical reuse
this exact mechanism with zero further engine changes, just a new
`Provider` implementation and a new `Vertical` config value. Existing
callers (anyone not setting `Query.Vertical`) see no behavior change:
same chain, same fallback semantics, same error type. `chain.Chain` and
the provider registry (ADR 001) stay untouched, so this doesn't reopen
either of those already-shipped, already-tested subsystems.

Negative: `Engine.New` must reject (or must be tested against) a config
where two providers with different verticals share the same provider
`Name` (unlikely given today's fixed provider names, but a real edge case
once third-party providers exist per the registry). A caller who
misspells or forgets a vertical's config gets a runtime error at
`Search()` time, not a config-validation-time error -- `Config.Validate()`
could be extended to catch an unconfigured `Vertical` up front, which is
worth doing as part of E7's implementation rather than deferred silently.

## Addendum 2026-08-29: E8 (images) reuses this mechanism, extends Result

E8's images vertical confirms the "verticals compose" prediction above:
it needs no `chain.Chain` or registry change, only a new provider and a
`Vertical: "images"` config value. Founder decision (2026-08-29): source
is SearXNG's existing `categories=images` query mode (the self-hosted
SearXNG instance this repo already integrates for general web search),
not a separate image-search API -- consistent with saka's "no API keys"
posture. The provider lives in `provider/searxng` (shares the package's
HTTP client and base-URL config) but registers under a distinct name,
`"searxng-images"`, not `"searxng"` -- a config entry's provider `Name`
should unambiguously say what it does; overloading one registered name
for two verticals would make `saka.json` harder to read and would not
save any real code (the images provider still needs its own `Search`
method to parse SearXNG's image-result JSON shape, which differs from
its general-web shape).

`types.Result` gains three optional fields for this: `ThumbnailURL`,
`Width`, `Height` (all `omitempty` in JSON, zero value when a provider
doesn't set them -- general web providers are unaffected and need no
changes). SearXNG's `categories=images` response includes `img_src`
(the full-size image URL -- mapped to `Result.URL`, keeping "URL of the
thing found" consistent across verticals), `thumbnail_src` (mapped to
the new `ThumbnailURL`), and `resolution` (a string like `"1920x1080"`
that the provider parses into `Width`/`Height`; a malformed or absent
resolution leaves both at zero rather than failing the whole result).

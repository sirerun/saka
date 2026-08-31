# ADR 004: Search results stream only after a provider succeeds, via a new Searcher method

## Status
Accepted

## Date
2026-08-30

## Context
E9 (Streaming Search Results) needs a way for a caller to receive
`Search` results incrementally instead of waiting for the full response.
`saka` already has one streaming precedent -- `FetchStream(ctx, url)
(<-chan Chunk, <-chan *Page, <-chan error)` -- alongside the blocking
`Fetch`. `Search` has no streaming counterpart yet.

Two questions needed resolving:

1. **Does streaming a search response mean streaming across the
   provider fallback chain, or streaming the winning provider's
   results?** `chain.Chain.Search` (chain/chain.go) tries configured
   providers in order and returns the first success. Streaming
   *across* that -- emitting partial/tentative results from a provider
   attempt that might still fail over to the next provider -- would
   leak chain internals to callers and force them to reconcile
   contradictory partial responses when a fallback occurs. Founder
   decision (2026-08-30): stream only once a provider has fully
   succeeded (mirroring how `FetchStream` streams the chunks of one
   already-selected page, not chunks from multiple candidate URLs).
   `chain.Chain` and its fallback semantics are unchanged.
2. **Does this reopen ADR 003's rejected alternative** (adding a method
   per vertical to `types.Searcher`, rejected because it doesn't scale
   past one or two verticals and breaks every implementer)? No --
   streaming is a single, vertical-agnostic capability, exactly like
   the existing `Fetch`/`FetchStream` pair. One method is added once,
   not once per vertical; a caller passes `Query.Vertical` through
   `SearchStream` exactly as it does through `Search`. ADR 003's
   `Query.Vertical` mechanism and this ADR compose without touching
   each other's code paths (verified by E9's task T9.5, which proves a
   vertical query streams from its own chain, not the general one).

## Decision
- `types.Searcher` gains `SearchStream(ctx context.Context, q Query)
  (<-chan Result, <-chan *Results, <-chan error)`, matching
  `FetchStream`'s exact channel shape: an item channel (`Result`, one
  per hit), a "done" channel carrying the final `*Results` (Provider,
  TookMs, Query -- the same summary `Search` returns today, delivered
  once all `Result`s have been sent), and an error channel.
- `saka.Engine` is the only concrete implementer of `types.Searcher` in
  this repo (confirmed by grep -- `provider/*` implement the narrower
  `types.Provider`, not `Searcher`), so this is a single-site change.
  `Engine.SearchStream` calls the existing, unmodified `e.Search(ctx,
  q)` synchronously to get one winning `*Results`, then streams its
  `.Results` slice over the item channel before closing it and sending
  the summary on the done channel. No new provider-level protocol, no
  `chain.Chain` change.
- Surfaces: REST (`GET /v1/search/stream`, SSE, mirroring
  `/v1/stream`'s `event: result`/`event: done`/`event: error`
  convention) and CLI (`saka search --stream`). MCP is explicitly
  **out of scope** for this pass -- this repo's MCP server runs over
  stdio as a request/response JSON-RPC transport, not SSE; there is no
  existing streaming tool-call surface to extend, and inventing one is
  a separate, larger protocol decision than E9's scope. Revisit only if
  a future MCP transport change adds native streaming tool calls.

## Consequences
Positive: reuses the exact channel-shape convention already established
by `FetchStream`, so callers familiar with one immediately understand
the other. `chain.Chain`, the provider registry (ADR 001), and the
vertical mechanism (ADR 003) all stay untouched -- this is purely an
`Engine`-level addition plus new REST/CLI surfaces.

Negative: `types.Searcher` is no longer purely "fixed" in the sense
ADR 003 described it (a second concrete method has now been added since
this repo's genesis) -- future proposals to add a *per-vertical*
Searcher method should still be rejected for the reasons ADR 003 gives;
this ADR's method is vertical-agnostic and does not set that precedent.
Any future external implementer of `types.Searcher` (none exist today)
must implement `SearchStream` too.

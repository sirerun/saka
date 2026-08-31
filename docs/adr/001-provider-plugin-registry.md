# ADR 001: Provider plugin registry replaces the hardcoded switch

## Status
Accepted

## Date
2026-08-29

## Context
`saka.New` and `Config.Validate` (saka.go) each contain a hardcoded switch
over the three built-in provider names (`duckduckgo`, `searxng`,
`startpage`). Adding a fourth provider -- built-in or third-party --
requires editing both switches in the root package. `docs/SPEC.md` section
14 already lists "additional providers via plugin convention" as Future
work, and E3 in `docs/plan.md` schedules it as frontier work once the
`internal/htmd` dedup (E2) gives provider packages a clean shared base to
build the registry against.

The alternative considered was leaving the switch as-is and just adding
cases for each new vertical/provider as they appear. That stays simplest
for exactly three providers but does not scale to third-party providers at
all (they cannot live outside the `saka` module and still be registered),
and every new built-in provider becomes a two-file, root-package change
even though the provider's own package is self-contained.

## Decision
Adopt **`init()`-side-effect registration** -- the same convention as
`database/sql`, the `image/{png,jpeg,gif}` codecs, and `net/http/pprof`:
each provider package registers its own factory from its own `init()`, and
a caller pulls a provider into the binary by importing its package (a
blank import is enough when nothing else from the package is referenced
directly).

The explicit-registration alternative -- `saka.New(cfg, providers
...Provider)`, or a `Register` step callers must invoke before `New` --
is rejected. It either breaks `New`'s existing `func(Config) (*Engine,
error)` signature (a compatibility break for every current library
caller), or, if signature-preserving, still makes `saka.New` /
`Config.Validate` the one place every provider -- built-in or third-party
-- must be individually wired before it's usable. That's the same
coupling T3.3/T3.4 exist to remove, just relocated to a different call
site. Only `init()`-side-effect registration makes a provider package
genuinely self-contained: nothing outside it needs to know it exists
beyond importing it.

**Where the registry lives.** Not the root `saka` package: `saka.go`
already imports `provider/{duckduckgo,searxng,startpage}` directly
(saka.go:14-16), so a provider package importing `saka` back, to call
`saka.Register` from its own `init()`, would be an import cycle. `types`
is the existing leaf package every provider already imports and that
never imports `saka` (see this repo's CLAUDE.md Architecture section) --
the canonical `Register`/`Lookup` implementation goes in `types`, and the
root `saka` package re-exports a thin wrapper plus a type alias, matching
the existing `Provider = types.Provider` re-export convention, so
third-party callers depend on `saka`'s public API and never need to
import `types` directly.

**Signature** (new file `types/registry.go`):

```go
// ProviderFactory constructs a Provider from a config entry.
type ProviderFactory func(cfg ProviderConfig) (Provider, error)

// Register adds a provider factory under name. Provider packages call
// this from their own init(). Returns an error -- never panics -- on a
// nil factory or a name that's already registered, so callers (including
// T3.5's unit tests) can assert on the return value without recover().
func Register(name string, factory ProviderFactory) error

// Lookup returns the factory registered under name, and whether one
// exists. Used by saka.New and Config.Validate in place of the switch.
func Lookup(name string) (ProviderFactory, bool)

// Registered returns the sorted names of every currently registered
// provider, for Config.Validate's "unknown provider" error message.
func Registered() []string
```

`saka.go` re-exports, alongside its existing type aliases:

```go
type ProviderFactory = types.ProviderFactory

// Register is a convenience wrapper over types.Register for third-party
// provider packages, so they depend on saka's public API, not types.
func Register(name string, factory ProviderFactory) error {
	return types.Register(name, factory)
}
```

**Built-in providers cost library callers nothing extra.** T3.3 moves each
built-in provider's registration into its own package's `init()`, calling
`types.Register` directly (each already imports `types`, never `saka` --
so no cycle there either). `saka.go` keeps importing all three provider
packages (once T3.4 removes the direct `duckduckgo.New()` /
`searxng.New()` / `startpage.New()` calls from `New`, those become blank
imports kept purely for the `init()` side effect), so `duckduckgo`,
`searxng`, and `startpage` register themselves the moment the `saka`
package is imported -- a caller who only wants the built-ins never
touches the registry or writes an extra import. Only genuinely
out-of-tree third-party providers need their own explicit blank import
(`_ "github.com/example/saka-provider-bing"`), which is the standard,
well-understood Go convention for this exact case and needs no new
mechanism from saka.

**Duplicate rejection returns an error, not a panic**, specifically so
T3.2's unit test ("registers two providers under the same name and
asserts the second call returns an error") doesn't need `recover()`. The
three built-in providers' own `init()` functions still panic if their own
`Register` call returns a non-nil error -- a duplicate or nil factory
among the built-ins is a saka bug caught at every program start, not a
runtime condition a library caller should have to handle.

## Consequences
Positive: a new provider (built-in or third-party) is addable without
touching the root `saka` package; `docs/SPEC.md`'s "Adding a provider"
section (3.3) simplifies to "implement `Provider`, call `Register`"; the
news/images verticals (E7/E8, if built as separate provider-like
implementations rather than a new `Query` field) can plug in the same way.

Negative: registration via `init()` side effects means a provider package
must be imported (a blank import suffices) for it to be usable -- an
implicit coupling some Go style guides discourage, though it's the same
tradeoff `database/sql` drivers and `image` codecs already ask Go
developers to accept, and built-in providers pay no extra cost since
`saka.go` imports all three unconditionally (above). `Config.Validate`'s
"known provider names" whitelist becomes dynamic (`types.Registered()`)
rather than a fixed list, which is the intended flexibility but changes
the error message shape for an unknown provider name -- covered by
T3.5's test for the unknown-provider error path. Two provider packages
that both register under the same name (e.g. two third-party packages
both claiming `"bing"`) fail at whichever `init()` runs second, and
Go's `init()` order across unrelated packages is import-order-dependent
and not something the registry controls -- a constraint to document for
third-party provider authors (T3.6), not one the registry design can
solve.

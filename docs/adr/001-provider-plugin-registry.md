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
Introduce a provider registry: a thread-safe `Register(name string, factory
func(cfg types.ProviderConfig) (types.Provider, error))` call that each
provider package invokes from its own `init()` (or an explicit
`Register()` call from `main`/`saka.New` for packages that prefer not to
rely on `init()` side effects -- resolved during T3.1's design task).
`saka.New` and `Config.Validate` resolve provider names through the
registry instead of a switch. The three built-in providers
(`provider/duckduckgo`, `provider/searxng`, `provider/startpage`) migrate
to self-registration as part of the same change (T3.3), so there is only
ever one convention, not a hybrid of built-in-switch plus
plugin-registry-for-everyone-else.

## Consequences
Positive: a new provider (built-in or third-party) is addable without
touching the root `saka` package; `docs/SPEC.md`'s "Adding a provider"
section (3.3) simplifies to "implement `Provider`, call `Register`"; the
news/images verticals (E7/E8, if built as separate provider-like
implementations rather than a new `Query` field) can plug in the same way.

Negative: registration via `init()` side effects (if chosen in T3.1) means
providers must be imported for their `init()` to run, which is an implicit
coupling some Go style guides discourage -- T3.1 must weigh this
explicitly against an explicit-registration alternative (e.g. `saka.New`
takes a list of provider factories) and record the choice in this ADR's
Decision section before T3.2 implements it. Config.Validate's "known
provider names" whitelist becomes dynamic (whatever is registered) rather
than a fixed list, which is the intended flexibility but changes the error
message shape for an unknown provider name -- covered by T3.5's test for
the unknown-provider error path.

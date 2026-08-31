# ADR 002: Coverage gate states its real floor, raised incrementally

## Status
Accepted

## Date
2026-08-29

## Context
`docs/SPEC.md` section 11 states "Coverage gate: >=70%" as if it were the
enforced CI policy. The actual gate in `.github/workflows/ci.yml` is 40%,
with an inline comment: "Honest v1 floor: network-heavy packages keep
total below 70%. Raise as more unit tests land." The real per-package
numbers as of 2026-08-29 (from `go test -race -coverprofile`): chain
92.9%, ratelimit 100%, searxng 80.0%, tools 71.4%, duckduckgo 68.2%,
server 59.6%, startpage 50.7%, types 36.4%, fetch 34.0%, root saka 30.3%;
total 51.8%. So the codebase is already above the enforced 40% floor, but
nowhere near the 70% the spec claims, and the weakest packages (fetch,
types, root saka) are exactly the ones with the least test investment.

Leaving this as-is means the canonical external spec overclaims the
project's own quality bar -- a reader trusting SPEC.md would believe CI
already enforces 70% when it enforces 40%, a 30-point gap with no
documented plan to close it.

## Decision
`docs/SPEC.md` section 11 is corrected to state the real floor (40%,
already exceeded at 51.8% actual) and links to this ADR for the raise
plan, instead of asserting a target CI does not enforce. E1's tasks
(T1.2-T1.4) add tests specifically to the three weakest packages (fetch,
types, root saka), and T1.5 raises the enforced gate from 40% to 55% once
those land and CI is verified green at the new floor -- a number the
codebase can actually sustain today, not aspirational. Future raises
follow the same pattern: land tests first, raise the gate second, never
the reverse (a gate raised ahead of real coverage just fails CI on the
next network-flaky package touch).

## Consequences
Positive: SPEC.md becomes accurate rather than aspirational; the 55% gate
is provably sustainable at merge time (T1.5 verifies CI green before
landing); the pattern (tests land, then gate rises) is documented once
here instead of re-litigated on every future coverage PR.

Negative: 55% is still short of the originally stated 70% -- closing the
remaining gap is not scheduled by this ADR or by E1; it is left as future
work the next coverage pass should pick up, using the same tests-then-gate
sequencing.

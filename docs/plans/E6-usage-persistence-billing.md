# E6 -- Usage Persistence & Billing

Acceptance: usage stats survive a restart, and paid tiers are actually billed (Stripe), not just RPM-gated.
fidelity: outline

Intent: server/usage.go's UsageStats is an in-memory map today (NOTES.md: "No Stripe/DB; usage is in-memory only"). This epic picks a durable store for usage counters and wires Stripe billing to the existing tier model in server/auth.go. Needs a store decision (embedded like SQLite vs external like Postgres/Redis) and a Stripe account (kind: human) before real decomposition -- deferred past the frontier per docs/plan.md's rolling-wave policy.

Exit criteria: T6.0 expands this file to executable fidelity with >=5 tasks, each carrying acceptance criteria and dependencies.

- [ ] T6.0 PLAN: expand E6 to executable fidelity (informed by frontier learnings, especially E3's registry pattern if billing needs a pluggable store backend)  Owner: pool  Est: 1h  kind: plan  delivers: [plans/E6.md at fidelity: executable]  deps: [T1.5, T2.4, T3.6, T4.4, T5.4]  acc: [parse_plan.py sees E6 with >= 5 tasks, every task carries acceptance criteria, deps resolve, fidelity flipped to executable]

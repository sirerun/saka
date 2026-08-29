# E8 -- Images Vertical

Acceptance: callers can search image results distinct from general web results.
fidelity: outline

Intent: docs/SPEC.md section 14 lists "images verticals" as Future work. Needs a source decision (SearXNG image category vs a dedicated image API) and a Result/Page shape decision (thumbnail URL, dimensions) before decomposition -- deferred past the frontier.

Exit criteria: T8.0 expands this file to executable fidelity with >=5 tasks, each carrying acceptance criteria and dependencies.

- [ ] T8.0 PLAN: expand E8 to executable fidelity (informed by E3's provider registry and E7's news-vertical shape, if E7 has landed by then)  Owner: pool  Est: 1h  kind: plan  delivers: [plans/E8.md at fidelity: executable]  deps: [T1.5, T2.4, T3.6, T4.4, T5.4]  acc: [parse_plan.py sees E8 with >= 5 tasks, every task carries acceptance criteria, deps resolve, fidelity flipped to executable]

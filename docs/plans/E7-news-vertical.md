# E7 -- News Vertical

Acceptance: callers can search news results distinct from general web results.
fidelity: outline

Intent: docs/SPEC.md section 14 lists "news ... verticals" as Future work. Needs a decision on source (a provider that supports a news-specific query mode, e.g. SearXNG's news category, vs a dedicated news API) before decomposition -- deferred past the frontier.

Exit criteria: T7.0 expands this file to executable fidelity with >=5 tasks, each carrying acceptance criteria and dependencies.

- [ ] T7.0 PLAN: expand E7 to executable fidelity (informed by E3's provider registry, which this likely builds on)  Owner: pool  Est: 1h  kind: plan  delivers: [plans/E7.md at fidelity: executable]  deps: [T1.5, T2.4, T3.6, T4.4, T5.4]  acc: [parse_plan.py sees E7 with >= 5 tasks, every task carries acceptance criteria, deps resolve, fidelity flipped to executable]

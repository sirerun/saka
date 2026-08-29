# E1 -- Coverage & Test Debt Paydown

Acceptance: CI's coverage gate is raised from 40% to 55% and passes; SPEC.md's coverage claim matches the enforced gate. Decision rationale: docs/adr/002-coverage-gate-honesty.md.
fidelity: executable

- [ ] T1.1 Reconcile SPEC.md section 11's "Coverage gate: >=70%" against ci.yml's actual 40% floor -- state the real floor and link docs/adr/002-coverage-gate-honesty.md  Owner: TBD  Est: 30m  verifies: [infrastructure]  acc: [docs/SPEC.md section 11 states the enforced gate percentage verbatim as it appears in .github/workflows/ci.yml]
- [ ] T1.2 Add table-driven tests for fetch/fetch.go's Fetch/FetchStream error paths (timeout, non-200, robots-disallowed, body-cap exceeded) -- currently 34.0% covered  Owner: TBD  Est: 1h  verifies: [infrastructure]  acc: [go test ./fetch/... -cover reports fetch package coverage >= 55%]
- [ ] T1.3 Add tests for types/types.go's Page.Chunks and splitChunks edge cases (empty text, exact chunk-size boundary, multi-byte runes) -- currently 36.4% covered  Owner: TBD  Est: 45m  verifies: [infrastructure]  acc: [go test ./types/... -cover reports types package coverage >= 60%]
- [ ] T1.4 Add tests for saka.go's Config.Validate edge cases and New()'s provider-construction error paths -- currently 30.3% covered  Owner: TBD  Est: 1h  verifies: [infrastructure]  acc: [go test . -cover reports root saka package coverage >= 55%]
- [ ] T1.5 Raise ci.yml's coverage gate from 40% to 55% and verify CI green at the new floor  Owner: TBD  Est: 30m  verifies: [infrastructure]  deps: [T1.2, T1.3, T1.4]  acc: [.github/workflows/ci.yml's coverage gate check compares against 55, and the next CI run on main passes it]

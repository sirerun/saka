# E3 -- Provider Plugin Registry

Acceptance: a new Provider can be registered and used from saka.json without editing saka.go's New or Config.Validate. Decision rationale: docs/adr/001-provider-plugin-registry.md.
fidelity: executable

- [ ] T3.1 Design the provider registry API (Register/factory signature; init()-side-effect vs explicit-registration tradeoff) and record the chosen approach in docs/adr/001-provider-plugin-registry.md's Decision section  Owner: TBD  Est: 1h  verifies: [UC-024]  deps: [T2.1]  acc: [docs/adr/001-provider-plugin-registry.md's Decision section names a concrete Register function signature, not TBD]
- [ ] T3.2 Implement the registry (thread-safe Register/Lookup, duplicate-name rejection) per T3.1's design  Owner: TBD  Est: 1h  verifies: [UC-024]  deps: [T3.1]  acc: [a unit test registers two providers under the same name and asserts the second call returns an error]
- [ ] T3.3 Migrate provider/duckduckgo, provider/searxng, provider/startpage to self-register via the new convention  Owner: TBD  Est: 1h  verifies: [UC-024]  deps: [T3.2]  acc: [saka.go's New no longer contains a hardcoded switch over provider names; all three built-in providers are constructible via the registry]
- [ ] T3.4 Update saka.New and Config.Validate to resolve providers through the registry instead of the hardcoded switch and name whitelist  Owner: TBD  Est: 1h  verifies: [UC-024]  deps: [T3.2]  acc: [Config.Validate rejects an unregistered provider name with an error naming it unregistered, and go test ./... passes]
- [ ] T3.5 Add unit tests for registry Register/Lookup, duplicate-name rejection, and the unknown-provider error path through Config.Validate  Owner: TBD  Est: 45m  verifies: [UC-024]  deps: [T3.2]  acc: [go test ./... -run TestRegistry passes and covers the unknown-provider path]
- [ ] T3.6 Update docs/SPEC.md section 3.3 ("Adding a provider") to document the registry convention  Owner: TBD  Est: 30m  verifies: [UC-024]  deps: [T3.4]  acc: [docs/SPEC.md section 3.3 shows a Register call, not "register in saka.New and Config.Validate"]

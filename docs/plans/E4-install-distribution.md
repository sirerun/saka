# E4 -- Install & Release Distribution

Acceptance: the documented install paths (curl one-liner, Homebrew) work against real, owned infrastructure instead of placeholders.
fidelity: executable

- [ ] T4.1 Decide the real install domain: own getsaka.dev, or repoint install.sh/README at github.com/sirerun/saka releases -- FOUNDER DECISION  Owner: David  Est: 15m  verifies: [UC-025]  kind: human  acc: [a decision is recorded in this task's completion note naming the chosen domain/URL]
- [ ] T4.2 Create the sirerun/tap GitHub repo with a Formula/saka.rb scaffold matching GoReleaser's brew section  Owner: David  Est: 1h  verifies: [UC-025]  kind: human  deps: [T4.1]  acc: [github.com/sirerun/tap exists and contains Formula/saka.rb]
- [ ] T4.3 Update install.sh and README.md to match the domain/URL decided in T4.1  Owner: TBD  Est: 30m  verifies: [UC-025]  deps: [T4.1]  acc: [install.sh's curl target and README's install snippet both reference the decided URL, not getsaka.dev, unless T4.1 chose to own that domain]
- [ ] T4.4 Cut a test release tag and verify GoReleaser publishes the Homebrew formula to sirerun/tap and `brew install sirerun/tap/saka` succeeds  Owner: TBD  Est: 1h  verifies: [UC-025]  deps: [T4.2, T4.3]  acc: [brew install sirerun/tap/saka installs a working saka binary on a clean machine/container]

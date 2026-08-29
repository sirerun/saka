# E2 -- internal/htmd Dedup

Acceptance: provider/duckduckgo and provider/startpage no longer duplicate HTML-scraping helpers; both import a shared internal/htmd package.
fidelity: executable

- [x] T2.1 Extract the HTML-scraping helpers duplicated between provider/duckduckgo and provider/startpage (user-agent rotation, response-body reading, common DOM-walk utilities) into a new internal/htmd package  Owner: this session  Est: 1.5h  verifies: [infrastructure]  acc: [internal/htmd package exists, exports the helpers previously duplicated, and go build ./... succeeds]  **Done 2026-08-29 (PR #18, merged). Exports UserAgents/RandomUserAgent/SetUserAgent, Parse, Walk/Attr/Class/HasClass/Text. Providers untouched, per scope. Follow-up found but not included: fetch/fetch.go has a third, previously untracked UA-rotation duplicate (`pickUA`) -- not yet a tracked task.**
- [ ] T2.2 Refactor provider/duckduckgo to import internal/htmd and delete its local copies of the extracted helpers  Owner: TBD  Est: 45m  verifies: [infrastructure]  deps: [T2.1]  acc: [grep for the extracted helper function bodies in provider/duckduckgo/duckduckgo.go returns no matches, and go test ./provider/duckduckgo/... passes]
- [ ] T2.3 Refactor provider/startpage to import internal/htmd and delete its local copies of the extracted helpers  Owner: TBD  Est: 45m  verifies: [infrastructure]  deps: [T2.1]  acc: [grep for the extracted helper function bodies in provider/startpage/startpage.go returns no matches, and go test ./provider/startpage/... passes]
- [ ] T2.4 Add unit tests for internal/htmd covering fixtures from both providers' existing test data  Owner: TBD  Est: 1h  verifies: [infrastructure]  deps: [T2.1]  acc: [go test ./internal/htmd/... -cover reports coverage >= 70%]

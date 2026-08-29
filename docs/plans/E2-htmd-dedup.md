# E2 -- internal/htmd Dedup

Acceptance: provider/duckduckgo and provider/startpage no longer duplicate HTML-scraping helpers; both import a shared internal/htmd package.
fidelity: executable

- [ ] T2.1 Extract the HTML-scraping helpers duplicated between provider/duckduckgo and provider/startpage (user-agent rotation, response-body reading, common DOM-walk utilities) into a new internal/htmd package  Owner: TBD  Est: 1.5h  verifies: [infrastructure]  acc: [internal/htmd package exists, exports the helpers previously duplicated, and go build ./... succeeds]
- [ ] T2.2 Refactor provider/duckduckgo to import internal/htmd and delete its local copies of the extracted helpers  Owner: TBD  Est: 45m  verifies: [infrastructure]  deps: [T2.1]  acc: [grep for the extracted helper function bodies in provider/duckduckgo/duckduckgo.go returns no matches, and go test ./provider/duckduckgo/... passes]
- [ ] T2.3 Refactor provider/startpage to import internal/htmd and delete its local copies of the extracted helpers  Owner: TBD  Est: 45m  verifies: [infrastructure]  deps: [T2.1]  acc: [grep for the extracted helper function bodies in provider/startpage/startpage.go returns no matches, and go test ./provider/startpage/... passes]
- [ ] T2.4 Add unit tests for internal/htmd covering fixtures from both providers' existing test data  Owner: TBD  Est: 1h  verifies: [infrastructure]  deps: [T2.1]  acc: [go test ./internal/htmd/... -cover reports coverage >= 70%]

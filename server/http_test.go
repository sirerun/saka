package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	saka "github.com/sirerun/saka"
)

func TestHandleSearchJSON(t *testing.T) {
	s := New(fakeEngine{})
	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=ai&n=5&format=json", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	var res saka.Results
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Results) == 0 || res.Results[0].URL == "" {
		t.Fatalf("expected results: %+v", res)
	}
}

func TestHandleFetchText(t *testing.T) {
	s := New(fakeEngine{})
	req := httptest.NewRequest(http.MethodGet, "/v1/fetch?url=https://example.com&format=text", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "body" {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestHandleSearchMissingQ(t *testing.T) {
	s := New(fakeEngine{})
	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rr.Code)
	}
}

// TestHandleSearchStreamResults is T9.2's core acceptance test: GET
// /v1/search/stream?q=... must emit one or more "event: result" SSE frames
// followed by exactly one "event: done" frame, mirroring /v1/stream's
// established chunk/done convention.
func TestHandleSearchStreamResults(t *testing.T) {
	s := New(fakeEngine{})
	req := httptest.NewRequest(http.MethodGet, "/v1/search/stream?q=ai", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	resultCount := strings.Count(body, "event: result")
	doneCount := strings.Count(body, "event: done")
	if resultCount == 0 {
		t.Fatalf("expected at least one event: result frame, got body: %q", body)
	}
	if doneCount != 1 {
		t.Fatalf("expected exactly one event: done frame, got %d: %q", doneCount, body)
	}
	if strings.Index(body, "event: done") < strings.LastIndex(body, "event: result") {
		t.Fatalf("event: done arrived before the last event: result: %q", body)
	}
	if !strings.Contains(body, `"https://x"`) {
		t.Fatalf("expected result data in body: %q", body)
	}
}

func TestHandleSearchStreamMissingQ(t *testing.T) {
	s := New(fakeEngine{})
	req := httptest.NewRequest(http.MethodGet, "/v1/search/stream", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rr.Code)
	}
}

// TestHandleSearchStreamVertical proves &vertical= is forwarded from
// GET /v1/search/stream into SearchStream exactly as it is for /v1/search
// (T7.4), streaming from the news chain rather than the general one. Reuses
// the fake-general/fake-news providers registered in vertical_test.go.
func TestHandleSearchStreamVertical(t *testing.T) {
	cfg := saka.Config{
		Providers: []saka.ProviderConfig{
			{Name: "fake-general", RPS: 1},
			{Name: "fake-news", RPS: 1, Vertical: "news"},
		},
		Fetch: saka.FetchConfig{RPS: 1, RespectRobots: false},
	}
	engine, err := saka.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s := New(engine)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/stream?q=climate&vertical=news", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "https://news.example/a") {
		t.Fatalf("expected news result in body: %q", body)
	}
	if strings.Contains(body, "https://general.example/a") {
		t.Fatalf("expected no general result in body: %q", body)
	}

	doneIdx := strings.Index(body, "event: done")
	if doneIdx == -1 {
		t.Fatalf("expected event: done frame: %q", body)
	}
	var done saka.Results
	if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.SplitN(body[doneIdx:], "\n", 2)[1], "data: ")), &done); err != nil {
		t.Fatal(err)
	}
	if done.Provider != "fake-news" {
		t.Fatalf("done.Provider = %q, want fake-news", done.Provider)
	}
}

// errSearchStreamEngine wraps fakeEngine, overriding only SearchStream to
// fail immediately, for exercising handleSearchStream's "event: error" path.
type errSearchStreamEngine struct{ fakeEngine }

func (errSearchStreamEngine) SearchStream(_ context.Context, _ saka.Query) (<-chan saka.Result, <-chan *saka.Results, <-chan error) {
	ch := make(chan saka.Result)
	close(ch)
	done := make(chan *saka.Results, 1)
	errc := make(chan error, 1)
	errc <- errors.New("boom")
	return ch, done, errc
}

func TestHandleSearchStreamError(t *testing.T) {
	s := New(errSearchStreamEngine{})
	req := httptest.NewRequest(http.MethodGet, "/v1/search/stream?q=ai", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "event: error") {
		t.Fatalf("expected event: error frame: %q", body)
	}
	if !strings.Contains(body, "boom") {
		t.Fatalf("expected error message in body: %q", body)
	}
	if strings.Contains(body, "event: done") {
		t.Fatalf("expected no event: done frame on error: %q", body)
	}
}

func TestAdminUsageBypassesSignedKeys(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	key, err := SignKey(priv, KeyPayload{ID: "k1", Tier: "pro", Exp: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	stats := NewUsageStats()
	stats.touch(key)
	stats.Record(key, func(ku *KeyUsage) { ku.Searches++ })

	s := NewWithOptions(fakeEngine{}, Options{
		Keys:     SignedKeys{Pub: pub},
		AdminKey: "admin-secret",
		Usage:    stats,
	})
	h := s.Handler()

	// Health stays open under auth.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != "ok" {
		t.Fatalf("health: %d %q", rr.Code, rr.Body.String())
	}

	// Admin bearer dumps all usage without a signed key.
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
	req.Header.Set("Authorization", "Bearer admin-secret")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("admin usage: %d %s", rr.Code, rr.Body.String())
	}
	var dump map[string]*KeyUsage
	if err := json.Unmarshal(rr.Body.Bytes(), &dump); err != nil {
		t.Fatal(err)
	}
	if dump[key] == nil || dump[key].Searches != 1 {
		t.Fatalf("dump=%+v", dump)
	}
}

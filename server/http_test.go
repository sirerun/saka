package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

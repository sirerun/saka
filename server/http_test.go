package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

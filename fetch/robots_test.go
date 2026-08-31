package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRobotsAllowedNoRobotsTxt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	rc := newRobotsCache()
	if !rc.Allowed(context.Background(), srv.Client(), "SakaBot", srv.URL+"/anything") {
		t.Error("expected allow when robots.txt is missing")
	}
}

func TestRobotsAllowedDisallow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /private\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := newRobotsCache()
	if rc.Allowed(context.Background(), srv.Client(), "SakaBot", srv.URL+"/private/x") {
		t.Error("expected disallow for /private path")
	}
	if !rc.Allowed(context.Background(), srv.Client(), "SakaBot", srv.URL+"/public") {
		t.Error("expected allow for /public path")
	}
}

func TestRobotsAllowedCachesPerHost(t *testing.T) {
	var robotsHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			atomic.AddInt32(&robotsHits, 1)
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /private\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := newRobotsCache()
	rc.Allowed(context.Background(), srv.Client(), "SakaBot", srv.URL+"/a")
	rc.Allowed(context.Background(), srv.Client(), "SakaBot", srv.URL+"/b")
	if got := atomic.LoadInt32(&robotsHits); got != 1 {
		t.Errorf("expected robots.txt fetched once, got %d", got)
	}
}

func TestRobotsAllowedBadURL(t *testing.T) {
	rc := newRobotsCache()
	if rc.Allowed(context.Background(), http.DefaultClient, "SakaBot", "http://example.com/\x7f") {
		t.Error("expected disallow (false) for malformed url")
	}
}

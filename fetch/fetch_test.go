package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const fetchArticleHTML = `<!DOCTYPE html>
<html><head><title>Fetch Article</title></head><body>
<article><p>` + `Fetched content goes here and needs to be long enough to win the density scoring against nav and footer noise. ` + `</p></article>
</body></html>`

func TestFetchSuccess(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(fetchArticleHTML))
	}))
	defer srv.Close()

	f := New(0, time.Minute, false)
	page, err := f.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Text, "Fetched content") {
		t.Errorf("unexpected text: %q", page.Text)
	}

	// second call should be served from the memory cache, not a new request.
	if _, err := f.Fetch(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected 1 origin hit, got %d", got)
	}
}

func TestFetchBadURL(t *testing.T) {
	f := New(0, time.Minute, false)
	if _, err := f.Fetch(context.Background(), "http://example.com/\x7f"); err == nil {
		t.Error("expected error for malformed url")
	}
}

func TestFetchNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	f := New(0, time.Minute, false)
	if _, err := f.Fetch(context.Background(), srv.URL); err == nil {
		t.Error("expected error for non-200 status")
	}
}

func TestFetchRespectsRobotsDisallow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
			return
		}
		_, _ = w.Write([]byte(fetchArticleHTML))
	}))
	defer srv.Close()

	f := New(0, time.Minute, true)
	if _, err := f.Fetch(context.Background(), srv.URL+"/page"); err == nil {
		t.Error("expected robots.txt disallow error")
	}
}

func TestFetchUsesDiskCache(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(fetchArticleHTML))
	}))
	defer srv.Close()

	dc, err := NewDiskCache(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	f := New(0, time.Minute, false)
	f.SetDiskCache(dc)

	if _, err := f.Fetch(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}

	// fresh fetcher, same disk cache: should hit disk, not origin.
	f2 := New(0, time.Minute, false)
	f2.SetDiskCache(dc)
	if _, err := f2.Fetch(context.Background(), srv.URL); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected 1 origin hit across fetchers sharing disk cache, got %d", got)
	}
}

func TestFetchTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hang past the caller's deadline
	}))
	defer srv.Close()
	defer close(block) // unblock the handler before srv.Close() waits on it

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	f := New(0, time.Minute, false)
	if _, err := f.Fetch(ctx, srv.URL); err == nil {
		t.Error("expected error for a request past its context deadline")
	}
}

func TestFetchBodyCapEnforced(t *testing.T) {
	padding := strings.Repeat("a", 6<<20) // past fetch.go's 5<<20 (5MB) cap
	body := "<html><body><article><p>" + padding + "MARKER_BEYOND_CAP</p></article></body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	f := New(0, time.Minute, false)
	page, err := f.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("expected the body cap to truncate rather than error, got: %v", err)
	}
	if strings.Contains(page.Text, "MARKER_BEYOND_CAP") {
		t.Error("body past the 5<<20 cap was not truncated -- marker placed beyond the cap leaked into extracted text")
	}
}

func TestFetchStreamSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fetchArticleHTML))
	}))
	defer srv.Close()

	f := New(0, time.Minute, false)
	chunks, done, errc := f.FetchStream(context.Background(), srv.URL)

	var text strings.Builder
	for c := range chunks {
		text.WriteString(c.Text)
	}
	select {
	case page := <-done:
		if !strings.Contains(page.Text, "Fetched content") {
			t.Errorf("unexpected streamed text: %q", page.Text)
		}
	case err := <-errc:
		t.Fatalf("unexpected stream error: %v", err)
	}
}

func TestFetchStreamRespectsRobotsDisallow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
			return
		}
		_, _ = w.Write([]byte(fetchArticleHTML))
	}))
	defer srv.Close()

	f := New(0, time.Minute, true)
	chunks, _, errc := f.FetchStream(context.Background(), srv.URL+"/page")
	for range chunks {
	}
	select {
	case err := <-errc:
		if err == nil {
			t.Error("expected robots.txt disallow error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream error")
	}
}

func TestFetchStreamNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := New(0, time.Minute, false)
	chunks, _, errc := f.FetchStream(context.Background(), srv.URL)
	for range chunks {
	}
	select {
	case err := <-errc:
		if err == nil {
			t.Error("expected error for non-200 status")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream error")
	}
}

func TestFetchStreamTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hang past the caller's deadline
	}))
	defer srv.Close()
	defer close(block) // unblock the handler before srv.Close() waits on it

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	f := New(0, time.Minute, false)
	chunks, _, errc := f.FetchStream(ctx, srv.URL)
	for range chunks {
	}
	select {
	case err := <-errc:
		if err == nil {
			t.Error("expected error for a request past its context deadline")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream error")
	}
}

func TestFetchStreamBodyCapEnforced(t *testing.T) {
	padding := strings.Repeat("a", 6<<20) // past fetch.go's 5<<20 (5MB) cap
	body := "<html><body><article><p>" + padding + "MARKER_BEYOND_CAP</p></article></body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	f := New(0, time.Minute, false)
	chunks, done, errc := f.FetchStream(context.Background(), srv.URL)
	for range chunks {
	}
	select {
	case page := <-done:
		if strings.Contains(page.Text, "MARKER_BEYOND_CAP") {
			t.Error("body past the 5<<20 cap was not truncated -- marker placed beyond the cap leaked into extracted text")
		}
	case err := <-errc:
		t.Fatalf("expected the body cap to truncate rather than error, got: %v", err)
	}
}

func TestPickUAReturnsKnownAgent(t *testing.T) {
	ua := pickUA()
	if ua == "" {
		t.Fatal("empty user agent")
	}
}

package fetch

import (
	"testing"
	"time"

	types "github.com/sirerun/saka/types"
)

func TestDiskCacheRoundTrip(t *testing.T) {
	dc, err := NewDiskCache(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	page := &types.Page{URL: "https://x", Title: "T", Text: "hello"}
	dc.Put(page.URL, page)

	got, ok := dc.Get("https://x")
	if !ok || got.Title != "T" || got.Text != "hello" {
		t.Fatalf("round trip failed: %+v ok=%v", got, ok)
	}
}

func TestDiskCacheExpiry(t *testing.T) {
	dc, _ := NewDiskCache(t.TempDir(), time.Millisecond)
	dc.Put("https://x", &types.Page{URL: "https://x", Text: "t"})
	time.Sleep(5 * time.Millisecond)
	if _, ok := dc.Get("https://x"); ok {
		t.Error("expired entry returned")
	}
}

func TestDiskCacheDifferentURLsDontCollide(t *testing.T) {
	dc, _ := NewDiskCache(t.TempDir(), time.Hour)
	dc.Put("https://a.com", &types.Page{URL: "https://a.com", Text: "A"})
	dc.Put("https://b.com", &types.Page{URL: "https://b.com", Text: "B"})
	a, _ := dc.Get("https://a.com")
	b, _ := dc.Get("https://b.com")
	if a.Text != "A" || b.Text != "B" {
		t.Error("cache collision")
	}
}

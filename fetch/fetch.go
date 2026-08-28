// Package fetch retrieves pages and extracts readable text.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/you/saka"
	"github.com/you/saka/ratelimit"
)

const botUserAgent = "SakaBot/1.0 (+https://github.com/you/saka)"

type Fetcher struct {
	client        *http.Client
	limiter       *ratelimit.Limiter
	mu            sync.Mutex
	cache         map[string]cacheEntry
	ttl           time.Duration
	robots        *robotsCache
	respectRobots bool
	disk          *DiskCache // optional L2 tier; may be nil
}

type cacheEntry struct {
	page      *saka.Page
	expiresAt time.Time
}

func New(rps float64, ttl time.Duration, respectRobots bool) *Fetcher {
	return &Fetcher{
		client:        &http.Client{Timeout: 20 * time.Second},
		limiter:       ratelimit.New(rps),
		cache:         make(map[string]cacheEntry),
		ttl:           ttl,
		robots:        newRobotsCache(),
		respectRobots: respectRobots,
	}
}

// SetDiskCache attaches an optional L2 disk cache. Added in the v1.1 pass
// (see NOTES.md: "wire in memory -> disk tiering") — the chat described
// this wiring in prose/snippets but never wrote the method itself, so
// this is a small addition to make that wiring compile.
func (f *Fetcher) SetDiskCache(dc *DiskCache) {
	f.disk = dc
}

// Fetch retrieves a URL and extracts readable article text.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*saka.Page, error) {
	if _, err := url.Parse(rawURL); err != nil {
		return nil, fmt.Errorf("fetch: bad url: %w", err)
	}

	// L1: memory
	f.mu.Lock()
	if ce, ok := f.cache[rawURL]; ok && time.Now().Before(ce.expiresAt) {
		f.mu.Unlock()
		return ce.page, nil
	}
	f.mu.Unlock()

	// L2: disk
	if f.disk != nil {
		if page, ok := f.disk.Get(rawURL); ok {
			f.mu.Lock()
			f.cache[rawURL] = cacheEntry{page: page, expiresAt: time.Now().Add(f.ttl)}
			f.mu.Unlock()
			return page, nil
		}
	}

	if err := f.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	if f.respectRobots {
		if !f.robots.Allowed(ctx, f.client, botUserAgent, rawURL) {
			return nil, fmt.Errorf("fetch: disallowed by robots.txt: %s", rawURL)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", pickUA())
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch: status %d for %s", resp.StatusCode, rawURL)
	}
	// Respect Retry-After on 429
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("fetch: rate limited by origin")
	}

	page, err := Extract(rawURL, io.LimitReader(resp.Body, 5<<20)) // 5MB cap
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.cache[rawURL] = cacheEntry{page: page, expiresAt: time.Now().Add(f.ttl)}
	f.mu.Unlock()
	if f.disk != nil {
		f.disk.Put(rawURL, page)
	}
	return page, nil
}

// FetchStream fetches a URL and streams text chunks as they are extracted.
// The done channel delivers the fully assembled page for caching.
func (f *Fetcher) FetchStream(ctx context.Context, rawURL string) (<-chan saka.Chunk, <-chan *saka.Page, <-chan error) {
	ch, done, errc2 := make(chan saka.Chunk, 16), make(chan *saka.Page, 1), make(chan error, 1)

	go func() {
		defer close(ch)
		if err := f.limiter.Wait(ctx); err != nil {
			errc2 <- err
			return
		}
		if f.respectRobots {
			if !f.robots.Allowed(ctx, f.client, botUserAgent, rawURL) {
				errc2 <- fmt.Errorf("fetch: disallowed by robots.txt: %s", rawURL)
				return
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			errc2 <- err
			return
		}
		req.Header.Set("User-Agent", pickUA())

		resp, err := f.client.Do(req)
		if err != nil {
			errc2 <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errc2 <- fmt.Errorf("fetch: status %d", resp.StatusCode)
			return
		}

		chunks, doneCh, errCh := ExtractStream(rawURL, io.LimitReader(resp.Body, 5<<20))
		for c := range chunks {
			ch <- c
		}
		select {
		case page := <-doneCh:
			f.mu.Lock()
			f.cache[rawURL] = cacheEntry{page: page, expiresAt: time.Now().Add(f.ttl)}
			f.mu.Unlock()
			if f.disk != nil {
				f.disk.Put(rawURL, page)
			}
			done <- page
		case err := <-errCh:
			errc2 <- err
		}
	}()
	return ch, done, errc2
}

func pickUA() string {
	uas := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/17.4 Safari/605.1.15",
	}
	return uas[time.Now().UnixNano()%int64(len(uas))]
}

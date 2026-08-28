// Package chain runs providers in order with rate limiting, retries, and a circuit breaker.
package chain

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/you/saka"
	"github.com/you/saka/ratelimit"
)

type entry struct {
	provider  saka.Provider
	limiter   *ratelimit.Limiter
	retries   int
	fails     int
	openUntil time.Time
}

type Chain struct {
	mu      sync.Mutex
	entries []*entry
}

func New(cfgs []saka.ProviderConfig, ps []saka.Provider) *Chain {
	c := &Chain{}
	for _, cfg := range cfgs {
		for _, p := range ps {
			if p.Name() == cfg.Name {
				c.entries = append(c.entries, &entry{
					provider: p,
					limiter:  ratelimit.New(cfg.RPS),
					retries:  cfg.Retries,
				})
			}
		}
	}
	return c
}

const breakerThreshold = 3
const breakerCooldown = 30 * time.Second

// Search tries each provider in order; skips open breakers; retries with backoff.
func (c *Chain) Search(ctx context.Context, q saka.Query) (*saka.Results, error) {
	start := time.Now()
	for _, e := range c.entries {
		if c.isOpen(e) {
			continue
		}
		var results []saka.Result
		var err error
		for attempt := 0; attempt <= e.retries; attempt++ {
			if err = e.limiter.Wait(ctx); err != nil {
				break
			}
			results, err = e.provider.Search(ctx, q)
			if err == nil {
				c.recordSuccess(e)
				return &saka.Results{
					Query:    q.Text,
					Results:  results,
					Provider: e.provider.Name(),
					TookMs:   time.Since(start).Milliseconds(),
				}, nil
			}
			if isRateLimit(err) {
				break // don't retry a rate-limited provider — fall through
			}
			// backoff with jitter
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		if err != nil {
			c.recordFailure(e)
		}
	}
	return nil, saka.ErrNoResults
}

func isRateLimit(err error) bool {
	_, ok := err.(*saka.RateLimitError)
	return ok
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<attempt) * 500 * time.Millisecond
	return d + time.Duration(rand.Int63n(int64(d/2)))
}

func (c *Chain) isOpen(e *entry) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Before(e.openUntil)
}

func (c *Chain) recordSuccess(e *entry) {
	c.mu.Lock()
	e.fails = 0
	c.mu.Unlock()
}

func (c *Chain) recordFailure(e *entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e.fails++
	if e.fails >= breakerThreshold {
		e.openUntil = time.Now().Add(breakerCooldown)
		e.fails = 0
	}
}

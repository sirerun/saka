// Package ratelimit — minimal token bucket, stdlib only.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	rate     float64 // tokens per second
	lastFill time.Time
}

func New(rps float64) *Limiter {
	if rps <= 0 {
		rps = 1
	}
	return &Limiter{tokens: rps, max: rps, rate: rps, lastFill: time.Now()}
}

func (l *Limiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := time.Now()
		l.tokens = min(l.max, l.tokens+now.Sub(l.lastFill).Seconds()*l.rate)
		l.lastFill = now
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		need := (1 - l.tokens) / l.rate
		l.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(need * float64(time.Second))):
		}
	}
}

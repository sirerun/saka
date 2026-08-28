// Package server — API key authentication with per-key rate limits.
package server

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// KeyTier defines a billing/usage tier.
type KeyTier struct {
	RPM    int  // requests per minute
	Stream bool // allow /v1/stream
}

var tiers = map[string]KeyTier{
	"free":     {RPM: 10, Stream: false},
	"standard": {RPM: 120, Stream: true},
	"pro":      {RPM: 600, Stream: true},
}

// KeySource maps an API key to a tier name. In production, back this with
// a DB or implement your own KeySource — a static map is fine to start.
type KeySource interface {
	Lookup(key string) (tier string, ok bool)
}

type StaticKeys map[string]string // key -> tier

func (s StaticKeys) Lookup(key string) (string, bool) {
	tier, ok := s[key]
	return tier, ok
}

// AuthMiddleware validates Authorization: Bearer <key> and enforces
// per-key rate limits. Mount it in front of the paid handler only.
//
// NOTE: the source chat later (v1.2 "usage tracking") replaced this
// signature with AuthMiddleware(keys KeySource, stats *UsageStats, next
// http.Handler) — see usage.go and NOTES.md for that (incompletely
// specified) revision. This is the original, self-contained version.
func AuthMiddleware(keys KeySource, next http.Handler) http.Handler {
	var (
		mu      sync.Mutex
		buckets = map[string]*struct {
			tokens   float64
			lastFill time.Time
		}{}
	)

	allow := func(key string, rpm int) bool {
		mu.Lock()
		defer mu.Unlock()
		b, ok := buckets[key]
		if !ok {
			b = &struct {
				tokens   float64
				lastFill time.Time
			}{tokens: float64(rpm), lastFill: time.Now()}
			buckets[key] = b
		}
		now := time.Now()
		b.tokens += now.Sub(b.lastFill).Minutes() * float64(rpm)
		if b.tokens > float64(rpm) {
			b.tokens = float64(rpm)
		}
		b.lastFill = now
		if b.tokens >= 1 {
			b.tokens--
			return true
		}
		return false
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		key, ok := strings.CutPrefix(auth, "Bearer ")
		if !ok || key == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="saka"`)
			http.Error(w, `{"error":"missing api key"}`, http.StatusUnauthorized)
			return
		}

		tierName, valid := keys.Lookup(key)
		if !valid {
			http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
			return
		}
		tier, known := tiers[tierName]
		if !known {
			tier = tiers["free"]
		}

		if !allow(key, tier.RPM) {
			w.Header().Set("Retry-After", strconv.Itoa(60/tier.RPM+1))
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(tier.RPM))
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		// Stream gating: cheap check, applied by the server before opening SSE.
		if r.URL.Path == "/v1/stream" && !tier.Stream {
			http.Error(w, `{"error":"streaming requires standard tier"}`, http.StatusPaymentRequired)
			return
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(tier.RPM))
		next.ServeHTTP(w, r)
	})
}

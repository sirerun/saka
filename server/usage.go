// Package server — usage tracking for billing.
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// UsageStats tracks per-key request counts and bytes served.
type UsageStats struct {
	mu    sync.Mutex
	byKey map[string]*KeyUsage
}

// NOTE (source-chat bug, fixed): the chat's version was
// `type KeyUsage struct	Day string ...` — missing the opening `{` after
// `struct` (a dropped character, same pattern as a couple of other spots
// in this v1.2 pass — see NOTES.md). Restored here.
type KeyUsage struct {
	Day       string    `json:"day"` // YYYY-MM-DD bucket
	Searches  int64     `json:"searches"`
	Fetches   int64     `json:"fetches"`
	Streams   int64     `json:"streams"`
	Errors4xx int64     `json:"errors_4xx"`
	Errors5xx int64     `json:"errors_5xx"`
	BytesOut  int64     `json:"bytes_out"`
	LastSeen  time.Time `json:"last_seen"`
}

func NewUsageStats() *UsageStats {
	return &UsageStats{byKey: make(map[string]*KeyUsage)}
}

var _ KeySource = recordingKeySource{} // compile-time: usage wraps KeySource

// recordingKeySource wraps any KeySource, recording usage as requests pass.
//
// Use it by wrapping your real KeySource before handing it to the
// existing AuthMiddleware in auth.go — the source chat's own v1.2 pass
// tried to give AuthMiddleware a second, stats-taking signature instead,
// which would collide with the one in auth.go (Go doesn't allow two
// functions named AuthMiddleware in one package). Wrapping the KeySource
// needs no signature change:
//
//	AuthMiddleware(recordingKeySource{inner: keys, stats: stats}, mux)
type recordingKeySource struct {
	inner KeySource
	stats *UsageStats
}

func (r recordingKeySource) Lookup(key string) (string, bool) {
	tier, ok := r.inner.Lookup(key)
	if ok {
		r.stats.touch(key)
	}
	return tier, ok
}

func (u *UsageStats) touch(key string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	ku, ok := u.byKey[key]
	if !ok || ku.Day != time.Now().Format("2006-01-02") {
		u.byKey[key] = &KeyUsage{Day: time.Now().Format("2006-01-02"), LastSeen: time.Now()}
		return
	}
	ku.LastSeen = time.Now()
}

// Record bumps a counter for a key after a request completes.
func (u *UsageStats) Record(key string, field func(*KeyUsage)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if ku, ok := u.byKey[key]; ok {
		field(ku)
	}
}

// Handler serves GET /v1/usage — the authenticated key sees only itself;
// an admin key (tier "admin") sees all.
func (u *UsageStats) Handler(adminKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		u.mu.Lock()
		defer u.mu.Unlock()

		if key == adminKey && adminKey != "" {
			_ = json.NewEncoder(w).Encode(u.byKey) // full dump for billing jobs
			return
		}
		if ku, ok := u.byKey[key]; ok {
			_ = json.NewEncoder(w).Encode(ku)
			return
		}
		_ = json.NewEncoder(w).Encode(KeyUsage{Day: time.Now().Format("2006-01-02")})
	}
}

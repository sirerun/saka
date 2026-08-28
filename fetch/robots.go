// Package fetch — robots.txt compliance, hand-rolled (no x/robotstxt dep needed).
package fetch

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// robotsCache caches parsed robots rules per host.
type robotsCache struct {
	mu   sync.Mutex
	deny map[string]*hostRules // host -> rules
}

type hostRules struct {
	disallow  []string
	fetchedAt time.Time
	allowAll  bool // no robots.txt or allow-all
}

func newRobotsCache() *robotsCache {
	return &robotsCache{deny: make(map[string]*hostRules)}
}

// Allowed checks whether the agent may fetch the path on this host.
// Results are cached per host for 1 hour.
func (rc *robotsCache) Allowed(ctx context.Context, client *http.Client, agent, rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Host

	rc.mu.Lock()
	hr, ok := rc.deny[host]
	rc.mu.Unlock()

	if !ok || time.Since(hr.fetchedAt) > time.Hour {
		hr = rc.fetchRules(ctx, client, u.Scheme+"://"+host+"/robots.txt", agent)
		rc.mu.Lock()
		rc.deny[host] = hr
		rc.mu.Unlock()
	}

	if hr.allowAll {
		return true
	}
	for _, prefix := range hr.disallow {
		if strings.HasPrefix(u.Path, prefix) {
			return false
		}
	}
	return true
}

func (rc *robotsCache) fetchRules(ctx context.Context, client *http.Client, robotsURL, agent string) *hostRules {
	hr := &hostRules{fetchedAt: time.Now(), allowAll: true}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return hr
	}
	req.Header.Set("User-Agent", agent)
	resp, err := client.Do(req)
	// NOTE (source-chat bug, fixed): the chat's own version called
	// `resp.Body.Close()` unconditionally *before* checking `resp != nil`,
	// which panics on a transport error (nil resp). The v1.0 release
	// checklist that came later in the chat flagged this exact spot
	// ("single nil-safe Body.Close") without showing the fix — this is
	// that single nil-safe close.
	if err != nil || resp == nil {
		return hr // no robots.txt -> allowed
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return hr // no robots.txt -> allowed
	}

	// Minimal robots parser: find group for our agent or "*", collect Disallow.
	const maxBytes = 256 << 10
	body := make([]byte, maxBytes)
	n, _ := resp.Body.Read(body)
	lines := strings.Split(string(body[:n]), "\n")

	inGroup, groupApplies := false, false
	for _, line := range lines {
		line = strings.TrimSpace(strings.ToLower(line))
		switch {
		case strings.HasPrefix(line, "user-agent:"):
			ua := strings.TrimSpace(strings.TrimPrefix(line, "user-agent:"))
			if inGroup && groupApplies {
				// new group starts; reset
				inGroup, groupApplies = false, false
			}
			inGroup = true
			groupApplies = groupApplies || ua == "*" || strings.Contains(agent, ua)
		case strings.HasPrefix(line, "disallow:"):
			if inGroup && groupApplies {
				p := strings.TrimSpace(strings.TrimPrefix(line, "disallow:"))
				if p != "" {
					hr.disallow = append(hr.disallow, p)
				}
			}
		case strings.HasPrefix(line, "sitemap:"), line == "":
			// ignore
		}
	}
	hr.allowAll = len(hr.disallow) == 0
	return hr
}

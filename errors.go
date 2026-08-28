package saka

import "time"

// RateLimitError signals a provider is throttled; triggers chain fallback.
type RateLimitError struct {
	Provider   string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "saka: " + e.Provider + " rate limited"
}

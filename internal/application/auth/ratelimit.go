// Package auth implements authentication application use cases and security rate limiting.
package auth

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	attempts []time.Time
}

// RateLimiter tracks and enforces rate limits in memory per key (e.g. remote IP).
type RateLimiter struct {
	mu          sync.Mutex
	maxAttempts int
	window      time.Duration
	clients     map[string]*rateLimitEntry
}

// NewRateLimiter constructs a new RateLimiter.
func NewRateLimiter(maxAttempts int, window time.Duration) *RateLimiter {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		maxAttempts: maxAttempts,
		window:      window,
		clients:     make(map[string]*rateLimitEntry),
	}
}

// Allow reports whether an attempt for key is permitted under the rate limit.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	entry, exists := r.clients[key]
	if !exists {
		r.clients[key] = &rateLimitEntry{
			attempts: []time.Time{now},
		}
		return true
	}

	// Filter out attempts older than window
	validAttempts := make([]time.Time, 0, len(entry.attempts)+1)
	for _, t := range entry.attempts {
		if t.After(cutoff) {
			validAttempts = append(validAttempts, t)
		}
	}

	if len(validAttempts) >= r.maxAttempts {
		entry.attempts = validAttempts
		return false
	}

	validAttempts = append(validAttempts, now)
	entry.attempts = validAttempts
	return true
}

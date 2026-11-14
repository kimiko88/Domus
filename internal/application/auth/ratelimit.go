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
const maxTrackedRateLimitClients = 10_000

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
	if exists {
		// Filter out attempts older than window.
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
		if len(validAttempts) > 0 {
			entry.attempts = append(validAttempts, now)
			return true
		}
		// Drop idle keys so the map does not grow forever.
		delete(r.clients, key)
	}

	if len(r.clients) >= maxTrackedRateLimitClients {
		// Sweep expired entries only when the bounded table is full.
		for clientKey, client := range r.clients {
			active := false
			for _, attempt := range client.attempts {
				if attempt.After(cutoff) {
					active = true
					break
				}
			}
			if !active {
				delete(r.clients, clientKey)
			}
		}
		// Do not evict active entries: doing so would let an attacker reset
		// their own limit by exhausting the key table with spoofed clients.
		if len(r.clients) >= maxTrackedRateLimitClients {
			return false
		}
	}

	r.clients[key] = &rateLimitEntry{attempts: []time.Time{now}}
	return true
}

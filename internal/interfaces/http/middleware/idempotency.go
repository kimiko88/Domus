// Package middleware provides HTTP middlewares for Domus.
package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// IdempotentResponse caches the HTTP response for an executed mutation.
type IdempotentResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	ExpiresAt  time.Time
	InProgress bool
}

// IdempotencyStore manages thread-safe caching of idempotency keys.
type IdempotencyStore struct {
	mu      sync.RWMutex
	cache   map[string]*IdempotentResponse
	ttl     time.Duration
	stopCh  chan struct{}
	stopped bool
}

// NewIdempotencyStore creates an in-memory idempotency cache with an active background purger.
func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	s := &IdempotencyStore{
		cache:  make(map[string]*IdempotentResponse),
		ttl:    ttl,
		stopCh: make(chan struct{}),
	}

	go s.purgeLoop()
	return s
}

func (s *IdempotencyStore) purgeLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, v := range s.cache {
				if now.After(v.ExpiresAt) {
					delete(s.cache, k)
				}
			}
			s.mu.Unlock()
		}
	}
}

// Close stops the background purger.
func (s *IdempotencyStore) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		s.stopped = true
		close(s.stopCh)
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	rec.body.Write(b)
	return rec.ResponseWriter.Write(b)
}

// Idempotency returns a middleware that guarantees at-most-once execution of mutating requests.
// When an X-Idempotency-Key or Idempotency-Key header is present on POST, PUT, PATCH, or DELETE,
// repeated requests return the cached response without re-executing the operation.
func Idempotency(store *IdempotencyStore) func(http.Handler) http.Handler {
	if store == nil {
		store = NewIdempotencyStore(60 * time.Second)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Idempotency keys are only meaningful for mutating HTTP methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("X-Idempotency-Key")
			if key == "" {
				key = r.Header.Get("Idempotency-Key")
			}
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check existing entry
			store.mu.Lock()
			entry, exists := store.cache[key]
			now := time.Now()

			if exists {
				if now.Before(entry.ExpiresAt) {
					if entry.InProgress {
						store.mu.Unlock()
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusConflict)
						_ = json.NewEncoder(w).Encode(map[string]any{
							"error":   "request_in_progress",
							"message": "A mutation with this idempotency key is currently processing",
						})
						return
					}

					// Return cached response
					store.mu.Unlock()
					for h, vals := range entry.Header {
						for _, v := range vals {
							w.Header().Add(h, v)
						}
					}
					w.Header().Set("X-Cache-Lookup", "HIT-IDEMPOTENT")
					w.WriteHeader(entry.StatusCode)
					_, _ = w.Write(entry.Body)
					return
				}
				// Expired entry, re-use key
				delete(store.cache, key)
			}

			// Register as in-progress
			store.cache[key] = &IdempotentResponse{
				InProgress: true,
				ExpiresAt:  now.Add(store.ttl),
			}
			store.mu.Unlock()

			// Intercept response
			rec := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // default if WriteHeader is not called
			}

			next.ServeHTTP(rec, r)

			// Store completed response
			store.mu.Lock()
			defer store.mu.Unlock()

			// Clone headers for clean reproduction
			clonedHeader := make(http.Header)
			for h, vals := range rec.Header() {
				clonedHeader[h] = append([]string(nil), vals...)
			}

			store.cache[key] = &IdempotentResponse{
				StatusCode: rec.statusCode,
				Header:     clonedHeader,
				Body:       rec.body.Bytes(),
				ExpiresAt:  time.Now().Add(store.ttl),
				InProgress: false,
			}
		})
	}
}

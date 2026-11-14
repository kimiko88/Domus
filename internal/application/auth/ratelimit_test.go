// Package auth_test tests the login rate limiter and auth application service (TDD red-to-green phase).
package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/auth"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	// 3 attempts per 100ms
	limiter := auth.NewRateLimiter(3, 100*time.Millisecond)

	ip := "192.168.1.10"

	assert.True(t, limiter.Allow(ip), "1st attempt should be allowed")
	assert.True(t, limiter.Allow(ip), "2nd attempt should be allowed")
	assert.True(t, limiter.Allow(ip), "3rd attempt should be allowed")
	assert.False(t, limiter.Allow(ip), "4th attempt should be blocked")

	// Other IP should not be blocked
	assert.True(t, limiter.Allow("192.168.1.20"), "Different IP should be allowed")

	// Wait for window to expire
	time.Sleep(120 * time.Millisecond)
	assert.True(t, limiter.Allow(ip), "Attempt should be allowed after window expires")
}


func TestRateLimiterBoundsTrackedClientMemory(t *testing.T) {
	limiter := auth.NewRateLimiter(1, time.Hour)
	for i := 0; i < 10_000; i++ {
		assert.True(t, limiter.Allow(fmt.Sprintf("192.0.2.%d", i)), "initial unique key %d should be allowed", i)
	}
	assert.False(t, limiter.Allow("198.51.100.250"), "new keys must be rejected rather than evicting active limits")
}

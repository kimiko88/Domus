package supervisor

import (
	"context"
	"net/http"
	"time"
)

// HTTPHealthChecker performs HTTP GET requests to check service readiness.
type HTTPHealthChecker struct {
	client *http.Client
}

// NewHTTPHealthChecker constructs a checker with a standard timeout.
func NewHTTPHealthChecker(timeout time.Duration) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Check sends a GET request to the specified URL; returns true if status is 2xx.
func (h *HTTPHealthChecker) Check(ctx context.Context, url string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	return false, nil
}

package e2e_test

import (
	"context"
	"testing"
	"time"

	serverpkg "github.com/domus-hub/domus/internal/interfaces/http"
)

// shutdownServer releases background adapters and listeners created by NewServer.
// httptest.Server.Close only closes the test HTTP listener; it does not stop the
// application-owned LAN discovery and event-bus workers.
func shutdownServer(t *testing.T, srv *serverpkg.Server) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shut down Domus server: %v", err)
	}
}

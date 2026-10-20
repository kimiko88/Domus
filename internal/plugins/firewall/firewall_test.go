package firewall_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/domus-hub/domus/internal/plugins/firewall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirewall_MockProvider(t *testing.T) {
	client := firewall.NewClient(firewall.Config{
		Provider: firewall.ProviderMock,
	})
	ctx := context.Background()

	// 1. Isolate device
	err := client.IsolateClient(ctx, "AA-BB-CC-11-22-33", "192.168.1.150", "Unauthorized DNS query to foreign WAN IP")
	require.NoError(t, err)

	// 2. List blocked
	blocked, err := client.ListBlockedClients(ctx)
	require.NoError(t, err)
	require.Len(t, blocked, 1)
	assert.Equal(t, "aa:bb:cc:11:22:33", blocked[0].MAC)
	assert.Equal(t, "192.168.1.150", blocked[0].IP)
	assert.Contains(t, blocked[0].Reason, "Unauthorized DNS")

	// 3. Status
	st, err := client.GetStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, st["blocked_count"])

	// 4. Unblock
	err = client.UnblockClient(ctx, "aa:bb:cc:11:22:33", "192.168.1.150")
	require.NoError(t, err)

	blockedAfter, _ := client.ListBlockedClients(ctx)
	assert.Empty(t, blockedAfter)
}

func TestFirewall_UniFiIntegration(t *testing.T) {
	var capturedCmd string
	var capturedMAC string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/s/home/cmd/stamgr", r.URL.Path)
		assert.Equal(t, "Bearer unifi-secret-token", r.Header.Get("Authorization"))

		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		capturedCmd, _ = payload["cmd"].(string)
		capturedMAC, _ = payload["mac"].(string)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"}}`))
	}))
	defer server.Close()

	client := firewall.NewClient(firewall.Config{
		Provider: firewall.ProviderUniFi,
		BaseURL:  server.URL,
		Site:     "home",
		APIKey:   "unifi-secret-token",
	})

	err := client.IsolateClient(context.Background(), "44-55-66-77-88-99", "192.168.1.80", "WAN leak")
	require.NoError(t, err)
	assert.Equal(t, "block-sta", capturedCmd)
	assert.Equal(t, "44:55:66:77:88:99", capturedMAC)
}

func TestFirewall_OPNsenseIntegration(t *testing.T) {
	var capturedEndpoint string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedEndpoint = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"saved"}`))
	}))
	defer server.Close()

	client := firewall.NewClient(firewall.Config{
		Provider: firewall.ProviderOPNsense,
		BaseURL:  server.URL,
		APIKey:   "opn-key-123",
	})

	err := client.IsolateClient(context.Background(), "11:22:33:44:55:66", "192.168.1.99", "China Cloud traffic")
	require.NoError(t, err)
	assert.Equal(t, "/api/firewall/alias_util/add/domus_quarantine", capturedEndpoint)
}

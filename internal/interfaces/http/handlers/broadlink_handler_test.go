package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apppluginruntime "github.com/domus-hub/domus/internal/application/plugin_runtime"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/plugins/broadlink"
	"github.com/stretchr/testify/require"
)

func TestBroadlinkHandlerUsesCurrentManagedAdapterAndHonorsDisable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	db, err := sqlite.Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1})
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, sqlite.Migrate(ctx, db))
	cipher, err := crypto.NewEnvelopeCipher([]byte("broadlink-handler-test-key"))
	require.NoError(t, err)
	repos := sqlite.NewRepositories(db)
	repos.Cipher = cipher
	manager, err := apppluginruntime.NewManager(repos, nil)
	require.NoError(t, err)

	_, err = manager.Create(ctx, broadlink.DefaultIntegrationID, broadlink.PluginType, "BroadLink Local", []byte(`{}`), true)
	require.NoError(t, err)
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop(context.Background())

	// Passing an old adapter pointer must not make it usable once manager mode
	// has been selected.
	legacyPointer := broadlink.NewAdapter(broadlink.DefaultIntegrationID, nil)
	handler := NewBroadlinkHandler(legacyPointer).WithPluginManager(manager)

	request := httptest.NewRequest(http.MethodGet, "/devices", nil)
	response := httptest.NewRecorder()
	handler.ListDevices(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	require.NoError(t, manager.SetEnabled(ctx, broadlink.DefaultIntegrationID, false))
	request = httptest.NewRequest(http.MethodGet, "/devices", nil)
	response = httptest.NewRecorder()
	handler.ListDevices(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Contains(t, response.Body.String(), "BROADLINK_UNAVAILABLE")
}

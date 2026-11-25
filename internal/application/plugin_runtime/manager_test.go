package plugin_runtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/integration"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/crypto"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/require"
)

type memoryIntegrationRepository struct { mu sync.RWMutex; items map[string]*integration.Integration }
func newMemoryIntegrationRepository() *memoryIntegrationRepository { return &memoryIntegrationRepository{items: make(map[string]*integration.Integration)} }
func (r *memoryIntegrationRepository) Create(ctx context.Context, in *integration.Integration) error { r.mu.Lock(); defer r.mu.Unlock(); if _, exists := r.items[in.ID]; exists { return errors.New("integration already exists") }; copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); r.items[in.ID] = &copy; return nil }
func (r *memoryIntegrationRepository) GetByID(ctx context.Context, id string) (*integration.Integration, error) { r.mu.RLock(); defer r.mu.RUnlock(); in, ok := r.items[id]; if !ok { return nil, integration.ErrIntegrationNotFound }; copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); return &copy, nil }
func (r *memoryIntegrationRepository) List(ctx context.Context) ([]*integration.Integration, error) { r.mu.RLock(); defer r.mu.RUnlock(); result := make([]*integration.Integration, 0, len(r.items)); for _, in := range r.items { copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); result = append(result, &copy) }; return result, nil }
func (r *memoryIntegrationRepository) Update(ctx context.Context, in *integration.Integration) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.items[in.ID]; !ok { return integration.ErrIntegrationNotFound }; copy := *in; copy.ConfigEncrypted = append([]byte(nil), in.ConfigEncrypted...); r.items[in.ID] = &copy; return nil }
func (r *memoryIntegrationRepository) SetStatus(ctx context.Context, id, status, lastError string) error { r.mu.Lock(); defer r.mu.Unlock(); in, ok := r.items[id]; if !ok { return integration.ErrIntegrationNotFound }; in.Status, in.LastError = status, lastError; return nil }
func (r *memoryIntegrationRepository) Delete(ctx context.Context, id string) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.items[id]; !ok { return integration.ErrIntegrationNotFound }; delete(r.items, id); return nil }

func newTestManager(t *testing.T) (*Manager, *memoryIntegrationRepository, *crypto.EnvelopeCipher) {
	t.Helper()
	repo := newMemoryIntegrationRepository()
	cipher, err := crypto.NewEnvelopeCipher([]byte("plugin-runtime-test-key"))
	require.NoError(t, err)
	db, err := sqlite.Open(config.DatabaseConfig{Path: ":memory:", WALMode: false, BusyTimeoutMs: 1000, MaxOpenConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.Migrate(context.Background(), db))
	repos := sqlite.NewRepositories(db)
	repos.Integrations = repo
	repos.Cipher = cipher
	manager, err := NewManager(repos, nil)
	require.NoError(t, err)
	return manager, repo, cipher
}

func TestManagerPersistsEncryptedConfigAndManagesPluginLifecycle(t *testing.T) {
	manager, repo, cipher := newTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second); defer cancel()
	require.NoError(t, manager.Start(ctx)); defer manager.Stop(context.Background())
	instance, err := manager.Create(ctx, "matter_main", "matter", "Matter Controller", json.RawMessage(`{}`), true); require.NoError(t, err)
	require.Equal(t, "matter_main", instance.ID); require.True(t, instance.Enabled); require.Equal(t, "connected", instance.Status)
	stored, err := repo.GetByID(ctx, "matter_main"); require.NoError(t, err); require.NotContains(t, string(stored.ConfigEncrypted), "device_secret")
	plain, err := cipher.Decrypt(string(stored.ConfigEncrypted)); require.NoError(t, err); require.JSONEq(t, `{}`, plain)
	require.NoError(t, manager.Restart(ctx, "matter_main"))
	require.NoError(t, manager.SetEnabled(ctx, "matter_main", false))
	instances, err := manager.ListInstances(ctx); require.NoError(t, err); require.Len(t, instances, 1); require.False(t, instances[0].Enabled); require.Equal(t, "disconnected", instances[0].Status)
	require.NoError(t, manager.SetEnabled(ctx, "matter_main", true))
	instances, err = manager.ListInstances(ctx); require.NoError(t, err); require.Equal(t, "connected", instances[0].Status)
	require.NoError(t, manager.Delete(ctx, "matter_main")); _, err = repo.GetByID(ctx, "matter_main"); require.ErrorIs(t, err, integration.ErrIntegrationNotFound)
}

func TestManagerRestoresEnabledInstancesAfterRestartWithoutExposingSecrets(t *testing.T) {
	manager, repo, cipher := newTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second); defer cancel()
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/devices":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"devices":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer bridge.Close()
	secretConfig := `{"bridge_url":"` + bridge.URL + `","token":"top-secret-plugin-token-0123456789abcdef","poll_interval":"60s"}`
	encrypted, err := cipher.Encrypt(secretConfig); require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, &integration.Integration{ID:"open_display", Type:"open_display", Name:"OpenDisplay", Enabled:true, ConfigEncrypted:[]byte(encrypted), Status:"disconnected", CreatedAt:time.Now().UTC(), UpdatedAt:time.Now().UTC()}))
	require.NoError(t, manager.Start(ctx)); defer manager.Stop(context.Background())
	instances, err := manager.ListInstances(ctx); require.NoError(t, err); require.Len(t, instances, 1); require.Equal(t, "connected", instances[0].Status)
	encoded, err := json.Marshal(instances); require.NoError(t, err); require.NotContains(t, string(encoded), "top-secret-plugin-token-0123456789abcdef"); require.NotContains(t, string(encoded), "token")
}

func TestManagerRejectsDuplicateAndUnknownPluginInstances(t *testing.T) {
	manager, _, _ := newTestManager(t); ctx := context.Background()
	_, err := manager.Create(ctx, "missing_one", "not_a_plugin", "Unknown", json.RawMessage(`{}`), false); require.Error(t, err)
	_, err = manager.Create(ctx, "matter_one", "matter", "Matter One", json.RawMessage(`{}`), false); require.NoError(t, err)
	_, err = manager.Create(ctx, "matter_one", "matter", "Duplicate", json.RawMessage(`{}`), false); require.Error(t, err)
}

func TestManagerEnforcesDeclaredInstanceMultiplicity(t *testing.T) {
	manager, _, _ := newTestManager(t)
	ctx := context.Background()
	_, err := manager.Create(ctx, "matter_primary", "matter", "Matter Primary", json.RawMessage(`{}`), false)
	require.NoError(t, err)
	_, err = manager.Create(ctx, "matter_secondary", "matter", "Matter Secondary", json.RawMessage(`{}`), false)
	require.ErrorIs(t, err, ErrSingleInstance)
	knxA := json.RawMessage(`{"gateway_addr":"192.168.1.10:3671","entities":[]}`)
	knxB := json.RawMessage(`{"gateway_addr":"192.168.1.11:3671","entities":[]}`)
	_, err = manager.Create(ctx, "knx_first_floor", "knx", "KNX First Floor", knxA, false)
	require.NoError(t, err)
	_, err = manager.Create(ctx, "knx_second_floor", "knx", "KNX Second Floor", knxB, false)
	require.NoError(t, err)
	voiceConfig := json.RawMessage(`{"server_url":"http://127.0.0.1:8080","bearer_token":"voice-satellite-test-token-0123456789abcdef","wake_address":"127.0.0.1:10400","wake_words":["hey_jarvis"]}`)
	_, err = manager.Create(ctx, "voice_kitchen", "voice_satellite", "Voice Kitchen", voiceConfig, false)
	require.NoError(t, err)
	_, err = manager.Create(ctx, "voice_bedroom", "voice_satellite", "Voice Bedroom", voiceConfig, false)
	require.NoError(t, err)
	instances, err := manager.ListInstances(ctx)
	require.NoError(t, err)
	require.Len(t, instances, 5)
}


func TestManagerStartsPluginsPreparedAfterRuntimeStartup(t *testing.T) {
	manager, _, _ := newTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop(context.Background())

	adapter, err := manager.Prepare(ctx, "weather_openmeteo", "weather_forecast", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Equal(t, "weather_forecast", adapter.ID())
	require.NoError(t, adapter.HealthCheck(ctx))
}

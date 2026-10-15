package notification_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	appNotification "github.com/domus-hub/domus/internal/application/notification"
	domainNotification "github.com/domus-hub/domus/internal/domain/notification"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/eventbus"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushService_LifecycleAndSend(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_push_service.db")

	db, err := sqlite.Open(config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	})
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewPushTokenRepository(db)
	sender := appNotification.NewInMemoryPushSender()
	bus := eventbus.NewMemoryBus(eventbus.Config{BufferSize: 100, Workers: 2})

	service := appNotification.NewPushService(repo, sender, bus, nil)

	// 1. Validation errors
	err = service.RegisterToken(ctx, &domainNotification.DevicePushToken{
		DeviceToken: "",
		UserID:      "u1",
		Platform:    domainNotification.PlatformIOS,
	})
	assert.Error(t, err)

	err = service.RegisterToken(ctx, &domainNotification.DevicePushToken{
		DeviceToken: "tok1",
		UserID:      "",
		Platform:    domainNotification.PlatformIOS,
	})
	assert.Error(t, err)

	err = service.RegisterToken(ctx, &domainNotification.DevicePushToken{
		DeviceToken: "tok1",
		UserID:      "u1",
		Platform:    "windows_phone",
	})
	assert.Error(t, err)

	// 2. Register iOS and Android tokens
	require.NoError(t, service.RegisterToken(ctx, &domainNotification.DevicePushToken{
		DeviceToken: "ios_token_1",
		UserID:      "alice",
		Platform:    domainNotification.PlatformIOS,
		DeviceName:  "iPhone 15",
	}))

	require.NoError(t, service.RegisterToken(ctx, &domainNotification.DevicePushToken{
		DeviceToken: "android_token_1",
		UserID:      "bob",
		Platform:    domainNotification.PlatformAndroid,
		DeviceName:  "Pixel 8",
	}))

	// 3. Send critical push to Alice only
	report, err := service.Send(ctx, domainNotification.PushMessage{
		UserID:   "alice",
		Title:    "Gas Leak Detected!",
		Body:     "Kitchen methane sensor alarm triggered",
		Critical: true,
		Volume:   0.9,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, report.TotalTokens)
	assert.Equal(t, 1, report.Successes)
	assert.Equal(t, 0, report.Failures)

	require.Len(t, sender.Dispatches, 1)
	apnsDispatch := sender.Dispatches[0]
	assert.Equal(t, "ios", apnsDispatch["platform"])
	assert.Equal(t, "ios_token_1", apnsDispatch["token"])

	payload := apnsDispatch["payload"].(map[string]any)
	aps := payload["aps"].(map[string]any)
	assert.Equal(t, "critical", aps["interruption-level"])
	sound := aps["sound"].(map[string]any)
	assert.Equal(t, 1, sound["critical"])
	assert.Equal(t, 0.9, sound["volume"])

	// 4. Send non-critical broadcast to all users
	reportBroadcast, err := service.Send(ctx, domainNotification.PushMessage{
		Title:    "Laundry Done",
		Body:     "Washing machine cycle completed",
		Critical: false,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, reportBroadcast.TotalTokens)
	assert.Equal(t, 2, reportBroadcast.Successes)

	// Check Android dispatch
	var androidFound bool
	for _, d := range sender.Dispatches {
		if d["platform"] == "android" {
			androidFound = true
			androidPayload := d["payload"].(map[string]any)
			assert.Equal(t, "normal", androidPayload["priority"])
			dataMap := androidPayload["data"].(map[string]any)
			assert.Equal(t, "Laundry Done", dataMap["title"])
		}
	}
	assert.True(t, androidFound)

	// 5. Expired token handling
	sender.FailOnTokens["ios_token_1"] = appNotification.ErrInvalidToken

	reportFail, err := service.Send(ctx, domainNotification.PushMessage{
		UserID: "alice",
		Title:  "Ping",
		Body:   "Check",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, reportFail.TotalTokens)
	assert.Equal(t, 0, reportFail.Successes)
	assert.Equal(t, 1, reportFail.Failures)

	// Verify ios_token_1 was automatically pruned from repository
	aliceTokens, err := service.GetTokensByUser(ctx, "alice")
	require.NoError(t, err)
	assert.Empty(t, aliceTokens)
}

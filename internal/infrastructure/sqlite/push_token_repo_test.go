package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/domain/notification"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushTokenRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_push.db")

	cfg := config.DatabaseConfig{
		Path:          dbPath,
		WALMode:       true,
		BusyTimeoutMs: 5000,
		MaxOpenConns:  1,
	}

	db, err := sqlite.Open(cfg)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewPushTokenRepository(db)

	token1 := &notification.DevicePushToken{
		UserID:      "user-1",
		Platform:    notification.PlatformIOS,
		DeviceToken: "apns_token_abc_123",
		DeviceName:  "iPhone 16 Pro",
		AppVersion:  "1.0.0",
	}

	token2 := &notification.DevicePushToken{
		UserID:      "user-1",
		Platform:    notification.PlatformAndroid,
		DeviceToken: "fcm_token_xyz_789",
		DeviceName:  "Pixel 9 Pro",
		AppVersion:  "1.0.0",
	}

	token3 := &notification.DevicePushToken{
		UserID:      "user-2",
		Platform:    notification.PlatformAndroid,
		DeviceToken: "fcm_token_s24_456",
		DeviceName:  "Galaxy S24",
		AppVersion:  "1.0.0",
	}

	// 1. Register tokens
	require.NoError(t, repo.RegisterToken(ctx, token1))
	require.NoError(t, repo.RegisterToken(ctx, token2))
	require.NoError(t, repo.RegisterToken(ctx, token3))

	// 2. Query user 1 tokens
	user1Tokens, err := repo.GetTokensByUser(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, user1Tokens, 2)

	// 3. Update token1 (upsert on device_token)
	token1Updated := &notification.DevicePushToken{
		UserID:      "user-1",
		Platform:    notification.PlatformIOS,
		DeviceToken: "apns_token_abc_123",
		DeviceName:  "iPhone 16 Pro Max",
		AppVersion:  "1.0.1",
	}
	require.NoError(t, repo.RegisterToken(ctx, token1Updated))

	user1Tokens, err = repo.GetTokensByUser(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, user1Tokens, 2)
	var foundUpdated bool
	for _, tk := range user1Tokens {
		if tk.DeviceToken == "apns_token_abc_123" {
			assert.Equal(t, "iPhone 16 Pro Max", tk.DeviceName)
			assert.Equal(t, "1.0.1", tk.AppVersion)
			foundUpdated = true
		}
	}
	assert.True(t, foundUpdated)

	// 4. GetAllTokens
	allTokens, err := repo.GetAllTokens(ctx)
	require.NoError(t, err)
	assert.Len(t, allTokens, 3)

	// 5. Revoke user-1's pixel token
	require.NoError(t, repo.RevokeToken(ctx, "fcm_token_xyz_789"))
	user1TokensAfterRevoke, err := repo.GetTokensByUser(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, user1TokensAfterRevoke, 1)
	assert.Equal(t, "apns_token_abc_123", user1TokensAfterRevoke[0].DeviceToken)

	// 6. Delete token by token string
	require.NoError(t, repo.DeleteToken(ctx, "fcm_token_s24_456"))
	user2Tokens, err := repo.GetTokensByUser(ctx, "user-2")
	require.NoError(t, err)
	assert.Empty(t, user2Tokens)
}

package sqlite_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	domauth "github.com/domus-hub/domus/internal/domain/auth"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
	"github.com/domus-hub/domus/internal/domain/user"
	"github.com/stretchr/testify/require"
)

func TestConsumeBackupCodeIsOneTime(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: filepath.Join(t.TempDir(), "two_factor.db"), WALMode: true,
		BusyTimeoutMs: 5000, MaxOpenConns: 1,
	})
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repos := sqlite.NewRepositories(db)
	now := time.Now().UTC()
	require.NoError(t, repos.Users.Create(ctx, &user.User{
		ID: "user-1", Username: "two_factor_user", PasswordHash: "test-hash",
		Role: user.RoleAdmin, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}))
	repo := sqlite.NewTwoFactorRepository(db)
	require.NoError(t, repo.Save(ctx, &domauth.TwoFactorConfig{
		UserID: "user-1", Secret: "JBSWY3DPEHPK3PXP", IsEnabled: true,
		BackupCodes: []string{"ABCD-EFGH", "JKLM-NPQR"},
	}))
	consumed, err := repo.ConsumeBackupCode(ctx, "user-1", " abcd-efgh ")
	require.NoError(t, err)
	require.True(t, consumed)
	consumed, err = repo.ConsumeBackupCode(ctx, "user-1", "ABCD-EFGH")
	require.NoError(t, err)
	require.False(t, consumed)
	cfg, err := repo.Get(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, []string{"JKLM-NPQR"}, cfg.BackupCodes)
	require.WithinDuration(t, time.Now(), cfg.UpdatedAt, time.Minute)
}


func TestBannedIPCountsConcurrentFailuresAtomically(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(config.DatabaseConfig{
		Path: filepath.Join(t.TempDir(), "banned_ip.db"), WALMode: true,
		BusyTimeoutMs: 10000, MaxOpenConns: 16,
	})
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, sqlite.Migrate(ctx, db))

	repo := sqlite.NewBannedIPRepository(db)
	const attempts = 32
	var wg sync.WaitGroup
	errCh := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, callErr := repo.RecordFailedAttempt(ctx, "192.0.2.10", 5, 15*time.Minute)
			errCh <- callErr
		}()
	}
	wg.Wait()
	close(errCh)
	for callErr := range errCh {
		require.NoError(t, callErr)
	}

	record, err := repo.Get(ctx, "192.0.2.10")
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Equal(t, attempts, record.FailedAttempts)
	require.True(t, record.IsBanned())
}

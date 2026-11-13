package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	domauth "github.com/domus-hub/domus/internal/domain/auth"
	"github.com/domus-hub/domus/internal/infrastructure/config"
	"github.com/domus-hub/domus/internal/infrastructure/sqlite"
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

	repo := sqlite.NewTwoFactorRepository(db)
	require.NoError(t, repo.Save(ctx, &domauth.TwoFactorConfig{
		UserID: "user-1", Secret: "JBSWY3DPEHPK3PXP", IsEnabled: true,
		BackupCodes: []string{"ABCD-EFGH", "JKLM-NPQR"},
	}))
	consumed, err := repo.ConsumeBackupCode(ctx, "user-1", "ABCD-EFGH")
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

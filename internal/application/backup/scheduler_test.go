package backup_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/backup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_RunOnceAndSelectiveRestore(t *testing.T) {
	tempDir := t.TempDir()

	dbFile := filepath.Join(tempDir, "domus.db")
	require.NoError(t, os.WriteFile(dbFile, []byte("SQLITE_DATABASE_TEST_DATA"), 0644))

	configFile := filepath.Join(tempDir, "domus.toml")
	require.NoError(t, os.WriteFile(configFile, []byte("port = 8080\nname = 'TestDomus'"), 0644))

	backupDir := filepath.Join(tempDir, "backups")
	require.NoError(t, os.MkdirAll(backupDir, 0755))

	offsiteDir := filepath.Join(tempDir, "nas_share")
	require.NoError(t, os.MkdirAll(offsiteDir, 0755))

	service := backup.NewService()

	cfg := backup.SchedulerConfig{
		Enabled:       true,
		Interval:      1 * time.Hour,
		DBPath:        dbFile,
		ConfigPath:    configFile,
		OutputDir:     backupDir,
		Passphrase:    "SecretMasterPassphrase123!",
		RetentionDays: 3,
		Offsite: &backup.OffsiteDestination{
			Type: backup.DestLocalPath,
			Path: offsiteDir,
		},
	}

	scheduler := backup.NewScheduler(service, cfg, nil, nil)
	ctx := context.Background()

	// 1. Run scheduled backup
	res, err := scheduler.RunOnce(ctx)
	require.NoError(t, err)
	assert.True(t, res.Encrypted)
	assert.FileExists(t, res.FilePath)

	// Verify offsite export
	exportedFile := filepath.Join(offsiteDir, filepath.Base(res.FilePath))
	assert.FileExists(t, exportedFile)

	// 2. Selective restore test: restore ONLY domus.db
	restoreDir := filepath.Join(tempDir, "restored_db_only")
	require.NoError(t, service.Restore(backup.RestoreOptions{
		ArchivePath:    res.FilePath,
		OutputDir:      restoreDir,
		Passphrase:     "SecretMasterPassphrase123!",
		SelectiveFiles: []string{"domus.db"},
	}))

	assert.FileExists(t, filepath.Join(restoreDir, "domus.db"))
	assert.NoFileExists(t, filepath.Join(restoreDir, "domus.toml"))

	dbContent, err := os.ReadFile(filepath.Join(restoreDir, "domus.db"))
	require.NoError(t, err)
	assert.Equal(t, "SQLITE_DATABASE_TEST_DATA", string(dbContent))
}

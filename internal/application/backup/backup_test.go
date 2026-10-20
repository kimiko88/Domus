// Package backup — TDD RED: backup and restore use case tests.
package backup_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/domus-hub/domus/internal/application/backup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func makeTempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "domus.db")
	require.NoError(t, os.WriteFile(dbPath, []byte("SQLite magic bytes here"), 0o600))
	return dbPath
}

func makeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "domus.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))
	return cfgPath
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestCreateBackup_PlainArchiveContainsDBAndConfig(t *testing.T) {
	dbPath := makeTempDB(t)
	cfgPath := makeTempConfig(t, `[server]\nport = 8080`)
	outDir := t.TempDir()

	svc := backup.NewService()
	result, err := svc.Create(backup.CreateOptions{
		DBPath:     dbPath,
		ConfigPath: cfgPath,
		OutputDir:  outDir,
		Passphrase: "", // plain (no encryption)
	})
	require.NoError(t, err)
	assert.FileExists(t, result.FilePath)
	assert.True(t, strings.HasSuffix(result.FilePath, ".tar.gz"))
	assert.Greater(t, result.SizeBytes, int64(0))

	// Verify archive contents
	names := listTarGzFiles(t, result.FilePath)
	assert.Contains(t, names, "domus.db")
	assert.Contains(t, names, "domus.toml")
	assert.Contains(t, names, "backup_manifest.json")
}

func TestCreateBackup_EncryptedArchiveIsOpaque(t *testing.T) {
	dbPath := makeTempDB(t)
	cfgPath := makeTempConfig(t, `[server]\nport = 8080`)
	outDir := t.TempDir()

	svc := backup.NewService()
	result, err := svc.Create(backup.CreateOptions{
		DBPath:     dbPath,
		ConfigPath: cfgPath,
		OutputDir:  outDir,
		Passphrase: "strong-passphrase",
	})
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(result.FilePath, ".tar.gz.enc"))
	assert.True(t, result.Encrypted)

	// Encrypted file must not be a valid gzip
	f, err := os.Open(result.FilePath)
	require.NoError(t, err)
	defer f.Close()
	_, err = gzip.NewReader(f)
	assert.Error(t, err, "encrypted file must not be a plain gzip")
}

func TestRestoreBackup_PlainRoundTrip(t *testing.T) {
	dbPath := makeTempDB(t)
	cfgPath := makeTempConfig(t, `[server]\nport = 8080`)
	outDir := t.TempDir()

	svc := backup.NewService()
	result, err := svc.Create(backup.CreateOptions{
		DBPath: dbPath, ConfigPath: cfgPath, OutputDir: outDir,
	})
	require.NoError(t, err)

	restoreDir := t.TempDir()
	err = svc.Restore(backup.RestoreOptions{
		ArchivePath: result.FilePath,
		OutputDir:   restoreDir,
	})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(restoreDir, "domus.db"))
	assert.FileExists(t, filepath.Join(restoreDir, "domus.toml"))
}

func TestRestoreBackup_EncryptedRoundTrip(t *testing.T) {
	dbPath := makeTempDB(t)
	cfgPath := makeTempConfig(t, `[server]\nport = 8080`)
	outDir := t.TempDir()

	svc := backup.NewService()
	result, err := svc.Create(backup.CreateOptions{
		DBPath: dbPath, ConfigPath: cfgPath, OutputDir: outDir, Passphrase: "my-pass",
	})
	require.NoError(t, err)

	restoreDir := t.TempDir()
	err = svc.Restore(backup.RestoreOptions{
		ArchivePath: result.FilePath,
		OutputDir:   restoreDir,
		Passphrase:  "my-pass",
	})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(restoreDir, "domus.db"))
}

func TestRestoreBackup_WrongPassphraseFails(t *testing.T) {
	dbPath := makeTempDB(t)
	cfgPath := makeTempConfig(t, `[server]\nport = 8080`)
	outDir := t.TempDir()

	svc := backup.NewService()
	result, err := svc.Create(backup.CreateOptions{
		DBPath: dbPath, ConfigPath: cfgPath, OutputDir: outDir, Passphrase: "correct",
	})
	require.NoError(t, err)

	restoreDir := t.TempDir()
	err = svc.Restore(backup.RestoreOptions{
		ArchivePath: result.FilePath,
		OutputDir:   restoreDir,
		Passphrase:  "wrong",
	})
	assert.Error(t, err)
}

func TestCreateBackup_ManifestHasTimestamp(t *testing.T) {
	dbPath := makeTempDB(t)
	cfgPath := makeTempConfig(t, "")
	outDir := t.TempDir()

	svc := backup.NewService()
	before := time.Now().UTC()
	result, err := svc.Create(backup.CreateOptions{DBPath: dbPath, ConfigPath: cfgPath, OutputDir: outDir})
	require.NoError(t, err)

	content := readTarGzFile(t, result.FilePath, "backup_manifest.json")
	assert.NotEmpty(t, content)
	assert.True(t, result.CreatedAt.After(before) || result.CreatedAt.Equal(before))
}

// ─── archive helpers ──────────────────────────────────────────────────────────

func listTarGzFiles(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()

	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, filepath.Base(hdr.Name))
	}
	return names
}

func readTarGzFile(t *testing.T, archivePath, name string) string {
	t.Helper()
	f, err := os.Open(archivePath)
	require.NoError(t, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if filepath.Base(hdr.Name) == name {
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, tr)
			return buf.String()
		}
	}
	t.Fatalf("file %q not found in archive %q", name, archivePath)
	return ""
}

func TestVerifyArchive_ValidEncryptedAndUnencrypted(t *testing.T) {
	dbPath := makeTempDB(t)
	cfgPath := makeTempConfig(t, "")
	outDir := t.TempDir()
	svc := backup.NewService()

	// 1. Unencrypted archive
	resPlain, err := svc.Create(backup.CreateOptions{
		DBPath:     dbPath,
		ConfigPath: cfgPath,
		OutputDir:  outDir,
	})
	require.NoError(t, err)

	vPlain, err := svc.VerifyArchive(resPlain.FilePath, "")
	require.NoError(t, err)
	assert.True(t, vPlain.Valid)
	assert.False(t, vPlain.Encrypted)
	assert.Contains(t, vPlain.Files, "domus.db")
	assert.Contains(t, vPlain.Files, "backup_manifest.json")

	// 2. Encrypted archive with passphrase
	resEnc, err := svc.Create(backup.CreateOptions{
		DBPath:     dbPath,
		ConfigPath: cfgPath,
		OutputDir:  outDir,
		Passphrase: "my-secure-vault-passphrase",
	})
	require.NoError(t, err)

	vEnc, err := svc.VerifyArchive(resEnc.FilePath, "my-secure-vault-passphrase")
	require.NoError(t, err)
	assert.True(t, vEnc.Valid)
	assert.True(t, vEnc.Encrypted)
	assert.Contains(t, vEnc.Files, "domus.db")

	// 3. Encrypted archive with wrong passphrase should fail
	_, errWrong := svc.VerifyArchive(resEnc.FilePath, "wrong-passphrase")
	require.Error(t, errWrong)
}


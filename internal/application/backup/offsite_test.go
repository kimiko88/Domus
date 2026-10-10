package backup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/domus-hub/domus/internal/application/backup"
)

func TestExportOffsite(t *testing.T) {
	tempDir := t.TempDir()
	dummyBackupFile := filepath.Join(tempDir, "domus-20261010.tar.gz.enc")
	_ = os.WriteFile(dummyBackupFile, []byte("ENCRYPTED_BACKUP_CONTENT"), 0644)

	res := backup.CreateResult{
		FilePath:  dummyBackupFile,
		SizeBytes: int64(len("ENCRYPTED_BACKUP_CONTENT")),
		Encrypted: true,
	}

	svc := backup.NewService()
	ctx := context.Background()

	// 1. Test DestLocalPath (NAS / SMB mount)
	nasDir := filepath.Join(tempDir, "nas_mount")
	err := svc.ExportOffsite(ctx, res, backup.OffsiteDestination{
		Type: backup.DestLocalPath,
		Path: nasDir,
	})
	if err != nil {
		t.Fatalf("local path offsite failed: %v", err)
	}

	copiedFile := filepath.Join(nasDir, filepath.Base(dummyBackupFile))
	content, err := os.ReadFile(copiedFile)
	if err != nil || string(content) != "ENCRYPTED_BACKUP_CONTENT" {
		t.Errorf("file was not copied to nas mount properly")
	}

	// 2. Test DestWebDAV (Nextcloud mock)
	var mu sync.Mutex
	receivedBody := ""
	webdavServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			data, _ := os.ReadFile(dummyBackupFile)
			mu.Lock()
			receivedBody = string(data)
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer webdavServer.Close()

	err = svc.ExportOffsite(ctx, res, backup.OffsiteDestination{
		Type:     backup.DestWebDAV,
		Endpoint: webdavServer.URL,
		Path:     "/remote.php/dav/backups",
		Username: "admin",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("webdav offsite failed: %v", err)
	}

	mu.Lock()
	body := receivedBody
	mu.Unlock()
	if body != "ENCRYPTED_BACKUP_CONTENT" {
		t.Errorf("webdav did not receive content")
	}
}
